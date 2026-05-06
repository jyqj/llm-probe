package channeltest

import (
	"fmt"

	"detector-service/internal/fingerprint"
)

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
