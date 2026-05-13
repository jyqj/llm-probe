package channeltest

import (
	"context"
	"fmt"
	"sync"
	"time"

	"detector-service/internal/fingerprint"
)

// ProbeInfo is a lightweight probe descriptor sent in the start event.
type ProbeInfo struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// StreamEvent is emitted during a streaming channel test run.
// Field names match the frontend SSE protocol in channel.js.
type StreamEvent struct {
	Type        string                  `json:"type"`                   // start, probe_start, probe_done, model_done, done, error
	RunID       string                  `json:"run_id,omitempty"`
	Model       string                  `json:"model,omitempty"`
	Models      []string                `json:"models,omitempty"`
	TotalProbes int                     `json:"total_probes,omitempty"`
	ModelProbes map[string][]ProbeInfo  `json:"model_probes,omitempty"`
	ProbeID     string                  `json:"probe_id,omitempty"`
	Label       string                  `json:"label,omitempty"`
	Probe       *ProbeResult            `json:"probe,omitempty"`
	Report      *Report                 `json:"report,omitempty"`
	Reports     []*Report               `json:"reports,omitempty"`
	Error       string                  `json:"error,omitempty"`
}

// ProbeResult tracks a single probe's execution.
type ProbeResult struct {
	ProbeID   string        `json:"probe_id"`
	Label     string        `json:"label"`
	LatencyMs int64         `json:"latency_ms"`
	Checks    []CheckResult `json:"checks"`
	Exchanges []Exchange    `json:"exchanges,omitempty"`
}

// ModelReport is the result for a single model within a multi-model run.
type ModelReport struct {
	Model        string         `json:"model"`
	Checks       []CheckResult  `json:"checks"`
	ProbeResults []ProbeResult  `json:"probe_results"`
	Score        *ScoreReport   `json:"score,omitempty"`
	Billing      *BillingEstimate `json:"billing,omitempty"`
	AvgLatencyMs int64          `json:"avg_latency_ms"`
}

// Run runs the channel test suite against a target for a single model.
// concurrency=0 means sequential.
func (p *Runner) Run(targetBase, targetKey, model string, concurrency int) (*Report, error) {
	return p.RunCtx(context.Background(), targetBase, targetKey, model, concurrency)
}

// RunCtx runs the suite with context support for cancellation.
func (p *Runner) RunCtx(ctx context.Context, targetBase, targetKey, model string, concurrency int) (*Report, error) {
	return p.runInternal(ctx, targetBase, targetKey, model, concurrency, nil)
}

// RunStream runs the suite with streaming events via onEvent callback.
func (p *Runner) RunStream(ctx context.Context, targetBase, targetKey, model string, concurrency int, onEvent func(StreamEvent)) (*Report, error) {
	return p.runInternal(ctx, targetBase, targetKey, model, concurrency, onEvent)
}

func (p *Runner) runInternal(ctx context.Context, targetBase, targetKey, model string, concurrency int, onEvent func(StreamEvent)) (*Report, error) {
	startedAt := time.Now()

	probes := ProbesForModel(model)
	results := p.executeProbesCtx(ctx, probes, targetBase, targetKey, model, concurrency, onEvent)

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var checks []CheckResult
	for _, r := range results {
		checks = append(checks, r.Checks...)
	}

	InjectLabels(checks)

	billing := EstimateRunCost(model, probes)

	selectedIDs := make(map[string]bool, len(probes))
	for _, pb := range probes {
		selectedIDs[pb.ID] = true
	}
	var skippedProbes []string
	for _, pb := range allProbes {
		if !selectedIDs[pb.ID] {
			skippedProbes = append(skippedProbes, pb.ID)
		}
	}

	report := &Report{
		Target:        targetBase,
		Model:         model,
		Timestamp:     time.Now(),
		ElapsedMs:     time.Since(startedAt).Milliseconds(),
		Checks:        checks,
		ProbeResults:  results,
		Billing:       &billing,
		SkippedProbes: skippedProbes,
	}
	report.Recommended = RecommendFixes(checks)
	report.Score = CalculateScore(checks, "full")
	report.Summary = BuildSummaryWithScore(checks, report.Score)
	report.RunProfile = fmt.Sprintf("Ran %d probes (%d checks), skipped %d. Model: %s. Est cost: $%.4f. Elapsed: %dms.",
		len(probes), len(checks), len(skippedProbes), model, billing.TotalCost, report.ElapsedMs)
	return report, nil
}

// RunMulti runs the suite for multiple models against the same target.
func (p *Runner) RunMulti(targetBase, targetKey string, models []string, concurrency int) ([]*Report, error) {
	return p.RunMultiCtx(context.Background(), targetBase, targetKey, models, concurrency)
}

// RunMultiCtx runs for multiple models with context support.
func (p *Runner) RunMultiCtx(ctx context.Context, targetBase, targetKey string, models []string, concurrency int) ([]*Report, error) {
	return p.runMultiInternal(ctx, targetBase, targetKey, models, concurrency, nil)
}

// RunMultiStream runs for multiple models with streaming events.
func (p *Runner) RunMultiStream(ctx context.Context, targetBase, targetKey string, models []string, concurrency int, onEvent func(StreamEvent)) ([]*Report, error) {
	return p.runMultiInternal(ctx, targetBase, targetKey, models, concurrency, onEvent)
}

func (p *Runner) runMultiInternal(ctx context.Context, targetBase, targetKey string, models []string, concurrency int, onEvent func(StreamEvent)) ([]*Report, error) {
	var reports []*Report
	for _, model := range models {
		if err := ctx.Err(); err != nil {
			return reports, err
		}
		report, err := p.runInternal(ctx, targetBase, targetKey, model, concurrency, onEvent)
		if err != nil {
			if ctx.Err() != nil {
				return reports, ctx.Err()
			}
			reports = append(reports, &Report{
				Model:     model,
				Target:    targetBase,
				Timestamp: time.Now(),
				Summary:   "error: " + err.Error(),
			})
			if onEvent != nil {
				onEvent(StreamEvent{Type: "model_done", Model: model, Report: reports[len(reports)-1]})
			}
			continue
		}
		reports = append(reports, report)
		if onEvent != nil {
			onEvent(StreamEvent{Type: "model_done", Model: model, Report: report})
		}
	}
	return reports, nil
}

// executeProbes runs probes with given concurrency (legacy, no context).
func (p *Runner) executeProbes(probes []*Probe, base, key, model string, concurrency int) []ProbeResult {
	return p.executeProbesCtx(context.Background(), probes, base, key, model, concurrency, nil)
}

// executeProbesCtx runs probes with context + optional event streaming.
func (p *Runner) executeProbesCtx(ctx context.Context, probes []*Probe, base, key, model string, concurrency int, onEvent func(StreamEvent)) []ProbeResult {
	results := make([]ProbeResult, len(probes))

	emit := func(ev StreamEvent) {
		if onEvent != nil {
			ev.Model = model
			onEvent(ev)
		}
	}

	runOne := func(i int, probe *Probe) {
		if ctx.Err() != nil {
			results[i] = ProbeResult{
				ProbeID: probe.ID, Label: probe.Label,
				Checks: []CheckResult{{Name: probe.ID, Pass: false, Detail: "cancelled"}},
			}
			return
		}
		emit(StreamEvent{Type: "probe_start", ProbeID: probe.ID, Label: probe.Label})
		results[i] = p.runSingleProbe(ctx, probe, base, key, model)
		emit(StreamEvent{Type: "probe_done", ProbeID: probe.ID, Probe: &results[i]})
	}

	if concurrency <= 0 {
		for i, probe := range probes {
			runOne(i, probe)
		}
		return results
	}

	// Required probes run first sequentially
	idx := 0
	for idx < len(probes) && probes[idx].Required {
		runOne(idx, probes[idx])
		idx++
	}

	// Optional probes run with concurrency
	if idx < len(probes) {
		optional := probes[idx:]
		optResults := make([]ProbeResult, len(optional))
		sem := make(chan struct{}, concurrency)
		var wg sync.WaitGroup
		wg.Add(len(optional))
		for i, probe := range optional {
			go func(ii int, pb *Probe) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				if ctx.Err() != nil {
					optResults[ii] = ProbeResult{
						ProbeID: pb.ID, Label: pb.Label,
						Checks: []CheckResult{{Name: pb.ID, Pass: false, Detail: "cancelled"}},
					}
					return
				}
				emit(StreamEvent{Type: "probe_start", ProbeID: pb.ID, Label: pb.Label})
				optResults[ii] = p.runSingleProbe(ctx, pb, base, key, model)
				emit(StreamEvent{Type: "probe_done", ProbeID: pb.ID, Probe: &optResults[ii]})
			}(i, probe)
		}
		wg.Wait()
		for i, r := range optResults {
			results[idx+i] = r
		}
	}

	return results
}

func (p *Runner) runSingleProbe(ctx context.Context, probe *Probe, base, key, model string) ProbeResult {
	rec := p.withRecorder()
	rec.ctx = ctx
	start := time.Now()
	checks, err := probe.Run(rec, base, key, model)
	elapsed := time.Since(start).Milliseconds()

	if err != nil {
		checks = []CheckResult{{Name: probe.ID, Pass: false, Detail: err.Error()}}
	}

	return ProbeResult{
		ProbeID:   probe.ID,
		Label:     probe.Label,
		LatencyMs: elapsed,
		Checks:    checks,
		Exchanges: rec.getExchanges(),
	}
}

// collectTokens sums input/output tokens from check results (from usage fields).
func collectTokens(checks []CheckResult) (int, int) {
	totalIn, totalOut := 0, 0
	for _, c := range checks {
		if c.Name == "message_start_usage" || c.Name == "minimal_input_tokens" {
			// Parse from detail or actual field if available
		}
	}
	return totalIn, totalOut
}

// RunMonitor runs only monitor-tagged probes (lightweight periodic check).
func (p *Runner) RunMonitor(targetBase, targetKey, model string) (*Report, error) {
	startedAt := time.Now()

	probes := FilterProbes("monitor")
	results := p.executeProbes(probes, targetBase, targetKey, model, 0)

	var checks []CheckResult
	for _, r := range results {
		checks = append(checks, r.Checks...)
	}
	InjectLabels(checks)

	report := &Report{
		Target:       targetBase,
		Model:        model,
		Timestamp:    time.Now(),
		ElapsedMs:    time.Since(startedAt).Milliseconds(),
		Checks:       checks,
		ProbeResults: results,
	}
	report.Recommended = RecommendFixes(checks)
	report.Score = CalculateScore(checks, "monitor")
	report.Summary = BuildSummaryWithScore(checks, report.Score)
	return report, nil
}

// collectAvgLatency computes average probe latency from results.
func collectAvgLatency(results []ProbeResult) int64 {
	if len(results) == 0 {
		return 0
	}
	var total int64
	for _, r := range results {
		total += r.LatencyMs
	}
	return total / int64(len(results))
}

// collectTotalUsage extracts total input/output tokens from response bodies.
func collectTotalUsage(checks []CheckResult) (int, int) {
	// Token counts are embedded in check details — for billing estimation,
	// we use probe EstTokens as the primary source.
	return 0, 0
}

// EstimateRunCost estimates the total cost for running selected probes against a model.
func EstimateRunCost(model string, probes []*Probe) BillingEstimate {
	caps := GetModelCaps(model)
	totalIn := 0
	for _, p := range probes {
		totalIn += p.EstTokens
	}
	estOut := totalIn / 5
	inCost := float64(totalIn) / 1_000_000 * caps.InputPrice
	outCost := float64(estOut) / 1_000_000 * caps.OutputPrice
	return BillingEstimate{
		InputTokens:  totalIn,
		OutputTokens: estOut,
		InputCost:    round4(inCost),
		OutputCost:   round4(outCost),
		TotalCost:    round4(inCost + outCost),
		PriceRatio:   round2(caps.InputPrice / 3.0),
		Source:       "estimated",
	}
}

// collectUsageFromResponse extracts input/output tokens from a parsed API response.
func collectUsageFromResponse(body map[string]any) (int, int) {
	usage, _ := body["usage"].(map[string]any)
	if usage == nil {
		return 0, 0
	}
	return fingerprint.IntVal(usage, "input_tokens"), fingerprint.IntVal(usage, "output_tokens")
}
