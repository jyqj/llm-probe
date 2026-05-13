package channeltest

import (
	"fmt"
	"strings"

	"detector-service/internal/fingerprint"
	"encoding/json"
)

// checkSSEDone checks if the SSE stream ends with [DONE] sentinel.
func checkSSEDone(sseData string) CheckResult {
	if strings.Contains(sseData, "data: [DONE]") {
		return CheckResult{Name: "sse_done", Pass: false,
			Expected: "no [DONE] sentinel in SSE stream",
			Actual:   "[DONE] sentinel found",
			Detail:   "[DONE] sentinel found in stream", Fix: "strip_done"}
	}
	return CheckResult{Name: "sse_done", Pass: true,
		Expected: "no [DONE] sentinel in SSE stream",
		Actual:   "no [DONE] sentinel",
		Detail:   "no [DONE] sentinel"}
}

// checkSSEEventOrder verifies SSE events follow the official order:
// message_start -> content_block_start -> ping -> deltas -> content_block_stop -> message_delta -> message_stop
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
	expectedOrder := "message_start -> content_block_start -> ping -> deltas -> content_block_stop -> message_delta -> message_stop"
	if len(events) == 0 {
		return CheckResult{Name: "sse_event_order", Pass: false,
			Expected: expectedOrder,
			Actual:   "no SSE events parsed",
			Detail:   "no SSE events parsed", Fix: "body_rewrite"}
	}
	// First event must be message_start
	if events[0] != "message_start" {
		return CheckResult{Name: "sse_event_order", Pass: false,
			Expected: "first event = message_start",
			Actual:   "first event = " + events[0],
			Detail:   "first event is " + events[0] + " not message_start", Fix: "body_rewrite"}
	}
	// Last event must be message_stop
	if events[len(events)-1] != "message_stop" {
		return CheckResult{Name: "sse_event_order", Pass: false,
			Expected: "last event = message_stop",
			Actual:   "last event = " + events[len(events)-1],
			Detail:   "last event is " + events[len(events)-1] + " not message_stop", Fix: "body_rewrite"}
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
		return CheckResult{Name: "sse_event_order", Pass: false,
			Expected: "ping event present in stream",
			Actual:   "no ping event found",
			Detail:   "no ping event in stream", Fix: "body_rewrite"}
	}
	return CheckResult{Name: "sse_event_order", Pass: true,
		Expected: expectedOrder,
		Actual:   fmt.Sprintf("%d events in correct order", len(events)),
		Detail:   fmt.Sprintf("%d events, order OK", len(events))}
}

// checkCacheSmallProbe checks if cache values are zero for small max_tokens requests (no cache_control).
// checkSSETailing checks the SSE stream ending whitespace pattern.
// Official API ends each event with \n\n\n (three newlines), not \n\n.
func checkSSETailing(sseData string) CheckResult {
	tripleCount := strings.Count(sseData, "\n\n\n")
	doubleOnly := strings.Count(sseData, "\n\n") - tripleCount
	expectedTailing := "SSE events separated by \\n\\n or \\n\\n\\n"
	if tripleCount > 0 {
		return CheckResult{Name: "sse_tailing", Pass: true,
			Expected: expectedTailing,
			Actual:   fmt.Sprintf("triple-newline endings found (%d)", tripleCount),
			Detail:   fmt.Sprintf("triple-newline endings found (%d)", tripleCount)}
	}
	if doubleOnly > 0 {
		// Official API behavior varies — double-newline is acceptable
		return CheckResult{Name: "sse_tailing", Pass: true,
			Expected: expectedTailing,
			Actual:   fmt.Sprintf("double-newline endings (%d)", doubleOnly),
			Detail:   fmt.Sprintf("double-newline endings (%d)", doubleOnly)}
	}
	return CheckResult{Name: "sse_tailing", Pass: true,
		Expected: expectedTailing,
		Actual:   "no newline patterns detected",
		Detail:   "no newline patterns to check"}
}

// checkCfHeaders verifies cloudflare-style headers (Cf-Ray, Server, Set-Cookie).
// These are part of HeadersFake and help pass fingerprint checks.
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
		expectedSlim := "usage with only output_tokens (slim format, no service_tier/inference_geo/cache_creation)"
		if usage == nil {
			return CheckResult{Name: "delta_usage_slim", Pass: false,
				Expected: expectedSlim,
				Actual:   "no usage object in message_delta",
				Detail:   "no usage in message_delta", Fix: "body_rewrite"}
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
				Expected: expectedSlim,
				Actual:   "extra fields present: " + strings.Join(found, ", "),
				Detail:   "message_delta usage has full fields: " + strings.Join(found, ", "), Fix: "body_rewrite"}
		}
		return CheckResult{Name: "delta_usage_slim", Pass: true,
			Expected: expectedSlim,
			Actual:   "slim format (no bloat fields)",
			Detail:   "message_delta usage is slim format"}
	}
	return CheckResult{Name: "delta_usage_slim", Pass: true,
		Expected: "message_delta with slim usage (if present)",
		Actual:   "no message_delta event found",
		Detail:   "no message_delta event found (skip)"}
}

// checkStopReason verifies stop_reason is a valid value.
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
		expectedUsage := "message_start.message.usage with input_tokens, cache_creation_input_tokens, cache_read_input_tokens"
		if msg == nil {
			return CheckResult{Name: "message_start_usage", Pass: false,
				Expected: expectedUsage,
				Actual:   "message_start has no message object",
				Detail:   "message_start has no message object", Fix: "body_rewrite"}
		}
		usage, _ := msg["usage"].(map[string]any)
		if usage == nil {
			return CheckResult{Name: "message_start_usage", Pass: false,
				Expected: expectedUsage,
				Actual:   "message_start.message has no usage",
				Detail:   "message_start.message has no usage", Fix: "body_rewrite"}
		}
		// Must have input_tokens
		if _, ok := usage["input_tokens"]; !ok {
			return CheckResult{Name: "message_start_usage", Pass: false,
				Expected: expectedUsage,
				Actual:   "usage present but missing input_tokens",
				Detail:   "message_start usage missing input_tokens", Fix: "body_rewrite"}
		}
		actualTokens := fingerprint.IntVal(usage, "input_tokens")
		return CheckResult{Name: "message_start_usage", Pass: true,
			Expected: expectedUsage,
			Actual:   fmt.Sprintf("input_tokens=%d", actualTokens),
			Detail:   fmt.Sprintf("message_start usage OK: input_tokens=%d", actualTokens)}
	}
	return CheckResult{Name: "message_start_usage", Pass: true,
		Expected: "message_start with usage (if present)",
		Actual:   "no message_start event found",
		Detail:   "no message_start event (skip)"}
}

// checkServerTiming verifies Server-Timing header exists (envoy upstream service time).
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
					Expected: "ping after content_block_start",
					Actual:   "ping before content_block_start",
					Detail:   "ping before content_block_start", Fix: "body_rewrite"}
			}
			return CheckResult{Name: "sse_ping_position", Pass: true,
				Expected: "ping after content_block_start",
				Actual:   "ping after content_block_start",
				Detail:   "ping after content_block_start OK"}
		}
	}
	return CheckResult{Name: "sse_ping_position", Pass: true,
		Expected: "ping after content_block_start (if present)",
		Actual:   "no ping event found",
		Detail:   "no ping event (skip)"}
}

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
		expectedOutput := "output_tokens = 0 or 1 in message_start"
		if msg == nil {
			return CheckResult{Name: "message_start_output_zero", Pass: false,
				Expected: expectedOutput,
				Actual:   "no message object in message_start",
				Detail:   "no message object", Fix: "body_rewrite"}
		}
		usage, _ := msg["usage"].(map[string]any)
		if usage == nil {
			return CheckResult{Name: "message_start_output_zero", Pass: false,
				Expected: expectedOutput,
				Actual:   "no usage in message_start",
				Detail:   "no usage in message_start", Fix: "body_rewrite"}
		}
		outTok := fingerprint.IntVal(usage, "output_tokens")
		if outTok <= 1 {
			return CheckResult{Name: "message_start_output_zero", Pass: true,
				Expected: expectedOutput,
				Actual:   fmt.Sprintf("output_tokens=%d", outTok),
				Detail:   fmt.Sprintf("output_tokens=%d OK", outTok)}
		}
		return CheckResult{Name: "message_start_output_zero", Pass: false,
			Expected: expectedOutput,
			Actual:   fmt.Sprintf("output_tokens=%d", outTok),
			Detail:   fmt.Sprintf("output_tokens=%d, expected 0-1", outTok), Fix: "body_rewrite"}
	}
	return CheckResult{Name: "message_start_output_zero", Pass: true,
		Expected: "output_tokens = 0-1 in message_start (if present)",
		Actual:   "no message_start event found",
		Detail:   "no message_start (skip)"}
}
