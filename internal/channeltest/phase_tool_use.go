package channeltest

var probeToolUse = &Probe{
	ID: "tool_use", Label: "工具调用探针",
	Tags:      []string{},
	EstTokens: 1000,
	Checks:    []string{"tool_use_id", "tool_stop_reason", "tool_forced_compliance", "web_search_result", "server_tool_type", "citations_present", "server_tool_usage"},
	Run:       (*Runner).runToolUseProbe,
}

func (p *Runner) runToolUseProbe(targetBase, targetKey, model string) ([]CheckResult, error) {
	body := toJSON(map[string]any{
		"model":       model,
		"max_tokens":  16000,
		"stream":      true,
		"temperature": 1,
		"system": []any{
			billingBlock(),
			map[string]any{
				"type":          "text",
				"text":          "You are Claude Code, Anthropic's official CLI for Claude.",
				"cache_control": map[string]any{"type": "ephemeral"},
			},
			map[string]any{
				"type":          "text",
				"text":          "You are an assistant for performing a web search tool use",
				"cache_control": map[string]any{"type": "ephemeral"},
			},
		},
		"tools": []any{
			map[string]any{
				"type":     "web_search_20250305",
				"name":     "web_search",
				"max_uses": 1,
			},
		},
		"tool_choice": map[string]any{"type": "tool", "name": "web_search"},
		"metadata":    genMetadata(),
		"messages": []any{
			map[string]any{
				"role": "user",
				"content": []any{map[string]any{
					"type":          "text",
					"text":          "Perform a web search for the query: latest AI news today",
					"cache_control": map[string]any{"type": "ephemeral"},
				}},
			},
		},
	})

	resp, err := p.send(targetBase, targetKey, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	sse, start, delta := readSSE(resp.Body)
	full := merge(start, delta, sse)
	p.recordStreamResult(full)
	if full == nil {
		return []CheckResult{
			{Name: "tool_use_id", Pass: false, Detail: "parse failed", Fix: "id_rewrite"},
			{Name: "web_search_result", Pass: false, Detail: "parse failed", Fix: "signature_rewrite"},
		}, nil
	}

	return []CheckResult{
		checkToolUseID(full),
		checkToolStopReason(full),
		checkToolForcedCompliance(full),
		checkWebSearchResult(full),
		checkServerToolType(full),
		checkCitationsPresent(full),
		checkServerToolUsage(full),
	}, nil
}
