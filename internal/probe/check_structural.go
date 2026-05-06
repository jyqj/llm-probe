package probe

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"detector-service/internal/fingerprint"
)

func hasXRatelimitHeaders(headers http.Header) bool {
	return headers.Get("X-Ratelimit-Limit-Requests") != "" &&
		headers.Get("X-Ratelimit-Limit-Tokens") != "" &&
		headers.Get("X-Ratelimit-Remaining-Requests") != "" &&
		headers.Get("X-Ratelimit-Remaining-Tokens") != ""
}

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
func checkHeaders(headers http.Header) CheckResult {
	// Accept both direct Anthropic Console-style headers and Azure/managed-channel
	// X-Ratelimit-* headers. The reference set marks both as clean channels.
	if hasXRatelimitHeaders(headers) {
		return CheckResult{Name: "headers", Pass: true, Detail: "managed-channel X-Ratelimit headers present"}
	}

	required := []string{
		"Anthropic-Ratelimit-Input-Tokens-Limit",
		"Anthropic-Ratelimit-Input-Tokens-Remaining",
		"Anthropic-Ratelimit-Input-Tokens-Reset",
		"Anthropic-Ratelimit-Output-Tokens-Limit",
		"Anthropic-Ratelimit-Output-Tokens-Remaining",
		"Anthropic-Ratelimit-Output-Tokens-Reset",
		"Anthropic-Ratelimit-Requests-Limit",
		"Anthropic-Ratelimit-Requests-Remaining",
		"Anthropic-Ratelimit-Requests-Reset",
		"Anthropic-Ratelimit-Tokens-Limit",
		"Anthropic-Ratelimit-Tokens-Remaining",
		"Anthropic-Ratelimit-Tokens-Reset",
		"Anthropic-Organization-Id",
	}
	missing := 0
	for _, h := range required {
		if headers.Get(h) == "" {
			missing++
		}
	}
	if missing == 0 {
		return CheckResult{Name: "headers", Pass: true, Detail: "all Anthropic ratelimit headers present"}
	}
	return CheckResult{Name: "headers", Pass: false,
		Detail: fmt.Sprintf("missing %d/%d ratelimit headers", missing, len(required)),
		Fix:    "headers_fake"}
}

// checkSSEDone checks if the SSE stream ends with [DONE] sentinel.
func checkSSEDone(sseData string) CheckResult {
	if strings.Contains(sseData, "data: [DONE]") {
		return CheckResult{Name: "sse_done", Pass: false, Detail: "[DONE] sentinel found in stream", Fix: "strip_done"}
	}
	return CheckResult{Name: "sse_done", Pass: true, Detail: "no [DONE] sentinel"}
}

// checkSSEEventOrder verifies SSE events follow the official order:
// message_start -> content_block_start -> ping -> deltas -> content_block_stop -> message_delta -> message_stop
func checkSSEEventOrder(sseData string) CheckResult {
	var events []string
	for _, line := range strings.Split(sseData, "\n") {
		if !strings.HasPrefix(line, "data: ") || line == "data: [DONE]" {
			continue
		}
		var ev map[string]any
		if json.Unmarshal([]byte(line[6:]), &ev) != nil {
			continue
		}
		t, _ := ev["type"].(string)
		if t != "" {
			events = append(events, t)
		}
	}
	if len(events) == 0 {
		return CheckResult{Name: "sse_event_order", Pass: false, Detail: "no SSE events parsed", Fix: "body_rewrite"}
	}
	// First event must be message_start
	if events[0] != "message_start" {
		return CheckResult{Name: "sse_event_order", Pass: false, Detail: "first event is " + events[0] + " not message_start", Fix: "body_rewrite"}
	}
	// Last event must be message_stop
	if events[len(events)-1] != "message_stop" {
		return CheckResult{Name: "sse_event_order", Pass: false, Detail: "last event is " + events[len(events)-1] + " not message_stop", Fix: "body_rewrite"}
	}
	// Ping should exist
	hasPing := false
	for _, e := range events {
		if e == "ping" {
			hasPing = true
			break
		}
	}
	if !hasPing {
		return CheckResult{Name: "sse_event_order", Pass: false, Detail: "no ping event in stream", Fix: "body_rewrite"}
	}
	return CheckResult{Name: "sse_event_order", Pass: true, Detail: fmt.Sprintf("%d events, order OK", len(events))}
}

// checkCacheSmallProbe checks if cache values are zero for small max_tokens requests (no cache_control).
func checkCacheSmallProbe(body map[string]any) CheckResult {
	usage, _ := body["usage"].(map[string]any)
	if usage == nil {
		return CheckResult{Name: "cache_small_probe", Pass: true, Detail: "no usage"}
	}
	ccCreate := fingerprint.IntVal(usage, "cache_creation_input_tokens")
	ccRead := fingerprint.IntVal(usage, "cache_read_input_tokens")
	if ccCreate != 0 || ccRead != 0 {
		return CheckResult{Name: "cache_small_probe", Pass: false,
			Detail: fmt.Sprintf("small probe has non-zero cache: create=%d read=%d", ccCreate, ccRead),
			Fix:    "small_probe_zero"}
	}
	return CheckResult{Name: "cache_small_probe", Pass: true, Detail: "cache values are zero for small probe"}
}

// checkCacheFake checks if cache values look reasonable when cache_control was used.
func checkCacheFake(body map[string]any) CheckResult {
	usage, _ := body["usage"].(map[string]any)
	if usage == nil {
		return CheckResult{Name: "cache_fake", Pass: true, Detail: "no usage"}
	}
	ccCreate := fingerprint.IntVal(usage, "cache_creation_input_tokens")
	ccRead := fingerprint.IntVal(usage, "cache_read_input_tokens")
	if ccCreate == 0 && ccRead == 0 {
		return CheckResult{Name: "cache_fake", Pass: false,
			Detail: "cache_control used but cache all zero", Fix: "cache_fake"}
	}
	return CheckResult{Name: "cache_fake", Pass: true,
		Detail: fmt.Sprintf("cache values non-zero: create=%d read=%d", ccCreate, ccRead)}
}

// checkWebSearchResult verifies web_search_tool_result content structure.
func checkWebSearchResult(body map[string]any) CheckResult {
	content, _ := body["content"].([]any)
	for _, cb := range content {
		m, ok := cb.(map[string]any)
		if !ok {
			continue
		}
		t, _ := m["type"].(string)
		if t == "web_search_tool_result" {
			// Check tool_use_id format
			tid, _ := m["tool_use_id"].(string)
			if tid != "" && !srvToolIDRe.MatchString(tid) {
				return CheckResult{Name: "web_search_result", Pass: false,
					Detail: "web_search_tool_result tool_use_id not srvtoolu_01 format: " + truncate(tid, 20), Fix: "id_rewrite"}
			}
			// Check content items have encrypted_content
			items, _ := m["content"].([]any)
			for _, item := range items {
				im, ok := item.(map[string]any)
				if !ok {
					continue
				}
				it, _ := im["type"].(string)
				if it == "web_search_result" {
					ec, _ := im["encrypted_content"].(string)
					if ec == "" {
						return CheckResult{Name: "web_search_result", Pass: false,
							Detail: "web_search_result missing encrypted_content", Fix: "signature_rewrite"}
					}
					// Verify it's valid base64
					_, err := base64.StdEncoding.DecodeString(ec)
					if err != nil {
						return CheckResult{Name: "web_search_result", Pass: false,
							Detail: "encrypted_content not valid base64", Fix: "signature_rewrite"}
					}
				}
			}
			return CheckResult{Name: "web_search_result", Pass: true, Detail: "web_search_tool_result structure OK"}
		}
	}
	return CheckResult{Name: "web_search_result", Pass: true, Detail: "no web_search_tool_result (skip)"}
}

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
func checkSSETailing(sseData string) CheckResult {
	// Count triple-newline sequences
	tripleCount := strings.Count(sseData, "\n\n\n")
	doubleOnly := strings.Count(sseData, "\n\n") - tripleCount
	if tripleCount > 0 {
		return CheckResult{Name: "sse_tailing", Pass: true,
			Detail: fmt.Sprintf("triple-newline endings found (%d)", tripleCount)}
	}
	if doubleOnly > 0 {
		// Informational only - not auto-fixable
		return CheckResult{Name: "sse_tailing", Pass: false,
			Detail: fmt.Sprintf("only double-newline endings (%d), official uses triple", doubleOnly)}
	}
	return CheckResult{Name: "sse_tailing", Pass: true, Detail: "no newline patterns to check"}
}

// checkCfHeaders verifies cloudflare-style headers (Cf-Ray, Server, Set-Cookie).
// These are part of HeadersFake and help pass fingerprint checks.
func checkCfHeaders(headers http.Header) CheckResult {
	if hasXRatelimitHeaders(headers) {
		return CheckResult{Name: "cf_headers", Pass: true, Detail: "managed-channel headers (no Cloudflare expected)"}
	}

	var missing []string
	if headers.Get("Cf-Ray") == "" {
		missing = append(missing, "Cf-Ray")
	}
	// Cf-Cache-Status is commonly present on Anthropic API, but not universal in
	// captured clean Console samples, so keep it optional.
	server := headers.Get("Server")
	if server == "" {
		missing = append(missing, "Server")
	}
	cookie := headers.Get("Set-Cookie")
	if cookie == "" || !strings.Contains(cookie, "_cfuvid") {
		missing = append(missing, "Set-Cookie(_cfuvid)")
	}
	if len(missing) == 0 {
		return CheckResult{Name: "cf_headers", Pass: true, Detail: "Cloudflare-style headers present"}
	}
	return CheckResult{Name: "cf_headers", Pass: false,
		Detail: "missing: " + strings.Join(missing, ", "), Fix: "headers_fake"}
}

// checkMessageDeltaUsage verifies message_delta usage is "slim" format
// (only input_tokens, cache_creation_input_tokens, cache_read_input_tokens, output_tokens).
// Full usage fields like service_tier, inference_geo, cache_creation should NOT appear in delta.
func checkMessageDeltaUsage(sseData string) CheckResult {
	for _, line := range strings.Split(sseData, "\n") {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var ev map[string]any
		if json.Unmarshal([]byte(line[6:]), &ev) != nil {
			continue
		}
		t, _ := ev["type"].(string)
		if t != "message_delta" {
			continue
		}
		usage, _ := ev["usage"].(map[string]any)
		if usage == nil {
			return CheckResult{Name: "delta_usage_slim", Pass: false, Detail: "no usage in message_delta", Fix: "body_rewrite"}
		}
		// Slim format should NOT contain these full-only fields
		bloatFields := []string{"service_tier", "inference_geo", "cache_creation"}
		var found []string
		for _, f := range bloatFields {
			if _, ok := usage[f]; ok {
				found = append(found, f)
			}
		}
		if len(found) > 0 {
			return CheckResult{Name: "delta_usage_slim", Pass: false,
				Detail: "message_delta usage has full fields: " + strings.Join(found, ", "), Fix: "body_rewrite"}
		}
		return CheckResult{Name: "delta_usage_slim", Pass: true, Detail: "message_delta usage is slim format"}
	}
	return CheckResult{Name: "delta_usage_slim", Pass: true, Detail: "no message_delta event found (skip)"}
}

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
func checkToolStopReason(body map[string]any) CheckResult {
	sr, _ := body["stop_reason"].(string)
	hasTool := false
	content, _ := body["content"].([]any)
	for _, cb := range content {
		m, ok := cb.(map[string]any)
		if !ok {
			continue
		}
		t, _ := m["type"].(string)
		if t == "tool_use" || t == "server_tool_use" || t == "web_search_tool_result" {
			hasTool = true
			break
		}
	}
	if hasTool && (sr == "tool_use" || sr == "server_tool_use" || sr == "end_turn") {
		return CheckResult{Name: "tool_stop_reason", Pass: true, Detail: "tool content present with stop_reason=" + sr}
	}
	if sr == "" {
		return CheckResult{Name: "tool_stop_reason", Pass: false,
			Detail: "stop_reason empty for forced tool request", Fix: "body_rewrite"}
	}
	return CheckResult{Name: "tool_stop_reason", Pass: false,
		Detail: "stop_reason=" + sr + " without expected tool content", Fix: "body_rewrite"}
}

// checkSmallProbeExact performs the detect_max "9-point" verification on a max_tokens=1 response:
// 1. output_tokens must be exactly 1
// 2. stop_reason must be "max_tokens"
// 3. cache_creation nested object must exist with ephemeral values = 0
// 4. top-level cache fields must be 0
func checkSmallProbeExact(body map[string]any) []CheckResult {
	var checks []CheckResult
	usage, _ := body["usage"].(map[string]any)
	if usage == nil {
		checks = append(checks, CheckResult{Name: "small_output_tokens", Pass: false, Detail: "no usage object", Fix: "body_rewrite"})
		return checks
	}

	// output_tokens must be exactly 1 for max_tokens=1
	outTok := fingerprint.IntVal(usage, "output_tokens")
	if outTok == 1 {
		checks = append(checks, CheckResult{Name: "small_output_tokens", Pass: true, Detail: "output_tokens=1 OK"})
	} else {
		checks = append(checks, CheckResult{Name: "small_output_tokens", Pass: false,
			Detail: fmt.Sprintf("output_tokens=%d expected 1", outTok), Fix: "body_rewrite"})
	}

	// stop_reason must be "max_tokens"
	sr, _ := body["stop_reason"].(string)
	if sr == "max_tokens" {
		checks = append(checks, CheckResult{Name: "small_stop_reason", Pass: true, Detail: "stop_reason=max_tokens OK"})
	} else {
		checks = append(checks, CheckResult{Name: "small_stop_reason", Pass: false,
			Detail: "stop_reason=" + sr + " expected max_tokens", Fix: "body_rewrite"})
	}

	// cache_creation nested: ephemeral values must be 0
	cc, hasCC := usage["cache_creation"].(map[string]any)
	if !hasCC {
		checks = append(checks, CheckResult{Name: "small_ephemeral_zero", Pass: false,
			Detail: "no cache_creation nested object", Fix: "body_rewrite"})
	} else {
		e5m := fingerprint.IntVal(cc, "ephemeral_5m_input_tokens")
		e1h := fingerprint.IntVal(cc, "ephemeral_1h_input_tokens")
		if e5m == 0 && e1h == 0 {
			checks = append(checks, CheckResult{Name: "small_ephemeral_zero", Pass: true,
				Detail: "ephemeral values both 0 OK"})
		} else {
			checks = append(checks, CheckResult{Name: "small_ephemeral_zero", Pass: false,
				Detail: fmt.Sprintf("ephemeral_5m=%d ephemeral_1h=%d should be 0", e5m, e1h), Fix: "small_probe_zero"})
		}
	}

	// top-level cache fields must be 0
	ccCreate := fingerprint.IntVal(usage, "cache_creation_input_tokens")
	ccRead := fingerprint.IntVal(usage, "cache_read_input_tokens")
	if ccCreate == 0 && ccRead == 0 {
		checks = append(checks, CheckResult{Name: "small_cache_zero", Pass: true,
			Detail: "cache_creation/read both 0 OK"})
	} else {
		checks = append(checks, CheckResult{Name: "small_cache_zero", Pass: false,
			Detail: fmt.Sprintf("cache create=%d read=%d should be 0", ccCreate, ccRead), Fix: "small_probe_zero"})
	}

	return checks
}

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
func checkStructuredJSONValid(body map[string]any) CheckResult {
	text := collectResponseText(body)
	if text == "" {
		return CheckResult{Name: "structured_json_valid", Pass: false, Detail: "no text content", Fix: "body_rewrite"}
	}
	var parsed any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return CheckResult{Name: "structured_json_valid", Pass: false,
			Detail: "response is not valid JSON: " + truncate(err.Error(), 60), Fix: "body_rewrite"}
	}
	return CheckResult{Name: "structured_json_valid", Pass: true, Detail: "valid JSON response"}
}

// checkStructuredSchemaMatch verifies the JSON response matches the expected schema {title: string, desc: string}.
func checkStructuredSchemaMatch(body map[string]any) CheckResult {
	text := collectResponseText(body)
	if text == "" {
		return CheckResult{Name: "structured_schema_match", Pass: false, Detail: "no text content", Fix: "body_rewrite"}
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(text), &obj); err != nil {
		return CheckResult{Name: "structured_schema_match", Pass: false,
			Detail: "not a JSON object", Fix: "body_rewrite"}
	}
	title, hasTitle := obj["title"].(string)
	desc, hasDesc := obj["desc"].(string)
	if !hasTitle || !hasDesc {
		var missing []string
		if !hasTitle {
			missing = append(missing, "title")
		}
		if !hasDesc {
			missing = append(missing, "desc")
		}
		return CheckResult{Name: "structured_schema_match", Pass: false,
			Detail: "missing required fields: " + strings.Join(missing, ", "), Fix: "body_rewrite"}
	}
	if title == "" || desc == "" {
		return CheckResult{Name: "structured_schema_match", Pass: false,
			Detail: "title or desc is empty", Fix: "body_rewrite"}
	}
	// Check no extra fields (additionalProperties: false)
	extra := 0
	for k := range obj {
		if k != "title" && k != "desc" {
			extra++
		}
	}
	if extra > 0 {
		return CheckResult{Name: "structured_schema_match", Pass: false,
			Detail: fmt.Sprintf("schema has %d extra fields (additionalProperties should be false)", extra), Fix: "body_rewrite"}
	}
	return CheckResult{Name: "structured_schema_match", Pass: true,
		Detail: "schema match: {title, desc} OK"}
}

// checkStructuredStopReason verifies stop_reason is "end_turn" when structured output is used.
// With output_config json_schema, stop_reason should be "end_turn" (not "max_tokens").
// "refusal" is also a valid structured output stop_reason (safety refusal).
func checkStructuredStopReason(body map[string]any) CheckResult {
	sr, _ := body["stop_reason"].(string)
	if sr == "" {
		return CheckResult{Name: "structured_stop_reason", Pass: false,
			Detail: "stop_reason is empty/null", Fix: "body_rewrite"}
	}
	if sr == "end_turn" {
		return CheckResult{Name: "structured_stop_reason", Pass: true, Detail: "stop_reason=end_turn OK"}
	}
	if sr == "refusal" {
		return CheckResult{Name: "structured_stop_reason", Pass: true, Detail: "stop_reason=refusal (safety)"}
	}
	return CheckResult{Name: "structured_stop_reason", Pass: false,
		Detail: "stop_reason=" + sr + " expected end_turn for structured output", Fix: "body_rewrite"}
}

// checkMessageStartUsage verifies the message_start event contains input-side usage fields.
// Official streaming: message_start.usage has input_tokens, cache_creation_input_tokens, cache_read_input_tokens.
func checkMessageStartUsage(sseData string) CheckResult {
	for _, line := range strings.Split(sseData, "\n") {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var ev map[string]any
		if json.Unmarshal([]byte(line[6:]), &ev) != nil {
			continue
		}
		t, _ := ev["type"].(string)
		if t != "message_start" {
			continue
		}
		msg, _ := ev["message"].(map[string]any)
		if msg == nil {
			return CheckResult{Name: "message_start_usage", Pass: false,
				Detail: "message_start has no message object", Fix: "body_rewrite"}
		}
		usage, _ := msg["usage"].(map[string]any)
		if usage == nil {
			return CheckResult{Name: "message_start_usage", Pass: false,
				Detail: "message_start.message has no usage", Fix: "body_rewrite"}
		}
		// Must have input_tokens
		if _, ok := usage["input_tokens"]; !ok {
			return CheckResult{Name: "message_start_usage", Pass: false,
				Detail: "message_start usage missing input_tokens", Fix: "body_rewrite"}
		}
		return CheckResult{Name: "message_start_usage", Pass: true,
			Detail: fmt.Sprintf("message_start usage OK: input_tokens=%d", fingerprint.IntVal(usage, "input_tokens"))}
	}
	return CheckResult{Name: "message_start_usage", Pass: true, Detail: "no message_start event (skip)"}
}

// checkServerTiming verifies Server-Timing header exists (envoy upstream service time).
func checkServerTiming(headers http.Header) CheckResult {
	if hasXRatelimitHeaders(headers) {
		return CheckResult{Name: "server_timing", Pass: true, Detail: "managed-channel headers (Server-Timing optional)"}
	}
	if envoy := headers.Get("X-Envoy-Upstream-Service-Time"); envoy != "" {
		return CheckResult{Name: "server_timing", Pass: true, Detail: "X-Envoy-Upstream-Service-Time present: " + truncate(envoy, 40)}
	}
	st := headers.Get("Server-Timing")
	if st == "" {
		return CheckResult{Name: "server_timing", Pass: false, Detail: "no Server-Timing/X-Envoy header", Fix: "headers_fake"}
	}
	if strings.Contains(st, "x-originResponse") {
		return CheckResult{Name: "server_timing", Pass: true, Detail: "Server-Timing OK: " + truncate(st, 40)}
	}
	return CheckResult{Name: "server_timing", Pass: false, Detail: "Server-Timing format unexpected: " + truncate(st, 40), Fix: "headers_fake"}
}

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
func checkCacheCreationComplete(body map[string]any) CheckResult {
	usage, _ := body["usage"].(map[string]any)
	if usage == nil {
		return CheckResult{Name: "cache_creation_complete", Pass: true, Detail: "no usage (skip)"}
	}
	cc, ok := usage["cache_creation"].(map[string]any)
	if !ok {
		return CheckResult{Name: "cache_creation_complete", Pass: false, Detail: "no cache_creation nested object", Fix: "body_rewrite"}
	}
	_, has5m := cc["ephemeral_5m_input_tokens"]
	_, has1h := cc["ephemeral_1h_input_tokens"]
	if has5m && has1h {
		return CheckResult{Name: "cache_creation_complete", Pass: true, Detail: "both ephemeral fields present"}
	}
	if has5m && !has1h {
		return CheckResult{Name: "cache_creation_complete", Pass: false,
			Detail: "missing ephemeral_1h_input_tokens", Fix: "body_rewrite"}
	}
	return CheckResult{Name: "cache_creation_complete", Pass: false,
		Detail: "missing ephemeral fields", Fix: "body_rewrite"}
}

// checkServerToolType verifies that web_search uses server_tool_use (not plain tool_use).
// Official API uses server_tool_use + web_search_tool_result for built-in tools.
// Proxies may downgrade to plain tool_use.
func checkServerToolType(body map[string]any) CheckResult {
	content, _ := body["content"].([]any)
	hasPlainToolUse := false
	hasServerToolUse := false
	for _, cb := range content {
		m, ok := cb.(map[string]any)
		if !ok {
			continue
		}
		t, _ := m["type"].(string)
		name, _ := m["name"].(string)
		if t == "tool_use" && name == "web_search" {
			hasPlainToolUse = true
		}
		if t == "server_tool_use" && name == "web_search" {
			hasServerToolUse = true
		}
	}
	if hasServerToolUse {
		return CheckResult{Name: "server_tool_type", Pass: true, Detail: "web_search uses server_tool_use"}
	}
	if hasPlainToolUse {
		return CheckResult{Name: "server_tool_type", Pass: false,
			Detail: "web_search uses plain tool_use instead of server_tool_use", Fix: "body_rewrite"}
	}
	return CheckResult{Name: "server_tool_type", Pass: true, Detail: "no web_search tool (skip)"}
}

// checkCitationsPresent verifies text blocks with citations exist in web search results.
// Official web_search responses include text blocks with citations arrays.
func checkCitationsPresent(body map[string]any) CheckResult {
	content, _ := body["content"].([]any)
	hasWebResult := false
	hasCitations := false
	for _, cb := range content {
		m, ok := cb.(map[string]any)
		if !ok {
			continue
		}
		t, _ := m["type"].(string)
		if t == "web_search_tool_result" {
			hasWebResult = true
		}
		if t == "text" {
			if _, ok := m["citations"]; ok {
				hasCitations = true
			}
		}
	}
	if !hasWebResult {
		return CheckResult{Name: "citations_present", Pass: true, Detail: "no web_search result (skip)"}
	}
	if hasCitations {
		return CheckResult{Name: "citations_present", Pass: true, Detail: "citations present in text blocks"}
	}
	return CheckResult{Name: "citations_present", Pass: false,
		Detail: "web_search result present but no text blocks with citations", Fix: "body_rewrite"}
}

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
func checkServerToolUsage(body map[string]any) CheckResult {
	// Only check if response actually contains server_tool_use or web_search_tool_result blocks
	content, _ := body["content"].([]any)
	hasServerTool := false
	for _, cb := range content {
		m, ok := cb.(map[string]any)
		if !ok {
			continue
		}
		t, _ := m["type"].(string)
		if t == "server_tool_use" || t == "web_search_tool_result" {
			hasServerTool = true
			break
		}
	}
	if !hasServerTool {
		return CheckResult{Name: "server_tool_usage", Pass: true, Detail: "no server tool in response (skip)"}
	}
	usage, _ := body["usage"].(map[string]any)
	if usage == nil {
		return CheckResult{Name: "server_tool_usage", Pass: false, Detail: "no usage object", Fix: "body_rewrite"}
	}
	stu, ok := usage["server_tool_use"].(map[string]any)
	if !ok {
		return CheckResult{Name: "server_tool_usage", Pass: false,
			Detail: "server tool used but usage.server_tool_use missing", Fix: "body_rewrite"}
	}
	if _, ok := stu["web_search_requests"]; !ok {
		return CheckResult{Name: "server_tool_usage", Pass: false,
			Detail: "server_tool_use missing web_search_requests", Fix: "body_rewrite"}
	}
	return CheckResult{Name: "server_tool_usage", Pass: true,
		Detail: fmt.Sprintf("server_tool_use present: %v", stu)}
}

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
func checkSSEPingPosition(sseData string) CheckResult {
	seenBlockStart := false
	for _, line := range strings.Split(sseData, "\n") {
		if !strings.HasPrefix(line, "data: ") || line == "data: [DONE]" {
			continue
		}
		var ev map[string]any
		if json.Unmarshal([]byte(line[6:]), &ev) != nil {
			continue
		}
		t, _ := ev["type"].(string)
		switch t {
		case "content_block_start":
			seenBlockStart = true
		case "ping":
			if !seenBlockStart {
				return CheckResult{Name: "sse_ping_position", Pass: false,
					Detail: "ping before content_block_start", Fix: "body_rewrite"}
			}
			return CheckResult{Name: "sse_ping_position", Pass: true, Detail: "ping after content_block_start OK"}
		}
	}
	return CheckResult{Name: "sse_ping_position", Pass: true, Detail: "no ping event (skip)"}
}

// checkMessageStartOutputZero verifies output_tokens in message_start is 0.
// Official API always starts with output_tokens=0 in message_start.
func checkMessageStartOutputZero(sseData string) CheckResult {
	for _, line := range strings.Split(sseData, "\n") {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var ev map[string]any
		if json.Unmarshal([]byte(line[6:]), &ev) != nil {
			continue
		}
		t, _ := ev["type"].(string)
		if t != "message_start" {
			continue
		}
		msg, _ := ev["message"].(map[string]any)
		if msg == nil {
			return CheckResult{Name: "message_start_output_zero", Pass: false, Detail: "no message object", Fix: "body_rewrite"}
		}
		usage, _ := msg["usage"].(map[string]any)
		if usage == nil {
			return CheckResult{Name: "message_start_output_zero", Pass: false, Detail: "no usage in message_start", Fix: "body_rewrite"}
		}
		outTok := fingerprint.IntVal(usage, "output_tokens")
		if outTok == 0 {
			return CheckResult{Name: "message_start_output_zero", Pass: true, Detail: "output_tokens=0 OK"}
		}
		return CheckResult{Name: "message_start_output_zero", Pass: false,
			Detail: fmt.Sprintf("output_tokens=%d, expected 0", outTok), Fix: "body_rewrite"}
	}
	return CheckResult{Name: "message_start_output_zero", Pass: true, Detail: "no message_start (skip)"}
}
