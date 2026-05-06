package intelligence

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
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
