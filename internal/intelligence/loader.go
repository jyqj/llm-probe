package intelligence

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// GenericDataset is a Dataset backed by an in-memory task list.
// It can be created from CSV, JSON, or programmatically.
type GenericDataset struct {
	DataName    string `json:"name"`
	DataVersion string `json:"version"`
	DataSource  string `json:"source,omitempty"`
	TaskList    []Task `json:"tasks"`
}

func (d *GenericDataset) Name() string    { return d.DataName }
func (d *GenericDataset) Version() string { return d.DataVersion }
func (d *GenericDataset) Source() string  { return d.DataSource }

func (d *GenericDataset) Tasks() []Task { return d.TaskList }

func (d *GenericDataset) Stats() Stats {
	stats := Stats{
		Name:       d.DataName,
		Version:    d.DataVersion,
		Source:     d.DataSource,
		TotalTasks: len(d.TaskList),
		Languages:  map[string]int{},
		Categories: map[string]int{},
	}
	for _, task := range d.TaskList {
		if task.Language != "" {
			stats.Languages[task.Language]++
		}
		if task.Category != "" {
			stats.Categories[task.Category]++
		}
	}
	return stats
}

func (d *GenericDataset) Filter(f Filter) []Task {
	ids := map[string]bool{}
	for _, id := range f.TaskIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			ids[id] = true
		}
	}
	limit := f.Limit
	if limit <= 0 || limit > len(d.TaskList) {
		limit = len(d.TaskList)
	}
	out := make([]Task, 0, limit)
	for _, task := range d.TaskList {
		if len(ids) > 0 && !ids[task.TaskID] {
			continue
		}
		if f.Language != "" && !strings.EqualFold(task.Language, f.Language) {
			continue
		}
		if f.Category != "" && !strings.EqualFold(task.Category, f.Category) {
			continue
		}
		out = append(out, task)
		if !f.ImportantFirst && len(out) >= limit {
			break
		}
	}
	if f.ImportantFirst {
		SortImportantFirst(out)
		if len(out) > limit {
			out = out[:limit]
		}
	}
	return out
}

func (d *GenericDataset) Find(taskID string) (Task, bool) {
	for _, task := range d.TaskList {
		if task.TaskID == taskID {
			return task, true
		}
	}
	return Task{}, false
}

// Summaries returns TaskSummary list from tasks.
func Summaries(tasks []Task, includeRubric bool) []TaskSummary {
	out := make([]TaskSummary, 0, len(tasks))
	for _, task := range tasks {
		out = append(out, task.Summary(includeRubric))
	}
	return out
}

// UniqueSorted returns sorted keys from a map.
func UniqueSorted(values map[string]int) []string {
	out := make([]string, 0, len(values))
	for v := range values {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

// SortImportantFirst orders tasks by rubric importance while preserving source
// order for equal priority. It is used by continuous benchmark monitoring so
// a low-frequency run covers the highest-signal tasks first.
func SortImportantFirst(tasks []Task) {
	sort.SliceStable(tasks, func(i, j int) bool {
		return taskImportanceScore(tasks[i]) > taskImportanceScore(tasks[j])
	})
}

func taskImportanceScore(t Task) int {
	score := 0
	for _, r := range t.Rubric {
		switch strings.ToLower(strings.TrimSpace(r.Annotations.Importance)) {
		case "must have", "must-have", "critical", "required":
			score += 100
		case "should have", "should-have", "high", "important":
			score += 50
		case "nice to have", "nice-to-have", "medium":
			score += 10
		case "low", "optional":
			score++
		}
	}
	if t.ReferenceAnswer != "" {
		score += 5
	}
	return score
}

// ── Loaders ──

// LoadCSV loads a dataset from a CSV reader.
// Required columns: task_id, prompt. Optional: reference_answer, language, category, rubric, plus any others as metadata.
func LoadCSV(r io.Reader, name, version, source string) (*GenericDataset, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	cr.ReuseRecord = false

	header, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("read csv header: %w", err)
	}
	cols := map[string]int{}
	for i, h := range header {
		cols[strings.TrimSpace(h)] = i
	}
	if _, ok := cols["task_id"]; !ok {
		return nil, fmt.Errorf("missing required column 'task_id'")
	}
	if _, ok := cols["prompt"]; !ok {
		return nil, fmt.Errorf("missing required column 'prompt'")
	}

	knownCols := map[string]bool{
		"task_id": true, "prompt": true, "reference_answer": true,
		"language": true, "category": true, "rubric": true,
	}

	var tasks []Task
	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read csv record: %w", err)
		}
		get := func(colName string) string {
			i, ok := cols[colName]
			if !ok || i >= len(rec) {
				return ""
			}
			return rec[i]
		}

		var rubric []RubricItem
		if raw := strings.TrimSpace(get("rubric")); raw != "" {
			_ = json.Unmarshal([]byte(raw), &rubric)
		}

		// Collect unknown columns as metadata
		var meta map[string]string
		for colName, idx := range cols {
			if knownCols[colName] || idx >= len(rec) {
				continue
			}
			val := strings.TrimSpace(rec[idx])
			if val == "" {
				continue
			}
			if meta == nil {
				meta = make(map[string]string)
			}
			meta[colName] = val
		}

		tasks = append(tasks, Task{
			TaskID:          get("task_id"),
			Prompt:          get("prompt"),
			ReferenceAnswer: get("reference_answer"),
			Language:        get("language"),
			Category:        get("category"),
			Rubric:          rubric,
			Metadata:        meta,
		})
	}

	return &GenericDataset{
		DataName:    name,
		DataVersion: version,
		DataSource:  source,
		TaskList:    tasks,
	}, nil
}

// LoadJSON loads a dataset from a JSON reader.
// Expected format: {"name":"...", "version":"...", "tasks": [...]}
// or just an array of tasks: [{"task_id":"...", "prompt":"...", ...}, ...]
func LoadJSON(r io.Reader, fallbackName, fallbackVersion, fallbackSource string) (*GenericDataset, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read json: %w", err)
	}

	// Try full format first
	var full struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Source  string `json:"source"`
		Tasks   []Task `json:"tasks"`
	}
	if err := json.Unmarshal(data, &full); err == nil && len(full.Tasks) > 0 {
		name := full.Name
		if name == "" {
			name = fallbackName
		}
		ver := full.Version
		if ver == "" {
			ver = fallbackVersion
		}
		src := full.Source
		if src == "" {
			src = fallbackSource
		}
		return &GenericDataset{DataName: name, DataVersion: ver, DataSource: src, TaskList: full.Tasks}, nil
	}

	// Try array format
	var tasks []Task
	if err := json.Unmarshal(data, &tasks); err == nil && len(tasks) > 0 {
		return &GenericDataset{
			DataName:    fallbackName,
			DataVersion: fallbackVersion,
			DataSource:  fallbackSource,
			TaskList:    tasks,
		}, nil
	}

	return nil, fmt.Errorf("cannot parse JSON as dataset or task array")
}
