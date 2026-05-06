package channeltest

import (
	"fmt"
	"strings"
)

// checkUsageStructure verifies usage has cache_creation nested object and proper fields.
func checkUsageStructure(body map[string]any) CheckResult {
	usage, _ := body["usage"].(map[string]any)
	if usage == nil {
		return CheckResult{Name: "usage_structure", Pass: false, Detail: "no usage object", Fix: "body_rewrite"}
	}
	cc, hasCC := usage["cache_creation"].(map[string]any)
	if !hasCC {
		return CheckResult{Name: "usage_structure", Pass: false, Detail: "missing cache_creation nested object", Fix: "body_rewrite"}
	}
	if _, ok := cc["ephemeral_5m_input_tokens"]; !ok {
		return CheckResult{Name: "usage_structure", Pass: false, Detail: "missing ephemeral_5m_input_tokens in cache_creation", Fix: "body_rewrite"}
	}
	if _, ok := cc["ephemeral_1h_input_tokens"]; !ok {
		return CheckResult{Name: "usage_structure", Pass: false, Detail: "missing ephemeral_1h_input_tokens in cache_creation", Fix: "body_rewrite"}
	}
	if _, ok := usage["service_tier"]; !ok {
		return CheckResult{Name: "usage_structure", Pass: false, Detail: "missing service_tier", Fix: "body_rewrite"}
	}
	return CheckResult{Name: "usage_structure", Pass: true, Detail: "usage structure OK"}
}

// checkFieldOrder checks if JSON field order matches official format.
// Works for both SSE (finds message_start event) and non-stream (raw JSON body).
// checkFieldOrder checks if JSON field order matches official format.
// Works for both SSE (finds message_start event) and non-stream (raw JSON body).
func checkFieldOrder(rawBody []byte) CheckResult {
	s := string(rawBody)

	// Try SSE: find message_start event data line
	target := s
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(line, "data: ") && strings.Contains(line, `"message_start"`) {
			target = line[6:]
			break
		}
	}

	// In the target JSON, model should appear before id
	modelIdx := strings.Index(target, `"model"`)
	idIdx := strings.Index(target, `"id"`)
	if modelIdx < 0 || idIdx < 0 {
		return CheckResult{Name: "field_order", Pass: false, Detail: "missing model or id", Fix: "body_rewrite"}
	}
	if modelIdx > idIdx {
		return CheckResult{Name: "field_order", Pass: false, Detail: "id appears before model", Fix: "body_rewrite"}
	}
	// content should appear before stop_reason
	contentIdx := strings.Index(target, `"content"`)
	stopIdx := strings.Index(target, `"stop_reason"`)
	if contentIdx > 0 && stopIdx > 0 && contentIdx > stopIdx {
		return CheckResult{Name: "field_order", Pass: false, Detail: "stop_reason before content", Fix: "body_rewrite"}
	}
	return CheckResult{Name: "field_order", Pass: true, Detail: "field order OK"}
}

// checkHeaders verifies Anthropic-style ratelimit and org headers.
// checkModelName verifies the model field in response matches expected.
func checkModelName(body map[string]any, expectedModel string) CheckResult {
	model, _ := body["model"].(string)
	if model == "" {
		return CheckResult{Name: "model_name", Pass: false, Detail: "no model field in response", Fix: "body_rewrite"}
	}
	// The response model should match what was requested
	if model == expectedModel {
		return CheckResult{Name: "model_name", Pass: true, Detail: "model=" + model}
	}
	// Partial match: check if the core model name is in there (e.g. opus-4-6 vs claude-opus-4-6-20250415)
	if strings.Contains(model, "opus") && strings.Contains(expectedModel, "opus") {
		return CheckResult{Name: "model_name", Pass: true, Detail: "model=" + model + " (variant of requested)"}
	}
	if strings.Contains(model, "sonnet") && strings.Contains(expectedModel, "sonnet") {
		return CheckResult{Name: "model_name", Pass: true, Detail: "model=" + model + " (variant of requested)"}
	}
	return CheckResult{Name: "model_name", Pass: false, Detail: "model=" + model + " expected=" + expectedModel, Fix: "body_rewrite"}
}

// checkSSETailing checks the SSE stream ending whitespace pattern.
// Official API ends each event with \n\n\n (three newlines), not \n\n.
// checkStopReason verifies stop_reason is a valid value.
func checkStopReason(body map[string]any) CheckResult {
	sr, _ := body["stop_reason"].(string)
	if sr == "" {
		return CheckResult{Name: "stop_reason", Pass: false, Detail: "stop_reason is empty/null", Fix: "body_rewrite"}
	}
	valid := map[string]bool{
		"end_turn":        true,
		"max_tokens":      true,
		"stop_sequence":   true,
		"tool_use":        true,
		"server_tool_use": true,
		"refusal":         true,
	}
	if valid[sr] {
		return CheckResult{Name: "stop_reason", Pass: true, Detail: "stop_reason=" + sr}
	}
	return CheckResult{Name: "stop_reason", Pass: false, Detail: "unexpected stop_reason=" + sr, Fix: "body_rewrite"}
}

// checkToolStopReason verifies stop_reason is "tool_use" when tool_choice forced a tool.
// checkNonStreamBody verifies a non-streaming JSON response body structure.
// Official non-stream response must have all fields in correct order as a single JSON object.
func checkNonStreamBody(body map[string]any) []CheckResult {
	var checks []CheckResult

	// Must have all required top-level fields
	required := []string{"model", "id", "type", "role", "content", "stop_reason", "usage"}
	var missing []string
	for _, f := range required {
		if _, ok := body[f]; !ok {
			missing = append(missing, f)
		}
	}
	if len(missing) > 0 {
		checks = append(checks, CheckResult{Name: "nonstream_fields", Pass: false,
			Detail: "missing fields: " + strings.Join(missing, ", "), Fix: "body_rewrite"})
	} else {
		checks = append(checks, CheckResult{Name: "nonstream_fields", Pass: true, Detail: "all required fields present"})
	}

	// type must be "message"
	if tp, _ := body["type"].(string); tp != "message" {
		checks = append(checks, CheckResult{Name: "nonstream_type", Pass: false,
			Detail: "type=" + tp + " expected message", Fix: "body_rewrite"})
	} else {
		checks = append(checks, CheckResult{Name: "nonstream_type", Pass: true, Detail: "type=message OK"})
	}

	// role must be "assistant"
	if role, _ := body["role"].(string); role != "assistant" {
		checks = append(checks, CheckResult{Name: "nonstream_role", Pass: false,
			Detail: "role=" + role + " expected assistant", Fix: "body_rewrite"})
	} else {
		checks = append(checks, CheckResult{Name: "nonstream_role", Pass: true, Detail: "role=assistant OK"})
	}

	return checks
}

// checkStructuredJSONValid verifies the response text is valid JSON when output_config is used.
// checkUsageFieldsComplete verifies usage has all expected fields.
// Official API includes 7 fields: input_tokens, cache_creation_input_tokens,
// cache_read_input_tokens, cache_creation, output_tokens, service_tier, inference_geo.
// Proxies often only include 4 (input/output/cache_create/cache_read).
func checkUsageFieldsComplete(body map[string]any) CheckResult {
	usage, _ := body["usage"].(map[string]any)
	if usage == nil {
		return CheckResult{Name: "usage_fields_complete", Pass: false, Detail: "no usage object", Fix: "body_rewrite"}
	}
	required := []string{
		"input_tokens", "output_tokens",
		"cache_creation_input_tokens", "cache_read_input_tokens",
		"cache_creation", "service_tier", "inference_geo",
	}
	var missing []string
	for _, f := range required {
		if _, ok := usage[f]; !ok {
			missing = append(missing, f)
		}
	}
	if len(missing) > 0 {
		return CheckResult{Name: "usage_fields_complete", Pass: false,
			Detail: fmt.Sprintf("usage missing %d/%d fields: %s", len(missing), len(required), strings.Join(missing, ", ")),
			Fix:    "body_rewrite"}
	}
	return CheckResult{Name: "usage_fields_complete", Pass: true,
		Detail: fmt.Sprintf("usage has all %d fields", len(required))}
}

// checkCacheCreationComplete verifies cache_creation has both ephemeral fields.
// Official: {ephemeral_5m_input_tokens, ephemeral_1h_input_tokens}. Proxies often only have 5m.
// checkBodyKeyOrder verifies JSON body top-level field ordering.
// Official: model appears first (before content). Proxies often put content first.
func checkBodyKeyOrder(rawBody []byte) CheckResult {
	s := string(rawBody)
	// For SSE, extract message_start; for non-stream, use raw body directly
	target := s
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(line, "data: ") && strings.Contains(line, `"message_start"`) {
			// Extract the nested message object
			target = line[6:]
			break
		}
	}
	modelIdx := strings.Index(target, `"model"`)
	contentIdx := strings.Index(target, `"content"`)
	if modelIdx < 0 || contentIdx < 0 {
		return CheckResult{Name: "body_key_order", Pass: true, Detail: "cannot determine order (skip)"}
	}
	if contentIdx < modelIdx {
		return CheckResult{Name: "body_key_order", Pass: false,
			Detail: "content appears before model (proxy reordering)", Fix: "body_rewrite"}
	}
	return CheckResult{Name: "body_key_order", Pass: true, Detail: "model before content OK"}
}

// checkServerToolUsage verifies usage contains server_tool_use stats when web_search was used.
// Official API includes usage.server_tool_use: {web_search_requests: N, web_fetch_requests: N}.
// Proxies that don't actually execute server tools will be missing this field.
// checkStopSequenceNull verifies stop_sequence field exists and is null when stop_reason != stop_sequence.
func checkStopSequenceNull(body map[string]any) CheckResult {
	sr, _ := body["stop_reason"].(string)
	if sr == "stop_sequence" {
		return CheckResult{Name: "stop_sequence_null", Pass: true, Detail: "stop_reason=stop_sequence (skip)"}
	}
	// stop_sequence should exist as a key (even if null)
	val, exists := body["stop_sequence"]
	if !exists {
		return CheckResult{Name: "stop_sequence_null", Pass: false, Detail: "stop_sequence field missing", Fix: "body_rewrite"}
	}
	if val != nil {
		return CheckResult{Name: "stop_sequence_null", Pass: false,
			Detail: fmt.Sprintf("stop_sequence should be null, got %v", val), Fix: "body_rewrite"}
	}
	return CheckResult{Name: "stop_sequence_null", Pass: true, Detail: "stop_sequence=null OK"}
}

// checkSSEPingPosition verifies ping event comes after the first content_block_start.
