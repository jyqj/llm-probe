package channeltest

import (
	"fmt"
	"sync"
	"time"
)

type targetSpec struct {
	baseURL string
	apiKey  string
	model   string
}

type caseSpec struct {
	Name     string
	Phase    string
	Required bool
	Run      func() ([]CheckResult, error)
}

// Run runs the channel test suite against a target.
func (p *Runner) Run(targetBase, targetKey, model string, quick bool) (*Report, error) {
	startedAt := time.Now()
	target := targetSpec{baseURL: targetBase, apiKey: targetKey, model: model}

	checks, err := p.runRequiredCases(p.requiredCases(target))
	if err != nil {
		return nil, err
	}

	if !quick {
		checks = append(checks, p.runOptionalCases(p.optionalCases(target))...)
	}

	mode := "full"
	if quick {
		mode = "quick"
	}

	report := &Report{
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

func (p *Runner) requiredCases(t targetSpec) []caseSpec {
	return []caseSpec{
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

func (p *Runner) optionalCases(t targetSpec) []caseSpec {
	return []caseSpec{
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

func (p *Runner) runRequiredCases(cases []caseSpec) ([]CheckResult, error) {
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

func (p *Runner) runOptionalCases(cases []caseSpec) []CheckResult {
	slots := make([][]CheckResult, len(cases))
	var wg sync.WaitGroup
	wg.Add(len(cases))
	for i, c := range cases {
		go func(idx int, c caseSpec) {
			defer wg.Done()
			slots[idx] = p.safeCase(c)
		}(i, c)
	}
	wg.Wait()

	var checks []CheckResult
	for _, slot := range slots {
		checks = append(checks, slot...)
	}
	return checks
}

func (p *Runner) safeCase(c caseSpec) []CheckResult {
	checks, err := c.Run()
	if err != nil {
		return []CheckResult{{Name: c.Name, Pass: false, Detail: err.Error()}}
	}
	return checks
}
