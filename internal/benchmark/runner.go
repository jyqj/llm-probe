package benchmark

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const defaultMaxTokens = 4096

func (r *Runner) run(ctx context.Context, ds Dataset, req RunRequest, onEvent func(StreamEvent)) (*RunReport, error) {
	if err := validateRequest(req); err != nil {
		return nil, err
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}
	concurrency := req.Concurrency
	if concurrency <= 0 {
		concurrency = 5
	}

	tasks := ds.Filter(Filter{
		Language: req.Language,
		Category: req.Category,
		TaskIDs:  req.TaskIDs,
		Limit:    req.Limit,
	})
	started := time.Now()
	total := len(tasks)

	report := &RunReport{
		DatasetName:    ds.Name(),
		DatasetVersion: ds.Version(),
		Source:         ds.Source(),
		Target:         strings.TrimRight(req.TargetBase, "/"),
		Model:          req.Model,
		StartedAt:      started,
		TaskTotal:      total,
		EvaluationNote: "Benchmark results recorded for offline/judge-based scoring.",
	}
	if total == 0 {
		report.ElapsedMs = time.Since(started).Milliseconds()
		return report, nil
	}

	results := make([]TaskRunResult, total)
	var completed int64
	var errorCount int64

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t Task) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			res := TaskRunResult{Index: idx, Task: t.Summary(false), Rubric: t.Rubric}
			oneStarted := time.Now()
			answer, err := r.ask(ctx, req.TargetBase, req.TargetKey, req.Model, maxTokens, t.Prompt)
			res.ElapsedMs = time.Since(oneStarted).Milliseconds()
			if err != nil {
				res.Error = err.Error()
				atomic.AddInt64(&errorCount, 1)
			} else {
				res.Answer = answer
			}
			results[idx] = res

			c := int(atomic.AddInt64(&completed, 1))
			e := int(atomic.LoadInt64(&errorCount))
			if onEvent != nil {
				onEvent(StreamEvent{
					Type:      "progress",
					Index:     idx,
					Total:     total,
					Completed: c,
					Errors:    e,
					Result:    &res,
				})
			}
		}(i, task)
	}
	wg.Wait()

	report.Results = results
	report.TaskCompleted = int(completed)
	report.TaskErrors = int(errorCount)
	report.ElapsedMs = time.Since(started).Milliseconds()

	if onEvent != nil {
		onEvent(StreamEvent{
			Type:      "complete",
			Total:     total,
			Completed: int(completed),
			Errors:    int(errorCount),
			Report:    report,
		})
	}
	return report, nil
}

func (r *Runner) ask(ctx context.Context, targetBase, targetKey, model string, maxTokens int, prompt string) (string, error) {
	payload := map[string]any{
		"model":      model,
		"max_tokens": maxTokens,
		"messages": []any{
			map[string]any{"role": "user", "content": prompt},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	url := strings.TrimRight(targetBase, "/") + "/v1/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-api-key", targetKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := r.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("target returned HTTP %d: %s", resp.StatusCode, truncateStr(string(data), 512))
	}
	return extractText(data), nil
}

func validateRequest(req RunRequest) error {
	if strings.TrimSpace(req.TargetBase) == "" {
		return fmt.Errorf("target_base is required")
	}
	if strings.TrimSpace(req.TargetKey) == "" {
		return fmt.Errorf("target_key is required")
	}
	if strings.TrimSpace(req.Model) == "" {
		return fmt.Errorf("model is required")
	}
	return nil
}

func extractText(data []byte) string {
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return string(data)
	}
	var parts []string
	if content, ok := root["content"].([]any); ok {
		for _, item := range content {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if text, ok := m["text"].(string); ok && text != "" {
				parts = append(parts, text)
			}
		}
	}
	if len(parts) > 0 {
		return strings.Join(parts, "\n")
	}
	return string(data)
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
