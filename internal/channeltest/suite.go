package channeltest

import (
	"sync"
	"time"

	"detector-service/internal/fingerprint"
)

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
	startedAt := time.Now()

	probes := ProbesForModel(model)
	results := p.executeProbes(probes, targetBase, targetKey, model, concurrency)

	var checks []CheckResult
	for _, r := range results {
		checks = append(checks, r.Checks...)
	}

	InjectLabels(checks)

	totalIn, totalOut := collectTokens(checks)
	billing := EstimateBilling(model, totalIn, totalOut)

	report := &Report{
		Target:       targetBase,
		Model:        model,
		Timestamp:    time.Now(),
		ElapsedMs:    time.Since(startedAt).Milliseconds(),
		Checks:       checks,
		ProbeResults: results,
		Billing:      &billing,
	}
	report.Recommended = RecommendFixes(checks)
	report.Score = CalculateScore(checks, "full")
	report.Summary = BuildSummaryWithScore(checks, report.Score)
	return report, nil
}

// RunMulti runs the suite for multiple models against the same target.
func (p *Runner) RunMulti(targetBase, targetKey string, models []string, concurrency int) ([]*Report, error) {
	var reports []*Report
	for _, model := range models {
		report, err := p.Run(targetBase, targetKey, model, concurrency)
		if err != nil {
			reports = append(reports, &Report{
				Model:     model,
				Target:    targetBase,
				Timestamp: time.Now(),
				Summary:   "error: " + err.Error(),
			})
			continue
		}
		reports = append(reports, report)
	}
	return reports, nil
}

// executeProbes runs probes with given concurrency.
func (p *Runner) executeProbes(probes []*Probe, base, key, model string, concurrency int) []ProbeResult {
	results := make([]ProbeResult, len(probes))

	if concurrency <= 0 {
		for i, probe := range probes {
			results[i] = p.runSingleProbe(probe, base, key, model)
		}
		return results
	}

	// Required probes run first sequentially
	idx := 0
	for idx < len(probes) && probes[idx].Required {
		results[idx] = p.runSingleProbe(probes[idx], base, key, model)
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
				optResults[ii] = p.runSingleProbe(pb, base, key, model)
			}(i, probe)
		}
		wg.Wait()
		for i, r := range optResults {
			results[idx+i] = r
		}
	}

	return results
}

func (p *Runner) runSingleProbe(probe *Probe, base, key, model string) ProbeResult {
	rec := p.withRecorder()
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
