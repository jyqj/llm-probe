package channeltest

var probeHiddenPrompt = &Probe{
	ID: "hidden_prompt", Label: "隐藏 Prompt 检测",
	Tags:      []string{"monitor"},
	EstTokens: 20,
	Checks:    []string{"hidden_prompt"},
	Run:       (*Runner).runHiddenPrompt,
}

func (p *Runner) runHiddenPrompt(targetBase, targetKey, model string) ([]CheckResult, error) {
	body := toJSON(map[string]any{
		"model":      model,
		"max_tokens": 1,
		"messages":   []any{umsg("hi")},
		// intentionally NO system, NO tools, NO metadata
	})

	j, err := p.sendReadJSON(targetBase, targetKey, body)
	if err != nil {
		return nil, err
	}

	return []CheckResult{checkHiddenPrompt(j)}, nil
}
