package intelligence

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// HFLoader fetches public datasets from HuggingFace datasets-server API.
type HFLoader struct {
	Client *http.Client
}

func NewHFLoader() *HFLoader {
	return &HFLoader{Client: &http.Client{Timeout: 60 * time.Second}}
}

// hfRow is one row from the HF API.
type hfRow struct {
	Row map[string]any `json:"row"`
}

type hfResponse struct {
	Rows []hfRow `json:"rows"`
	Num  int     `json:"num_rows_total"`
}

// FetchRows fetches rows from a HuggingFace dataset via the datasets-server API.
// Returns raw rows as []map[string]any.
func (l *HFLoader) FetchRows(dataset, config, split string, limit int) ([]map[string]any, error) {
	if config == "" {
		config = "default"
	}
	if split == "" {
		split = "test"
	}

	var allRows []map[string]any
	batchSize := 100
	if limit > 0 && limit < batchSize {
		batchSize = limit
	}

	for offset := 0; ; offset += batchSize {
		if limit > 0 && offset >= limit {
			break
		}
		thisBatch := batchSize
		if limit > 0 && offset+thisBatch > limit {
			thisBatch = limit - offset
		}

		u := fmt.Sprintf("https://datasets-server.huggingface.co/rows?dataset=%s&config=%s&split=%s&offset=%d&length=%d",
			url.QueryEscape(dataset), url.QueryEscape(config), url.QueryEscape(split),
			offset, thisBatch)

		resp, err := l.Client.Get(u)
		if err != nil {
			return nil, fmt.Errorf("fetch HF rows: %w", err)
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
		resp.Body.Close()
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("HF API %d: %s", resp.StatusCode, truncateStr(string(body), 300))
		}
		if err != nil {
			return nil, err
		}

		var hf hfResponse
		if err := json.Unmarshal(body, &hf); err != nil {
			return nil, fmt.Errorf("parse HF response: %w", err)
		}

		for _, r := range hf.Rows {
			allRows = append(allRows, r.Row)
		}

		// Check if we got all rows
		total := hf.Num
		if len(allRows) >= total || len(hf.Rows) < thisBatch {
			break
		}
		if limit > 0 && len(allRows) >= limit {
			break
		}
	}
	return allRows, nil
}

// LoadDataset fetches and converts a HuggingFace dataset to a GenericDataset.
// adapter maps HF column names → Task fields.
func (l *HFLoader) LoadDataset(dataset, config, split string, limit int, adapter ColumnAdapter) (*GenericDataset, error) {
	rows, err := l.FetchRows(dataset, config, split, limit)
	if err != nil {
		return nil, err
	}

	tasks := make([]Task, 0, len(rows))
	for _, row := range rows {
		task := adapter.ToTask(row)
		if task.TaskID != "" && task.Prompt != "" {
			tasks = append(tasks, task)
		}
	}

	return &GenericDataset{
		DataName:    adapter.DatasetName(),
		DataVersion: adapter.DatasetVersion(),
		DataSource:  "https://huggingface.co/datasets/" + dataset,
		TaskList:    tasks,
	}, nil
}

// ColumnAdapter maps raw HF rows to generic Tasks.
type ColumnAdapter interface {
	DatasetName() string
	DatasetVersion() string
	ToTask(row map[string]any) Task
}

// ═══════════════════════════════════════════
// Built-in adapters for known intelligence datasets
// ═══════════════════════════════════════════

// SWEBenchAdapter maps SWE-bench Verified rows to Tasks.
type SWEBenchAdapter struct{}

func (SWEBenchAdapter) DatasetName() string    { return "SWE-bench-Verified" }
func (SWEBenchAdapter) DatasetVersion() string { return "hf-2026" }
func (SWEBenchAdapter) ToTask(row map[string]any) Task {
	return Task{
		TaskID:          strVal(row, "instance_id"),
		Prompt:          strVal(row, "problem_statement"),
		ReferenceAnswer: strVal(row, "patch"),
		Language:        "python",
		Category:        strVal(row, "difficulty"),
		Metadata: map[string]string{
			"repo":        strVal(row, "repo"),
			"base_commit": strVal(row, "base_commit"),
			"hints":       truncateStr(strVal(row, "hints_text"), 500),
			"version":     strVal(row, "version"),
		},
	}
}

// GPQAAdapter maps GPQA Diamond CSV rows to Tasks.
type GPQAAdapter struct{}

func (GPQAAdapter) DatasetName() string    { return "GPQA-Diamond" }
func (GPQAAdapter) DatasetVersion() string { return "v1" }
func (GPQAAdapter) ToTask(row map[string]any) Task {
	question := strVal(row, "Question")
	correct := strVal(row, "Correct Answer")
	wrong1 := strVal(row, "Incorrect Answer 1")
	wrong2 := strVal(row, "Incorrect Answer 2")
	wrong3 := strVal(row, "Incorrect Answer 3")

	// Stable deterministic shuffle based on question hash to avoid position bias.
	type choice struct {
		text    string
		correct bool
	}
	all := []choice{
		{correct, true}, {wrong1, false}, {wrong2, false}, {wrong3, false},
	}
	h := hashStr(question)
	// Fisher-Yates with deterministic seed
	for i := len(all) - 1; i > 0; i-- {
		j := int(h) % (i + 1)
		all[i], all[j] = all[j], all[i]
		h = h*2654435761 + 1
	}

	labels := []string{"A", "B", "C", "D"}
	var prompt strings.Builder
	prompt.WriteString(question)
	prompt.WriteString("\n\nChoices:\n")
	correctLabel := "A"
	for i, c := range all {
		if c.text != "" {
			prompt.WriteString(fmt.Sprintf("%s. %s\n", labels[i], c.text))
		}
		if c.correct {
			correctLabel = labels[i]
		}
	}
	prompt.WriteString("\nAnswer with just the letter (A, B, C, or D).")

	subdomain := strVal(row, "Subdomain")
	if subdomain == "" {
		subdomain = strVal(row, "High-level domain")
	}

	taskID := strVal(row, "Record ID")
	if taskID == "" {
		taskID = fmt.Sprintf("gpqa_%x", hashStr(question))
	}

	return Task{
		TaskID:          taskID,
		Prompt:          prompt.String(),
		ReferenceAnswer: correctLabel,
		Category:        subdomain,
		Language:        strVal(row, "High-level domain"),
	}
}

// HLEAdapter maps Humanity's Last Exam rows to Tasks.
type HLEAdapter struct{}

func (HLEAdapter) DatasetName() string    { return "HLE" }
func (HLEAdapter) DatasetVersion() string { return "v1" }
func (HLEAdapter) ToTask(row map[string]any) Task {
	question := strVal(row, "question")
	answer := strVal(row, "answer")
	category := strVal(row, "field")
	if category == "" {
		category = strVal(row, "subfield")
	}

	prompt := question
	// If multiple choice, format options
	if opts := strVal(row, "options"); opts != "" {
		prompt += "\n\nOptions: " + opts
		prompt += "\n\nAnswer with just the letter."
	}

	taskID := strVal(row, "id")
	if taskID == "" {
		taskID = fmt.Sprintf("hle_%x", hashStr(question))
	}

	return Task{
		TaskID:          taskID,
		Prompt:          prompt,
		ReferenceAnswer: answer,
		Category:        category,
		Language:        strVal(row, "field"),
		Metadata: map[string]string{
			"question_type": strVal(row, "question_type"),
			"subfield":      strVal(row, "subfield"),
		},
	}
}

// LiveCodeBenchAdapter maps LiveCodeBench rows to Tasks.
type LiveCodeBenchAdapter struct{}

func (LiveCodeBenchAdapter) DatasetName() string    { return "LiveCodeBench" }
func (LiveCodeBenchAdapter) DatasetVersion() string { return "v5" }
func (LiveCodeBenchAdapter) ToTask(row map[string]any) Task {
	taskID := strVal(row, "question_id")
	if taskID == "" {
		taskID = strVal(row, "id")
	}
	prompt := strVal(row, "question_content")
	if prompt == "" {
		prompt = strVal(row, "question")
	}
	if prompt == "" {
		prompt = strVal(row, "problem_description")
	}
	lang := strVal(row, "language")
	if lang == "" {
		lang = "python"
	}
	difficulty := strVal(row, "difficulty")
	if difficulty == "" {
		difficulty = strVal(row, "contest_difficulty")
	}
	return Task{
		TaskID:   taskID,
		Prompt:   prompt,
		Category: difficulty,
		Language: lang,
		Metadata: map[string]string{
			"platform":   strVal(row, "platform"),
			"contest_id": strVal(row, "contest_id"),
		},
	}
}

// ═══════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════

func strVal(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case json.Number:
		return t.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func hashStr(s string) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}
