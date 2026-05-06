package channeltest

import "detector-service/internal/channeltest/data"

// ════════════════════════════════════════════════════════════
//  Phase 2c: self_intro
//  cctest 04_self_intro: thinking=nil (NOT set), stream=true,
//  28 tools, full system, max_tokens=1024
// ════════════════════════════════════════════════════════════

func (p *Runner) runSelfIntroProbe(targetBase, targetKey, model string) ([]CheckResult, error) {
	body := toJSON(map[string]any{
		"model":      model,
		"max_tokens": 1024,
		"stream":     true,
		"system":     fullSystem(),
		"tools":      data.Tools(),
		"metadata":   genMetadata(),
		"messages":   []any{umsg("用一句话介绍你自己，包含标题和描述")},
		// NO thinking field — matches cctest 04
		"output_config": map[string]any{
			"format": map[string]any{
				"type": "json_schema",
				"schema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"title": map[string]any{"type": "string"},
						"desc":  map[string]any{"type": "string"},
					},
					"required":             []string{"title", "desc"},
					"additionalProperties": false,
				},
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
			{Name: "no_thinking_leak", Pass: false, Detail: "parse failed", Fix: "body_rewrite"},
			{Name: "identity_response", Pass: false, Detail: "parse failed"},
			{Name: "structured_json_valid", Pass: false, Detail: "parse failed", Fix: "body_rewrite"},
			{Name: "structured_schema_match", Pass: false, Detail: "parse failed", Fix: "body_rewrite"},
			{Name: "structured_stop_reason", Pass: false, Detail: "parse failed", Fix: "body_rewrite"},
		}, nil
	}

	return []CheckResult{
		checkNoThinkingWhenDisabled(full),
		checkIdentityResponse(full),
		checkStructuredJSONValid(full),
		checkStructuredSchemaMatch(full),
		checkStructuredStopReason(full),
	}, nil
}
