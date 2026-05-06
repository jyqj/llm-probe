package channeltest

// ════════════════════════════════════════════════════════════
//  Phase 2g: hidden_prompt
//  Send a bare request with NO system prompt. If input_tokens is abnormally
//  high (>15 for "hi"), the upstream has injected a hidden system prompt.
// ════════════════════════════════════════════════════════════

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
