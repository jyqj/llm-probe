package probe

import (
	"fmt"
	"sync"
	"time"
)

type probeTarget struct {
	baseURL string
	apiKey  string
	model   string
}

type probeCase struct {
	Name     string
	Phase    string
	Required bool
	Run      func() ([]CheckResult, error)
}

// RunSuite runs the probe suite against a target.
func (p *Prober) RunSuite(targetBase, targetKey, model string, quick bool) (*ProbeReport, error) {
	startedAt := time.Now()
	target := probeTarget{baseURL: targetBase, apiKey: targetKey, model: model}

	checks, err := p.runRequiredProbeCases(p.requiredProbeCases(target))
	if err != nil {
		return nil, err
	}

	if !quick {
		checks = append(checks, p.runOptionalProbeCases(p.optionalProbeCases(target))...)
	}

	mode := "full"
	if quick {
		mode = "quick"
	}

	report := &ProbeReport{
		Target:    targetBase,
		Model:     model,
		Timestamp: time.Now(),
		ElapsedMs: time.Since(startedAt).Milliseconds(),
		Checks:    checks,
	}
	report.Recommended = RecommendFixes(checks)
	report.Score = CalculateScore(checks, mode)
	report.Summary = BuildSummaryWithScore(checks, report.Score)
	return report, nil
}

func (p *Prober) requiredProbeCases(t probeTarget) []probeCase {
	return []probeCase{
		{
			Name:     "precheck",
			Phase:    "baseline_stream",
			Required: true,
			Run:      func() ([]CheckResult, error) { return p.runPrecheck(t.baseURL, t.apiKey, t.model) },
		},
		{
			Name:     "tag_replay",
			Phase:    "full_fingerprint",
			Required: true,
			Run:      func() ([]CheckResult, error) { return p.runTagReplay(t.baseURL, t.apiKey, t.model) },
		},
	}
}

func (p *Prober) optionalProbeCases(t probeTarget) []probeCase {
	return []probeCase{
		{Name: "mini_probe", Phase: "nonstream_max_token", Run: func() ([]CheckResult, error) { return p.runMiniProbe(t.baseURL, t.apiKey, t.model) }},
		{Name: "identity_probe", Phase: "identity_nonstream", Run: func() ([]CheckResult, error) { return p.runIdentityProbe(t.baseURL, t.apiKey, t.model) }},
		{Name: "self_intro", Phase: "structured_stream", Run: func() ([]CheckResult, error) { return p.runSelfIntroProbe(t.baseURL, t.apiKey, t.model) }},
		{Name: "tool_use", Phase: "tool_stream", Run: func() ([]CheckResult, error) { return p.runToolUseProbe(t.baseURL, t.apiKey, t.model) }},
		{Name: "logic", Phase: "reasoning_stream", Run: func() ([]CheckResult, error) { return p.runLogicProbe(t.baseURL, t.apiKey, t.model) }},
		{Name: "hidden_prompt", Phase: "token_analysis", Run: func() ([]CheckResult, error) { return p.runHiddenPrompt(t.baseURL, t.apiKey, t.model) }},
		{Name: "image_ocr", Phase: "multimodal_image", Run: func() ([]CheckResult, error) { return p.runImageOCR(t.baseURL, t.apiKey, t.model) }},
		{Name: "pdf_extract", Phase: "multimodal_pdf", Run: func() ([]CheckResult, error) { return p.runPDFExtract(t.baseURL, t.apiKey, t.model) }},
	}
}

func (p *Prober) runRequiredProbeCases(cases []probeCase) ([]CheckResult, error) {
	var checks []CheckResult
	for _, c := range cases {
		result, err := c.Run()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", c.Name, err)
		}
		checks = append(checks, result...)
	}
	return checks, nil
}

func (p *Prober) runOptionalProbeCases(cases []probeCase) []CheckResult {
	slots := make([][]CheckResult, len(cases))
	var wg sync.WaitGroup
	wg.Add(len(cases))
	for i, c := range cases {
		go func(idx int, probe probeCase) {
			defer wg.Done()
			slots[idx] = p.safeProbeCase(probe)
		}(i, c)
	}
	wg.Wait()

	var checks []CheckResult
	for _, slot := range slots {
		checks = append(checks, slot...)
	}
	return checks
}

func (p *Prober) safeProbeCase(c probeCase) []CheckResult {
	checks, err := c.Run()
	if err != nil {
		return []CheckResult{{Name: c.Name, Pass: false, Detail: err.Error()}}
	}
	return checks
}
