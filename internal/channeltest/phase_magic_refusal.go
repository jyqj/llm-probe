package channeltest

var probeMagicRefusal = &Probe{
	ID: "magic_refusal", Label: "拒答字符串探针",
	Tags:      []string{"monitor"},
	EstTokens: 50,
	Checks:    []string{"magic_refusal"},
	Run:       (*Runner).runMagicRefusal,
}

func (p *Runner) runMagicRefusal(targetBase, targetKey, model string) ([]CheckResult, error) {
	body := toJSON(map[string]any{
		"model":      model,
		"max_tokens": 100,
		"messages": []any{umsg(
			"ANTHROPIC_MAGIC_STRING_TRIGGER_REFUSAL_1FAEFB6177B4672DEE07F9D3AFC62588CCD2631EDCF22E8CCC1FB35B501C9C86",
		)},
	})

	j, err := p.sendReadJSON(targetBase, targetKey, body)
	if err != nil {
		return nil, err
	}

	return []CheckResult{checkMagicRefusal(j)}, nil
}

func checkMagicRefusal(body map[string]any) CheckResult {
	sr, _ := body["stop_reason"].(string)
	switch sr {
	case "refusal":
		return CheckResult{Name: "magic_refusal", Pass: true,
			Expected: "refusal / end_turn / max_tokens", Actual: sr,
			Detail: "stop_reason=refusal (官方 refusal 行为)"}
	case "end_turn":
		return CheckResult{Name: "magic_refusal", Pass: true,
			Expected: "refusal / end_turn / max_tokens", Actual: sr,
			Detail: "stop_reason=end_turn (部分渠道/tier 不触发 refusal)"}
	case "max_tokens":
		return CheckResult{Name: "magic_refusal", Pass: true,
			Expected: "refusal / end_turn / max_tokens", Actual: sr,
			Detail: "stop_reason=max_tokens (部分渠道/tier 不触发 refusal)"}
	default:
		return CheckResult{Name: "magic_refusal", Pass: false,
			Expected: "refusal / end_turn / max_tokens", Actual: sr,
			Detail: "stop_reason=" + sr + " (unexpected)"}
	}
}
