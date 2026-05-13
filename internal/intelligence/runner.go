package intelligence

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

	"detector-service/internal/intelligence/eval"
)

func (r *Runner) run(ctx context.Context, ds Dataset, req RunRequest, onEvent func(StreamEvent)) (*RunReport, error) {
	if err := validateRequest(req); err != nil {
		return nil, err
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
		ID:             fmt.Sprintf("%d", started.UnixNano()),
		DatasetName:    ds.Name(),
		DatasetVersion: ds.Version(),
		Source:         ds.Source(),
		Target:         strings.TrimRight(req.TargetBase, "/"),
		Model:          req.Model,
		Thinking:       req.Thinking,
		Effort:         req.Effort,
		ThinkingMode:   req.ThinkingMode,
		MaxTokens:      req.MaxTokens,
		StartedAt:      started,
		TaskTotal:      total,
		EvaluationNote: "Intelligence-test results recorded with automatic evaluation where reference answers are available.",
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
			answer, err := r.ask(ctx, req, t.Prompt)
			res.ElapsedMs = time.Since(oneStarted).Milliseconds()
			if err != nil {
				res.Error = err.Error()
				atomic.AddInt64(&errorCount, 1)
			} else {
				res.Answer = answer
				expected := t.ReferenceAnswer
				evalType := ""
				if t.Metadata != nil {
					evalType = t.Metadata["eval_type"]
				}
				er := eval.EvaluateTaskResult(answer, expected, evalType)
				if er.EvalType != "manual" {
					res.Pass = &er.Pass
					res.Score = &er.Score
					res.EvalType = er.EvalType
					res.JudgeReason = er.JudgeReason
				}
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

	var evalResults []eval.EvalResult
	categories := make(map[int]string)
	for _, r := range results {
		if r.Score != nil {
			evalResults = append(evalResults, eval.EvalResult{
				Pass:     *r.Pass,
				Score:    *r.Score,
				EvalType: r.EvalType,
			})
			categories[len(evalResults)-1] = r.Task.Category
		}
	}
	if len(evalResults) > 0 {
		agg := eval.Aggregate(evalResults, categories)
		report.ScoreTotal = &agg.ScoreTotal
		report.PassRate = &agg.PassRate
		report.TotalEvaluated = agg.TotalEvaluated
		report.TotalPassed = agg.TotalPassed
	}

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

func (r *Runner) ask(ctx context.Context, runReq RunRequest, prompt string) (string, error) {
	maxTok := 4096
	if runReq.MaxTokens > 0 {
		maxTok = runReq.MaxTokens
	}

	payload := map[string]any{
		"model":      runReq.Model,
		"max_tokens": maxTok,
		"messages": []any{
			map[string]any{"role": "user", "content": prompt},
		},
	}

	switch runReq.ThinkingMode {
	case "adaptive_only", "adaptive":
		payload["thinking"] = map[string]any{"type": "adaptive"}
	case "enabled":
		payload["thinking"] = map[string]any{"type": "enabled", "budget_tokens": 10000}
	default:
		if runReq.Thinking {
			if tp := deriveThinkingParam(runReq.Model); tp != nil {
				payload["thinking"] = tp
			}
		}
	}

	if runReq.Effort != "" {
		payload["output_config"] = map[string]any{"effort": runReq.Effort}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	url := strings.TrimRight(runReq.TargetBase, "/") + "/v1/messages"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("x-api-key", runReq.TargetKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := r.Client.Do(httpReq)
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

// deriveThinkingParam infers the correct thinking parameter from model name.
// Returns nil for models that don't support thinking (e.g. Haiku, unknown models).
func deriveThinkingParam(model string) map[string]any {
	m := strings.ToLower(model)
	switch {
	case strings.Contains(m, "haiku"):
		return nil
	case strings.Contains(m, "opus-4-7"):
		return map[string]any{"type": "adaptive"}
	case strings.Contains(m, "opus-4-6"), strings.Contains(m, "sonnet-4-6"):
		return map[string]any{"type": "adaptive"}
	case strings.Contains(m, "opus-4-5"):
		return map[string]any{"type": "enabled", "budget_tokens": 10000}
	case strings.Contains(m, "sonnet"), strings.Contains(m, "opus"):
		return map[string]any{"type": "adaptive"}
	default:
		return nil
	}
}
