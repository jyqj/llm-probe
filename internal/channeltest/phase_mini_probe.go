package channeltest

var probeMiniProbe = &Probe{
	ID: "mini_probe", Label: "极简非流式探针",
	Tags:      []string{"monitor"},
	EstTokens: 100,
	Checks:    []string{"backend_type", "small_output_tokens", "small_stop_reason", "small_ephemeral_zero", "small_cache_zero", "token_budget"},
	Run:       (*Runner).runMiniProbe,
}

func (p *Runner) runMiniProbe(targetBase, targetKey, model string) ([]CheckResult, error) {
	body := toJSON(map[string]any{
		"model":      model,
		"max_tokens": 1,
		"stream":     false,
		"system":     fullSystem(),
		"metadata":   genMetadata(),
		"messages":   []any{umsg("hi")},
	})

	j, err := p.sendReadJSON(targetBase, targetKey, body)
	if err != nil {
		return nil, err
	}

	var checks []CheckResult
	checks = append(checks, checkBackendType(j))
	checks = append(checks, checkSmallProbeExact(j)...)
	checks = append(checks, checkTokenBudget(j, model))
	return checks, nil
}
