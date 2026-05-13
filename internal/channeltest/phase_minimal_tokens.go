package channeltest

import (
	"fmt"

	"detector-service/internal/fingerprint"
)

var probeMinimalTokens = &Probe{
	ID: "minimal_tokens", Label: "最小 Token 计费探针",
	Tags:      []string{"monitor"},
	EstTokens: 20,
	Checks:    []string{"minimal_input_tokens", "minimal_output_tokens"},
	Run:       (*Runner).runMinimalTokens,
}

func (p *Runner) runMinimalTokens(targetBase, targetKey, model string) ([]CheckResult, error) {
	body := toJSON(map[string]any{
		"model":      model,
		"max_tokens": 4,
		"stream":     false,
		"messages":   []any{umsg("hi")},
	})

	j, err := p.sendReadJSON(targetBase, targetKey, body)
	if err != nil {
		return nil, err
	}

	var checks []CheckResult
	usage, _ := j["usage"].(map[string]any)
	if usage == nil {
		return []CheckResult{
			{Name: "minimal_input_tokens", Pass: false, Expected: "8-12", Actual: "无 usage", Detail: "no usage object"},
			{Name: "minimal_output_tokens", Pass: false, Expected: "1-4", Actual: "无 usage", Detail: "no usage object"},
		}, nil
	}

	inTok := fingerprint.IntVal(usage, "input_tokens")
	inActual := fmt.Sprintf("%d", inTok)
	if inTok >= 8 && inTok <= 16 {
		checks = append(checks, CheckResult{
			Name: "minimal_input_tokens", Pass: true,
			Expected: "8-16", Actual: inActual,
			Detail: fmt.Sprintf("input_tokens=%d (范围 8-16 OK)", inTok),
		})
	} else {
		checks = append(checks, CheckResult{
			Name: "minimal_input_tokens", Pass: false,
			Expected: "8-16", Actual: inActual,
			Detail: fmt.Sprintf("input_tokens=%d, 期望 8-16", inTok),
		})
	}

	outTok := fingerprint.IntVal(usage, "output_tokens")
	outActual := fmt.Sprintf("%d", outTok)
	if outTok >= 1 && outTok <= 4 {
		checks = append(checks, CheckResult{
			Name: "minimal_output_tokens", Pass: true,
			Expected: "1-4", Actual: outActual,
			Detail: fmt.Sprintf("output_tokens=%d (范围 1-4 OK)", outTok),
		})
	} else {
		checks = append(checks, CheckResult{
			Name: "minimal_output_tokens", Pass: false,
			Expected: "1-4", Actual: outActual,
			Detail: fmt.Sprintf("output_tokens=%d, 期望 1-4", outTok),
		})
	}

	return checks, nil
}
