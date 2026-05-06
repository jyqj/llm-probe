package channeltest

// ════════════════════════════════════════════════════════════
//  Phase 2a: mini_probe (detect_max 9-point)
//  max_tokens=1, stream=false, system with cache_control, billing header
// ════════════════════════════════════════════════════════════

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
