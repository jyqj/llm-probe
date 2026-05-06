package channeltest

// ════════════════════════════════════════════════════════════
//  Phase 2d: tool_use
//  cctest 05_tool_use: web_search_20250305, tool_choice=web_search,
//  temperature=1, 3 system blocks (billing + CC + short instruction),
//  cache_control on user content, metadata
// ════════════════════════════════════════════════════════════

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
