package probe

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"bedrock-gateway/internal/fingerprint"
)

var (
	msgIDRe     = regexp.MustCompile(`^msg_01[0-9A-Za-z]{22}$`)
	toolIDRe    = regexp.MustCompile(`^toolu_01[0-9A-Za-z]{22}$`)
	srvToolIDRe = regexp.MustCompile(`^srvtoolu_01[0-9A-Za-z]{22}$`)
	reqIDRe     = regexp.MustCompile(`^req_01[0-9A-Za-z]+$`)
)

// checkIDFormat verifies the message ID matches msg_01{22} format.
func checkIDFormat(body map[string]any) CheckResult {
	id, _ := body["id"].(string)
	if id == "" {
		return CheckResult{Name: "id_format", Pass: false, Detail: "no id field", Fix: "IDRewrite"}
	}
	if msgIDRe.MatchString(id) {
		return CheckResult{Name: "id_format", Pass: true, Detail: "msg_01 format OK"}
	}
	return CheckResult{Name: "id_format", Pass: false, Detail: "got " + truncate(id, 30), Fix: "IDRewrite"}
}

// checkToolUseID verifies tool_use content blocks have toolu_01{22} IDs.
func checkToolUseID(body map[string]any) CheckResult {
	content, _ := body["content"].([]any)
	for _, cb := range content {
		m, ok := cb.(map[string]any)
		if !ok {
			continue
		}
		t, _ := m["type"].(string)
		if t == "tool_use" {
			id, _ := m["id"].(string)
			if id == "" {
				return CheckResult{Name: "tool_use_id", Pass: false, Detail: "tool_use has no id", Fix: "IDRewrite"}
			}
			if !toolIDRe.MatchString(id) {
				return CheckResult{Name: "tool_use_id", Pass: false, Detail: "tool_use id: " + truncate(id, 30), Fix: "IDRewrite"}
			}
			return CheckResult{Name: "tool_use_id", Pass: true, Detail: "toolu_01 format OK"}
		}
		if t == "server_tool_use" {
			id, _ := m["id"].(string)
			if id == "" {
				return CheckResult{Name: "tool_use_id", Pass: false, Detail: "server_tool_use has no id", Fix: "IDRewrite"}
			}
			if !srvToolIDRe.MatchString(id) {
				return CheckResult{Name: "tool_use_id", Pass: false, Detail: "server_tool_use id: " + truncate(id, 30), Fix: "IDRewrite"}
			}
			return CheckResult{Name: "tool_use_id", Pass: true, Detail: "srvtoolu_01 format OK"}
		}
	}
	return CheckResult{Name: "tool_use_id", Pass: true, Detail: "no tool_use blocks (skip)"}
}

// checkSignature checks thinking.signature is a valid protobuf blob.
func checkSignature(body map[string]any) CheckResult {
	content, _ := body["content"].([]any)
	for _, cb := range content {
		m, ok := cb.(map[string]any)
		if !ok {
			continue
		}
		t, _ := m["type"].(string)
		if t != "thinking" {
			continue
		}
		sig, _ := m["signature"].(string)
		if sig == "" {
			return CheckResult{Name: "signature", Pass: false, Detail: "thinking block has no signature", Fix: "SignatureRewrite"}
		}
		raw, err := base64.StdEncoding.DecodeString(sig)
		if err != nil {
			return CheckResult{Name: "signature", Pass: false, Detail: "signature not valid base64", Fix: "SignatureRewrite"}
		}
		if len(raw) < 4 || raw[0] != 0x12 {
			return CheckResult{Name: "signature", Pass: false, Detail: "signature protobuf tag mismatch", Fix: "SignatureRewrite"}
		}
		ok2, _ := fingerprint.VerifySignature(sig)
		if ok2 {
			return CheckResult{Name: "signature", Pass: true, Detail: "gateway-issued signature verified"}
		}
		return CheckResult{Name: "signature", Pass: true, Detail: "valid protobuf signature (non-gateway)"}
	}
	return CheckResult{Name: "signature", Pass: false, Detail: "no thinking block found", Fix: "SignatureRewrite"}
}

// checkUsageStructure verifies usage has cache_creation nested object and proper fields.
func checkUsageStructure(body map[string]any) CheckResult {
	usage, _ := body["usage"].(map[string]any)
	if usage == nil {
		return CheckResult{Name: "usage_structure", Pass: false, Detail: "no usage object", Fix: "BodyRewrite"}
	}
	cc, hasCC := usage["cache_creation"].(map[string]any)
	if !hasCC {
		return CheckResult{Name: "usage_structure", Pass: false, Detail: "missing cache_creation nested object", Fix: "BodyRewrite"}
	}
	if _, ok := cc["ephemeral_5m_input_tokens"]; !ok {
		return CheckResult{Name: "usage_structure", Pass: false, Detail: "missing ephemeral_5m_input_tokens in cache_creation", Fix: "BodyRewrite"}
	}
	if _, ok := cc["ephemeral_1h_input_tokens"]; !ok {
		return CheckResult{Name: "usage_structure", Pass: false, Detail: "missing ephemeral_1h_input_tokens in cache_creation", Fix: "BodyRewrite"}
	}
	if _, ok := usage["service_tier"]; !ok {
		return CheckResult{Name: "usage_structure", Pass: false, Detail: "missing service_tier", Fix: "BodyRewrite"}
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
		return CheckResult{Name: "field_order", Pass: false, Detail: "missing model or id", Fix: "BodyRewrite"}
	}
	if modelIdx > idIdx {
		return CheckResult{Name: "field_order", Pass: false, Detail: "id appears before model", Fix: "BodyRewrite"}
	}
	// content should appear before stop_reason
	contentIdx := strings.Index(target, `"content"`)
	stopIdx := strings.Index(target, `"stop_reason"`)
	if contentIdx > 0 && stopIdx > 0 && contentIdx > stopIdx {
		return CheckResult{Name: "field_order", Pass: false, Detail: "stop_reason before content", Fix: "BodyRewrite"}
	}
	return CheckResult{Name: "field_order", Pass: true, Detail: "field order OK"}
}

// checkHeaders verifies Anthropic-style ratelimit and org headers.
func checkHeaders(headers http.Header) CheckResult {
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
		return CheckResult{Name: "headers", Pass: true, Detail: "all ratelimit headers present"}
	}
	return CheckResult{Name: "headers", Pass: false,
		Detail: fmt.Sprintf("missing %d/%d ratelimit headers", missing, len(required)),
		Fix:    "HeadersFake"}
}

// checkRequestID verifies Request-Id header matches req_01 format.
func checkRequestID(headers http.Header) CheckResult {
	rid := headers.Get("Request-Id")
	if rid == "" {
		return CheckResult{Name: "request_id", Pass: false, Detail: "no Request-Id header", Fix: "HeadersFake"}
	}
	if reqIDRe.MatchString(rid) {
		return CheckResult{Name: "request_id", Pass: true, Detail: "Request-Id format OK: " + truncate(rid, 20)}
	}
	return CheckResult{Name: "request_id", Pass: false, Detail: "Request-Id not req_01 format: " + truncate(rid, 20), Fix: "HeadersFake"}
}

// checkXNewApiVersion checks for the X-New-Api-Version header (indicates non-official).
func checkXNewApiVersion(headers http.Header) CheckResult {
	if headers.Get("X-New-Api-Version") != "" {
		return CheckResult{Name: "x_new_api_version", Pass: false,
			Detail: "X-New-Api-Version header present (non-official)", Fix: "HeadersFake"}
	}
	return CheckResult{Name: "x_new_api_version", Pass: true, Detail: "no X-New-Api-Version header"}
}

// checkSSEDone checks if the SSE stream ends with [DONE] sentinel.
func checkSSEDone(sseData string) CheckResult {
	if strings.Contains(sseData, "data: [DONE]") {
		return CheckResult{Name: "sse_done", Pass: false, Detail: "[DONE] sentinel found in stream", Fix: "StripDone"}
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
		return CheckResult{Name: "sse_event_order", Pass: false, Detail: "no SSE events parsed", Fix: "BodyRewrite"}
	}
	// First event must be message_start
	if events[0] != "message_start" {
		return CheckResult{Name: "sse_event_order", Pass: false, Detail: "first event is " + events[0] + " not message_start", Fix: "BodyRewrite"}
	}
	// Last event must be message_stop
	if events[len(events)-1] != "message_stop" {
		return CheckResult{Name: "sse_event_order", Pass: false, Detail: "last event is " + events[len(events)-1] + " not message_stop", Fix: "BodyRewrite"}
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
		return CheckResult{Name: "sse_event_order", Pass: false, Detail: "no ping event in stream", Fix: "BodyRewrite"}
	}
	return CheckResult{Name: "sse_event_order", Pass: true, Detail: fmt.Sprintf("%d events, order OK", len(events))}
}

// checkContainer checks if response body contains a "container" field.
func checkContainer(body map[string]any) CheckResult {
	if _, ok := body["container"]; ok {
		return CheckResult{Name: "container", Pass: false, Detail: "container field present", Fix: "StripContainer"}
	}
	return CheckResult{Name: "container", Pass: true, Detail: "no container field"}
}

// checkBedrockState checks if usage contains bedrock_state.
func checkBedrockState(body map[string]any) CheckResult {
	usage, _ := body["usage"].(map[string]any)
	if usage == nil {
		return CheckResult{Name: "bedrock_state", Pass: true, Detail: "no usage object"}
	}
	if _, ok := usage["bedrock_state"]; ok {
		return CheckResult{Name: "bedrock_state", Pass: false, Detail: "bedrock_state present in usage", Fix: "StripBedrock"}
	}
	return CheckResult{Name: "bedrock_state", Pass: true, Detail: "no bedrock_state"}
}

// checkInferenceGeo checks if inference_geo has a valid value.
func checkInferenceGeo(body map[string]any, model string) CheckResult {
	usage, _ := body["usage"].(map[string]any)
	if usage == nil {
		return CheckResult{Name: "inference_geo", Pass: false, Detail: "no usage object", Fix: "ForceGeo"}
	}
	geo, _ := usage["inference_geo"].(string)
	expected := fingerprint.GeoForModel(model)
	if geo == "" {
		return CheckResult{Name: "inference_geo", Pass: false, Detail: "missing inference_geo", Fix: "ForceGeo"}
	}
	if geo != expected {
		return CheckResult{Name: "inference_geo", Pass: false,
			Detail: "inference_geo=" + geo + " expected=" + expected, Fix: "ForceGeo"}
	}
	return CheckResult{Name: "inference_geo", Pass: true, Detail: "inference_geo=" + geo}
}

// checkStopDetails checks if stop_details field exists in the response.
func checkStopDetails(body map[string]any) CheckResult {
	if _, ok := body["stop_details"]; ok {
		return CheckResult{Name: "stop_details", Pass: true, Detail: "stop_details present"}
	}
	return CheckResult{Name: "stop_details", Pass: false, Detail: "stop_details missing", Fix: "BodyRewrite"}
}

// checkStopDetailsStructure verifies stop_details.type matches stop_reason.
// Official API: stop_details: {type: "end_turn"|"stop_sequence"|"max_tokens"}
func checkStopDetailsStructure(body map[string]any) CheckResult {
	sd, ok := body["stop_details"].(map[string]any)
	if !ok {
		// stop_details might be null — that's a separate check
		return CheckResult{Name: "stop_details_structure", Pass: true, Detail: "stop_details null (skip)"}
	}
	sdType, _ := sd["type"].(string)
	if sdType == "" {
		return CheckResult{Name: "stop_details_structure", Pass: false,
			Detail: "stop_details has no type field", Fix: "BodyRewrite"}
	}
	sr, _ := body["stop_reason"].(string)
	if sr != "" && sdType != sr {
		return CheckResult{Name: "stop_details_structure", Pass: false,
			Detail: fmt.Sprintf("stop_details.type=%s != stop_reason=%s", sdType, sr), Fix: "BodyRewrite"}
	}
	return CheckResult{Name: "stop_details_structure", Pass: true,
		Detail: "stop_details.type=" + sdType + " matches stop_reason"}
}

// checkThinkingPresent checks if a thinking block exists when thinking was requested.
func checkThinkingPresent(body map[string]any) CheckResult {
	content, _ := body["content"].([]any)
	for _, cb := range content {
		m, ok := cb.(map[string]any)
		if !ok {
			continue
		}
		t, _ := m["type"].(string)
		if t == "thinking" || t == "redacted_thinking" {
			return CheckResult{Name: "thinking_present", Pass: true, Detail: "thinking block found"}
		}
	}
	return CheckResult{Name: "thinking_present", Pass: false, Detail: "no thinking block in response", Fix: "ThinkingInject"}
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
			Fix:    "SmallProbeZero"}
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
			Detail: "cache_control used but cache all zero", Fix: "CacheFake"}
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
					Detail: "web_search_tool_result tool_use_id not srvtoolu_01 format: " + truncate(tid, 20), Fix: "IDRewrite"}
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
							Detail: "web_search_result missing encrypted_content", Fix: "SignatureRewrite"}
					}
					// Verify it's valid base64
					_, err := base64.StdEncoding.DecodeString(ec)
					if err != nil {
						return CheckResult{Name: "web_search_result", Pass: false,
							Detail: "encrypted_content not valid base64", Fix: "SignatureRewrite"}
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
		return CheckResult{Name: "model_name", Pass: false, Detail: "no model field in response", Fix: "BodyRewrite"}
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
	return CheckResult{Name: "model_name", Pass: false, Detail: "model=" + model + " expected=" + expectedModel, Fix: "BodyRewrite"}
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
	var missing []string
	if headers.Get("Cf-Ray") == "" {
		missing = append(missing, "Cf-Ray")
	}
	if headers.Get("Cf-Cache-Status") == "" {
		missing = append(missing, "Cf-Cache-Status")
	}
	// Server should be cloudflare-like, not upstream leaks
	server := headers.Get("Server")
	if server == "" {
		missing = append(missing, "Server")
	}
	// Set-Cookie with _cfuvid pattern
	cookie := headers.Get("Set-Cookie")
	if cookie == "" || !strings.Contains(cookie, "_cfuvid") {
		missing = append(missing, "Set-Cookie(_cfuvid)")
	}
	if len(missing) == 0 {
		return CheckResult{Name: "cf_headers", Pass: true, Detail: "Cf-Ray, Server, Set-Cookie all present"}
	}
	return CheckResult{Name: "cf_headers", Pass: false,
		Detail: "missing: " + strings.Join(missing, ", "), Fix: "HeadersFake"}
}

// checkThinkingOrder verifies thinking block appears before text block in content.
// Official API always puts thinking at index 0 when thinking is enabled.
func checkThinkingOrder(body map[string]any) CheckResult {
	content, _ := body["content"].([]any)
	if len(content) == 0 {
		return CheckResult{Name: "thinking_order", Pass: true, Detail: "no content blocks"}
	}
	firstType := ""
	hasThinking := false
	for i, cb := range content {
		m, ok := cb.(map[string]any)
		if !ok {
			continue
		}
		t, _ := m["type"].(string)
		if i == 0 {
			firstType = t
		}
		if t == "thinking" || t == "redacted_thinking" {
			hasThinking = true
			if i > 0 {
				return CheckResult{Name: "thinking_order", Pass: false,
					Detail: fmt.Sprintf("thinking at index %d, should be 0", i), Fix: "ThinkingInject"}
			}
		}
	}
	if !hasThinking {
		return CheckResult{Name: "thinking_order", Pass: true, Detail: "no thinking block (skip)"}
	}
	if firstType == "thinking" || firstType == "redacted_thinking" {
		return CheckResult{Name: "thinking_order", Pass: true, Detail: "thinking at index 0"}
	}
	return CheckResult{Name: "thinking_order", Pass: false, Detail: "first block is " + firstType + " not thinking", Fix: "ThinkingInject"}
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
			return CheckResult{Name: "delta_usage_slim", Pass: false, Detail: "no usage in message_delta", Fix: "BodyRewrite"}
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
				Detail: "message_delta usage has full fields: " + strings.Join(found, ", "), Fix: "BodyRewrite"}
		}
		return CheckResult{Name: "delta_usage_slim", Pass: true, Detail: "message_delta usage is slim format"}
	}
	return CheckResult{Name: "delta_usage_slim", Pass: true, Detail: "no message_delta event found (skip)"}
}

// checkStopReason verifies stop_reason is a valid value.
func checkStopReason(body map[string]any) CheckResult {
	sr, _ := body["stop_reason"].(string)
	if sr == "" {
		return CheckResult{Name: "stop_reason", Pass: false, Detail: "stop_reason is empty/null", Fix: "BodyRewrite"}
	}
	valid := map[string]bool{
		"end_turn":      true,
		"max_tokens":    true,
		"stop_sequence": true,
		"tool_use":      true,
		"refusal":       true,
	}
	if valid[sr] {
		return CheckResult{Name: "stop_reason", Pass: true, Detail: "stop_reason=" + sr}
	}
	return CheckResult{Name: "stop_reason", Pass: false, Detail: "unexpected stop_reason=" + sr, Fix: "BodyRewrite"}
}

// checkToolStopReason verifies stop_reason is "tool_use" when tool_choice forced a tool.
func checkToolStopReason(body map[string]any) CheckResult {
	sr, _ := body["stop_reason"].(string)
	if sr == "tool_use" {
		return CheckResult{Name: "tool_stop_reason", Pass: true, Detail: "stop_reason=tool_use OK"}
	}
	if sr == "" {
		return CheckResult{Name: "tool_stop_reason", Pass: false,
			Detail: "stop_reason empty, expected tool_use", Fix: "BodyRewrite"}
	}
	return CheckResult{Name: "tool_stop_reason", Pass: false,
		Detail: "stop_reason=" + sr + " expected tool_use", Fix: "BodyRewrite"}
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
		checks = append(checks, CheckResult{Name: "small_output_tokens", Pass: false, Detail: "no usage object", Fix: "BodyRewrite"})
		return checks
	}

	// output_tokens must be exactly 1 for max_tokens=1
	outTok := fingerprint.IntVal(usage, "output_tokens")
	if outTok == 1 {
		checks = append(checks, CheckResult{Name: "small_output_tokens", Pass: true, Detail: "output_tokens=1 OK"})
	} else {
		checks = append(checks, CheckResult{Name: "small_output_tokens", Pass: false,
			Detail: fmt.Sprintf("output_tokens=%d expected 1", outTok), Fix: "BodyRewrite"})
	}

	// stop_reason must be "max_tokens"
	sr, _ := body["stop_reason"].(string)
	if sr == "max_tokens" {
		checks = append(checks, CheckResult{Name: "small_stop_reason", Pass: true, Detail: "stop_reason=max_tokens OK"})
	} else {
		checks = append(checks, CheckResult{Name: "small_stop_reason", Pass: false,
			Detail: "stop_reason=" + sr + " expected max_tokens", Fix: "BodyRewrite"})
	}

	// cache_creation nested: ephemeral values must be 0
	cc, hasCC := usage["cache_creation"].(map[string]any)
	if !hasCC {
		checks = append(checks, CheckResult{Name: "small_ephemeral_zero", Pass: false,
			Detail: "no cache_creation nested object", Fix: "BodyRewrite"})
	} else {
		e5m := fingerprint.IntVal(cc, "ephemeral_5m_input_tokens")
		e1h := fingerprint.IntVal(cc, "ephemeral_1h_input_tokens")
		if e5m == 0 && e1h == 0 {
			checks = append(checks, CheckResult{Name: "small_ephemeral_zero", Pass: true,
				Detail: "ephemeral values both 0 OK"})
		} else {
			checks = append(checks, CheckResult{Name: "small_ephemeral_zero", Pass: false,
				Detail: fmt.Sprintf("ephemeral_5m=%d ephemeral_1h=%d should be 0", e5m, e1h), Fix: "SmallProbeZero"})
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
			Detail: fmt.Sprintf("cache create=%d read=%d should be 0", ccCreate, ccRead), Fix: "SmallProbeZero"})
	}

	return checks
}

// checkBackendType detects the backend type from the message ID prefix.
// msg_bdrk_ = Bedrock, gen- = OpenRouter, chatcmpl- = OneAPI/sub2api
func checkBackendType(body map[string]any) CheckResult {
	id, _ := body["id"].(string)
	if id == "" {
		return CheckResult{Name: "backend_type", Pass: false, Detail: "no id field"}
	}
	switch {
	case strings.HasPrefix(id, "msg_bdrk_"):
		return CheckResult{Name: "backend_type", Pass: false, Detail: "Bedrock backend (msg_bdrk_)", Fix: "IDRewrite"}
	case strings.HasPrefix(id, "gen-"):
		return CheckResult{Name: "backend_type", Pass: false, Detail: "OpenRouter backend (gen-)", Fix: "IDRewrite"}
	case strings.HasPrefix(id, "chatcmpl-"):
		return CheckResult{Name: "backend_type", Pass: false, Detail: "OneAPI/sub2api backend (chatcmpl-)", Fix: "IDRewrite"}
	case strings.HasPrefix(id, "msg_01"):
		return CheckResult{Name: "backend_type", Pass: true, Detail: "official format (msg_01)"}
	case strings.HasPrefix(id, "msg_"):
		return CheckResult{Name: "backend_type", Pass: true, Detail: "Anthropic format (msg_)"}
	default:
		return CheckResult{Name: "backend_type", Pass: false, Detail: "unknown id prefix: " + truncate(id, 20), Fix: "IDRewrite"}
	}
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
			Detail: "missing fields: " + strings.Join(missing, ", "), Fix: "BodyRewrite"})
	} else {
		checks = append(checks, CheckResult{Name: "nonstream_fields", Pass: true, Detail: "all required fields present"})
	}

	// type must be "message"
	if tp, _ := body["type"].(string); tp != "message" {
		checks = append(checks, CheckResult{Name: "nonstream_type", Pass: false,
			Detail: "type=" + tp + " expected message", Fix: "BodyRewrite"})
	} else {
		checks = append(checks, CheckResult{Name: "nonstream_type", Pass: true, Detail: "type=message OK"})
	}

	// role must be "assistant"
	if role, _ := body["role"].(string); role != "assistant" {
		checks = append(checks, CheckResult{Name: "nonstream_role", Pass: false,
			Detail: "role=" + role + " expected assistant", Fix: "BodyRewrite"})
	} else {
		checks = append(checks, CheckResult{Name: "nonstream_role", Pass: true, Detail: "role=assistant OK"})
	}

	return checks
}

// checkThinkingDisplayOmitted verifies thinking block format for display="omitted" mode.
// On Opus 4.7+, thinking.thinking may be "" (empty) but signature must still be present.
func checkThinkingDisplayOmitted(body map[string]any, model string) CheckResult {
	// Only relevant for opus-4-7 and later
	if !strings.Contains(model, "opus-4-7") && !strings.Contains(model, "opus-4-8") {
		return CheckResult{Name: "thinking_display_omitted", Pass: true, Detail: "not opus-4-7+ model (skip)"}
	}
	content, _ := body["content"].([]any)
	for _, cb := range content {
		m, ok := cb.(map[string]any)
		if !ok {
			continue
		}
		t, _ := m["type"].(string)
		if t != "thinking" {
			continue
		}
		thinkText, _ := m["thinking"].(string)
		sig, _ := m["signature"].(string)
		if sig == "" {
			return CheckResult{Name: "thinking_display_omitted", Pass: false,
				Detail: "opus-4-7 thinking block missing signature", Fix: "SignatureRewrite"}
		}
		// display=omitted: thinking="" is valid; display=summarized: thinking has content
		// Both are valid for opus-4-7, just verify signature is always present
		if thinkText == "" {
			return CheckResult{Name: "thinking_display_omitted", Pass: true,
				Detail: "display=omitted mode: thinking empty, signature present"}
		}
		return CheckResult{Name: "thinking_display_omitted", Pass: true,
			Detail: "display=summarized mode: thinking has content, signature present"}
	}
	return CheckResult{Name: "thinking_display_omitted", Pass: false,
		Detail: "no thinking block found for opus-4-7", Fix: "ThinkingInject"}
}

// checkNoThinkingWhenDisabled verifies no thinking block exists when thinking was NOT requested.
func checkNoThinkingWhenDisabled(body map[string]any) CheckResult {
	content, _ := body["content"].([]any)
	for _, cb := range content {
		m, ok := cb.(map[string]any)
		if !ok {
			continue
		}
		t, _ := m["type"].(string)
		if t == "thinking" || t == "redacted_thinking" {
			return CheckResult{Name: "no_thinking_leak", Pass: false,
				Detail: "thinking block present but thinking was NOT requested", Fix: "BodyRewrite"}
		}
	}
	return CheckResult{Name: "no_thinking_leak", Pass: true, Detail: "no unexpected thinking block"}
}

// checkTagReplay verifies the response contains the echoed tag from the request.
// cctest sends a random tag and expects the LLM to echo it back.
func checkTagReplay(body map[string]any, tag string) CheckResult {
	if tag == "" {
		return CheckResult{Name: "tag_replay", Pass: true, Detail: "no tag to check (skip)"}
	}
	content, _ := body["content"].([]any)
	for _, cb := range content {
		m, ok := cb.(map[string]any)
		if !ok {
			continue
		}
		t, _ := m["type"].(string)
		if t == "text" {
			text, _ := m["text"].(string)
			if strings.Contains(text, tag) {
				return CheckResult{Name: "tag_replay", Pass: true, Detail: "tag found in response: " + tag}
			}
		}
	}
	return CheckResult{Name: "tag_replay", Pass: false, Detail: "tag NOT found in response: " + tag}
}

// checkIdentityResponse verifies the response content mentions Claude or Anthropic.
// cctest sends identity probe questions to check if the LLM knows what it is.
// Also handles structured JSON output (output_config json_schema) where the text
// content is a JSON string containing title/desc fields.
func checkIdentityResponse(body map[string]any) CheckResult {
	text := collectResponseText(body)
	lower := strings.ToLower(text)
	if strings.Contains(lower, "claude") || strings.Contains(lower, "anthropic") {
		return CheckResult{Name: "identity_response", Pass: true, Detail: "response mentions Claude/Anthropic"}
	}
	// Try parsing as JSON (structured output from output_config json_schema)
	if text != "" {
		var structured map[string]any
		if json.Unmarshal([]byte(text), &structured) == nil {
			combined := ""
			for _, v := range structured {
				if s, ok := v.(string); ok {
					combined += " " + s
				}
			}
			combinedLower := strings.ToLower(combined)
			if strings.Contains(combinedLower, "claude") || strings.Contains(combinedLower, "anthropic") {
				return CheckResult{Name: "identity_response", Pass: true, Detail: "structured output mentions Claude/Anthropic"}
			}
		}
	}
	if text == "" {
		return CheckResult{Name: "identity_response", Pass: false, Detail: "no text content in response"}
	}
	return CheckResult{Name: "identity_response", Pass: false, Detail: "response does not mention Claude or Anthropic"}
}

// checkPoisonAnswer verifies the answer to the classic poison bottle problem.
// 1000 bottles, 1 poisoned, 24h, answer = 10 mice (binary encoding).
func checkPoisonAnswer(body map[string]any) CheckResult {
	text := collectResponseText(body)
	if text == "" {
		return CheckResult{Name: "poison_answer", Pass: false, Detail: "no text content"}
	}
	if strings.Contains(text, "10") {
		return CheckResult{Name: "poison_answer", Pass: true, Detail: "contains correct answer (10 mice)"}
	}
	return CheckResult{Name: "poison_answer", Pass: false, Detail: "answer 10 not found in response"}
}

// checkLogicAnswer verifies the answer to the 3-switch puzzle.
// The correct method involves turning switches on, waiting, then checking heat.
func checkLogicAnswer(body map[string]any) CheckResult {
	text := collectResponseText(body)
	lower := strings.ToLower(text)
	if text == "" {
		return CheckResult{Name: "logic_answer", Pass: false, Detail: "no text content"}
	}
	// The answer involves heat/temperature/warm — checking if bulb is warm
	heatKeywords := []string{"热", "温度", "摸", "warm", "heat", "hot", "touch"}
	for _, kw := range heatKeywords {
		if strings.Contains(lower, kw) {
			return CheckResult{Name: "logic_answer", Pass: true, Detail: "contains heat/warm method"}
		}
	}
	return CheckResult{Name: "logic_answer", Pass: false, Detail: "heat method not found in response"}
}

// collectResponseText extracts all text content from a response body.
func collectResponseText(body map[string]any) string {
	content, _ := body["content"].([]any)
	var parts []string
	for _, cb := range content {
		m, ok := cb.(map[string]any)
		if !ok {
			continue
		}
		t, _ := m["type"].(string)
		if t == "text" {
			if txt, _ := m["text"].(string); txt != "" {
				parts = append(parts, txt)
			}
		}
	}
	return strings.Join(parts, "\n")
}

// checkStructuredJSONValid verifies the response text is valid JSON when output_config is used.
func checkStructuredJSONValid(body map[string]any) CheckResult {
	text := collectResponseText(body)
	if text == "" {
		return CheckResult{Name: "structured_json_valid", Pass: false, Detail: "no text content", Fix: "BodyRewrite"}
	}
	var parsed any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return CheckResult{Name: "structured_json_valid", Pass: false,
			Detail: "response is not valid JSON: " + truncate(err.Error(), 60), Fix: "BodyRewrite"}
	}
	return CheckResult{Name: "structured_json_valid", Pass: true, Detail: "valid JSON response"}
}

// checkStructuredSchemaMatch verifies the JSON response matches the expected schema {title: string, desc: string}.
func checkStructuredSchemaMatch(body map[string]any) CheckResult {
	text := collectResponseText(body)
	if text == "" {
		return CheckResult{Name: "structured_schema_match", Pass: false, Detail: "no text content", Fix: "BodyRewrite"}
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(text), &obj); err != nil {
		return CheckResult{Name: "structured_schema_match", Pass: false,
			Detail: "not a JSON object", Fix: "BodyRewrite"}
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
			Detail: "missing required fields: " + strings.Join(missing, ", "), Fix: "BodyRewrite"}
	}
	if title == "" || desc == "" {
		return CheckResult{Name: "structured_schema_match", Pass: false,
			Detail: "title or desc is empty", Fix: "BodyRewrite"}
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
			Detail: fmt.Sprintf("schema has %d extra fields (additionalProperties should be false)", extra), Fix: "BodyRewrite"}
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
			Detail: "stop_reason is empty/null", Fix: "BodyRewrite"}
	}
	if sr == "end_turn" {
		return CheckResult{Name: "structured_stop_reason", Pass: true, Detail: "stop_reason=end_turn OK"}
	}
	if sr == "refusal" {
		return CheckResult{Name: "structured_stop_reason", Pass: true, Detail: "stop_reason=refusal (safety)"}
	}
	return CheckResult{Name: "structured_stop_reason", Pass: false,
		Detail: "stop_reason=" + sr + " expected end_turn for structured output", Fix: "BodyRewrite"}
}

// checkImageOCR verifies the model can read text from an image.
// The test image contains known text that should be echoed back.
func checkImageOCR(body map[string]any) CheckResult {
	text := strings.TrimSpace(strings.ToUpper(collectResponseText(body)))
	if text == "" {
		return CheckResult{Name: "image_ocr", Pass: false, Detail: "no text content in response"}
	}
	// The image contains repeating pattern text — check if any substantial text was extracted
	// Since the exact OCR result depends on the image content, we just verify non-empty response
	// with reasonable length (the image has visible text)
	if len(text) >= 2 {
		return CheckResult{Name: "image_ocr", Pass: true, Detail: "image OCR returned text: " + truncate(text, 40)}
	}
	return CheckResult{Name: "image_ocr", Pass: false, Detail: "OCR response too short: " + truncate(text, 40)}
}

// checkPDFExtract verifies the model can extract text from a PDF document.
// The test PDF contains "BYPWNXWP".
func checkPDFExtract(body map[string]any) CheckResult {
	text := strings.TrimSpace(strings.ToUpper(collectResponseText(body)))
	if text == "" {
		return CheckResult{Name: "pdf_extract", Pass: false, Detail: "no text content in response"}
	}
	if strings.Contains(text, "BYPWNXWP") {
		return CheckResult{Name: "pdf_extract", Pass: true, Detail: "PDF text correctly extracted: BYPWNXWP"}
	}
	return CheckResult{Name: "pdf_extract", Pass: false, Detail: "expected BYPWNXWP, got: " + truncate(text, 40)}
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
				Detail: "message_start has no message object", Fix: "BodyRewrite"}
		}
		usage, _ := msg["usage"].(map[string]any)
		if usage == nil {
			return CheckResult{Name: "message_start_usage", Pass: false,
				Detail: "message_start.message has no usage", Fix: "BodyRewrite"}
		}
		// Must have input_tokens
		if _, ok := usage["input_tokens"]; !ok {
			return CheckResult{Name: "message_start_usage", Pass: false,
				Detail: "message_start usage missing input_tokens", Fix: "BodyRewrite"}
		}
		return CheckResult{Name: "message_start_usage", Pass: true,
			Detail: fmt.Sprintf("message_start usage OK: input_tokens=%d", fingerprint.IntVal(usage, "input_tokens"))}
	}
	return CheckResult{Name: "message_start_usage", Pass: true, Detail: "no message_start event (skip)"}
}

// checkServerTiming verifies Server-Timing header exists (envoy upstream service time).
func checkServerTiming(headers http.Header) CheckResult {
	st := headers.Get("Server-Timing")
	if st == "" {
		return CheckResult{Name: "server_timing", Pass: false, Detail: "no Server-Timing header", Fix: "HeadersFake"}
	}
	if strings.Contains(st, "x-originResponse") {
		return CheckResult{Name: "server_timing", Pass: true, Detail: "Server-Timing OK: " + truncate(st, 40)}
	}
	return CheckResult{Name: "server_timing", Pass: false, Detail: "Server-Timing format unexpected: " + truncate(st, 40), Fix: "HeadersFake"}
}

// --- helpers ---

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
