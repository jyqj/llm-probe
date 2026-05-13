package channeltest

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"detector-service/internal/fingerprint"
)

var (
	msgIDRe     = regexp.MustCompile(`^msg_01[0-9A-Za-z]{22}$`)
	toolIDRe    = regexp.MustCompile(`^toolu_01[0-9A-Za-z]{22}$`)
	srvToolIDRe = regexp.MustCompile(`^srvtoolu_01[0-9A-Za-z]{22}$`)
	reqIDRe     = regexp.MustCompile(`^req_01[0-9A-Za-z]+$`)
	uuidRe      = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
)

// checkIDFormat verifies the message ID matches msg_01{22} format.
func checkIDFormat(body map[string]any) CheckResult {
	id, _ := body["id"].(string)
	if id == "" {
		return CheckResult{Name: "id_format", Pass: false, Expected: "msg_01{22} 格式 ID", Actual: "id 字段为空", Detail: "no id field", Fix: "id_rewrite"}
	}
	if msgIDRe.MatchString(id) {
		return CheckResult{Name: "id_format", Pass: true, Expected: "msg_01 前缀", Actual: "msg_01 前缀", Detail: "msg_01 format OK"}
	}
	return CheckResult{Name: "id_format", Pass: false, Expected: "msg_01 前缀", Actual: truncate(id, 30), Detail: "got " + truncate(id, 30), Fix: "id_rewrite"}
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
				return CheckResult{Name: "tool_use_id", Pass: false, Expected: "toolu_01{22} 格式 ID", Actual: "id 字段为空", Detail: "tool_use has no id", Fix: "id_rewrite"}
			}
			if !toolIDRe.MatchString(id) {
				return CheckResult{Name: "tool_use_id", Pass: false, Expected: "toolu_01 前缀", Actual: truncate(id, 30), Detail: "tool_use id: " + truncate(id, 30), Fix: "id_rewrite"}
			}
			return CheckResult{Name: "tool_use_id", Pass: true, Expected: "toolu_01 前缀", Actual: "toolu_01 前缀", Detail: "toolu_01 format OK"}
		}
		if t == "server_tool_use" {
			id, _ := m["id"].(string)
			if id == "" {
				return CheckResult{Name: "tool_use_id", Pass: false, Expected: "srvtoolu_01{22} 格式 ID", Actual: "id 字段为空", Detail: "server_tool_use has no id", Fix: "id_rewrite"}
			}
			if !srvToolIDRe.MatchString(id) {
				return CheckResult{Name: "tool_use_id", Pass: false, Expected: "srvtoolu_01 前缀", Actual: truncate(id, 30), Detail: "server_tool_use id: " + truncate(id, 30), Fix: "id_rewrite"}
			}
			return CheckResult{Name: "tool_use_id", Pass: true, Expected: "srvtoolu_01 前缀", Actual: "srvtoolu_01 前缀", Detail: "srvtoolu_01 format OK"}
		}
	}
	return CheckResult{Name: "tool_use_id", Pass: true, Expected: "tool_use 块存在时使用 toolu_01 前缀", Actual: "无 tool_use 块", Detail: "no tool_use blocks (skip)"}
}

// checkRequestID verifies Request-Id header matches req_01 format.
func checkRequestID(headers http.Header) CheckResult {
	rid := headers.Get("Request-Id")
	if rid != "" {
		if reqIDRe.MatchString(rid) {
			return CheckResult{Name: "request_id", Pass: true, Expected: "req_01 前缀", Actual: "req_01 前缀", Detail: "Request-Id format OK: " + truncate(rid, 20)}
		}
		return CheckResult{Name: "request_id", Pass: false, Expected: "req_01 前缀", Actual: truncate(rid, 20), Detail: "Request-Id not req_01 format: " + truncate(rid, 20), Fix: "headers_fake"}
	}

	// Azure/managed clean channels expose UUID X-Request-Id instead.
	xrid := headers.Get("X-Request-Id")
	if xrid != "" && uuidRe.MatchString(xrid) {
		return CheckResult{Name: "request_id", Pass: true, Expected: "req_01 或 UUID 格式", Actual: "UUID 格式", Detail: "X-Request-Id UUID format OK: " + truncate(xrid, 20)}
	}
	return CheckResult{Name: "request_id", Pass: false, Expected: "Request-Id 或 X-Request-Id 头", Actual: "两者均缺失", Detail: "no Request-Id/X-Request-Id header", Fix: "headers_fake"}
}

// checkXNewApiVersion checks for the X-New-Api-Version header (indicates non-official).
func checkXNewApiVersion(headers http.Header) CheckResult {
	if headers.Get("X-New-Api-Version") != "" {
		return CheckResult{Name: "x_new_api_version", Pass: false, Expected: "无 X-New-Api-Version 头", Actual: "存在 X-New-Api-Version 头",
			Detail: "X-New-Api-Version header present (non-official)", Fix: "headers_fake"}
	}
	return CheckResult{Name: "x_new_api_version", Pass: true, Expected: "无 X-New-Api-Version 头", Actual: "无 X-New-Api-Version 头", Detail: "no X-New-Api-Version header"}
}

// checkContainer checks if response body contains a "container" field.
func checkContainer(body map[string]any) CheckResult {
	if _, ok := body["container"]; ok {
		return CheckResult{Name: "container", Pass: false, Expected: "无 container 字段", Actual: "存在 container 字段", Detail: "container field present", Fix: "strip_container"}
	}
	return CheckResult{Name: "container", Pass: true, Expected: "无 container 字段", Actual: "无 container 字段", Detail: "no container field"}
}

// checkBedrockState checks if usage contains bedrock_state.
func checkBedrockState(body map[string]any) CheckResult {
	usage, _ := body["usage"].(map[string]any)
	if usage == nil {
		return CheckResult{Name: "bedrock_state", Pass: true, Expected: "无 bedrock_state 字段", Actual: "无 usage 对象", Detail: "no usage object"}
	}
	if _, ok := usage["bedrock_state"]; ok {
		return CheckResult{Name: "bedrock_state", Pass: false, Expected: "无 bedrock_state 字段", Actual: "存在 bedrock_state", Detail: "bedrock_state present in usage", Fix: "strip_bedrock"}
	}
	return CheckResult{Name: "bedrock_state", Pass: true, Expected: "无 bedrock_state 字段", Actual: "无 bedrock_state 字段", Detail: "no bedrock_state"}
}

// checkInferenceGeo checks if inference_geo has a valid value.
func checkInferenceGeo(body map[string]any, model string) CheckResult {
	usage, _ := body["usage"].(map[string]any)
	if usage == nil {
		return CheckResult{Name: "inference_geo", Pass: false, Expected: "global 或 not_available", Actual: "无 usage 对象", Detail: "no usage object", Fix: "force_geo"}
	}
	geo, _ := usage["inference_geo"].(string)
	if geo == "" {
		return CheckResult{Name: "inference_geo", Pass: false, Expected: "global 或 not_available", Actual: "字段缺失", Detail: "missing inference_geo", Fix: "force_geo"}
	}
	// Clean reference channels use both global (Console) and not_available
	// (Azure/managed). Treat both as valid; this check should catch missing or
	// obviously non-standard values, not distinguish clean providers.
	if geo == "global" || geo == "not_available" {
		return CheckResult{Name: "inference_geo", Pass: true, Expected: "global 或 not_available", Actual: geo, Detail: "inference_geo=" + geo}
	}
	return CheckResult{Name: "inference_geo", Pass: false, Expected: "global 或 not_available", Actual: geo,
		Detail: "inference_geo=" + geo + " expected global/not_available", Fix: "force_geo"}
}

// checkStopDetails checks if stop_details field exists in the response.
func checkStopDetails(body map[string]any) CheckResult {
	if _, ok := body["stop_details"]; ok {
		return CheckResult{Name: "stop_details", Pass: true, Expected: "存在 stop_details 字段", Actual: "存在 stop_details 字段", Detail: "stop_details present"}
	}
	return CheckResult{Name: "stop_details", Pass: false, Expected: "存在 stop_details 字段", Actual: "stop_details 缺失", Detail: "stop_details missing", Fix: "body_rewrite"}
}

// checkStopDetailsStructure verifies stop_details.type matches stop_reason.
// Official API: stop_details: {type: "end_turn"|"stop_sequence"|"max_tokens"}
func checkStopDetailsStructure(body map[string]any) CheckResult {
	sd, ok := body["stop_details"].(map[string]any)
	if !ok {
		// stop_details might be null — that's a separate check
		return CheckResult{Name: "stop_details_structure", Pass: true, Expected: "stop_details.type 与 stop_reason 一致", Actual: "stop_details 为 null（跳过）", Detail: "stop_details null (skip)"}
	}
	sdType, _ := sd["type"].(string)
	if sdType == "" {
		return CheckResult{Name: "stop_details_structure", Pass: false, Expected: "stop_details.type 非空", Actual: "type 字段为空",
			Detail: "stop_details has no type field", Fix: "body_rewrite"}
	}
	sr, _ := body["stop_reason"].(string)
	if sr != "" && sdType != sr {
		return CheckResult{Name: "stop_details_structure", Pass: false, Expected: "stop_details.type=" + sr, Actual: "stop_details.type=" + sdType,
			Detail: fmt.Sprintf("stop_details.type=%s != stop_reason=%s", sdType, sr), Fix: "body_rewrite"}
	}
	return CheckResult{Name: "stop_details_structure", Pass: true, Expected: "stop_details.type 与 stop_reason 一致", Actual: "stop_details.type=" + sdType,
		Detail: "stop_details.type=" + sdType + " matches stop_reason"}
}

// checkBackendType detects the backend type from the message ID prefix.
// msg_bdrk_ = Bedrock, gen- = OpenRouter, chatcmpl- = OneAPI/sub2api
func checkBackendType(body map[string]any) CheckResult {
	id, _ := body["id"].(string)
	if id == "" {
		return CheckResult{Name: "backend_type", Pass: false, Expected: "msg_01 或 msg_ 前缀", Actual: "id 字段为空", Detail: "no id field"}
	}
	switch {
	case strings.HasPrefix(id, "msg_bdrk_"):
		return CheckResult{Name: "backend_type", Pass: false, Expected: "msg_01 或 msg_ 前缀", Actual: "msg_bdrk_ 前缀（Bedrock）", Detail: "Bedrock backend (msg_bdrk_)", Fix: "id_rewrite"}
	case strings.HasPrefix(id, "gen-"):
		return CheckResult{Name: "backend_type", Pass: false, Expected: "msg_01 或 msg_ 前缀", Actual: "gen- 前缀（OpenRouter）", Detail: "OpenRouter backend (gen-)", Fix: "id_rewrite"}
	case strings.HasPrefix(id, "chatcmpl-"):
		return CheckResult{Name: "backend_type", Pass: false, Expected: "msg_01 或 msg_ 前缀", Actual: "chatcmpl- 前缀（OneAPI/sub2api）", Detail: "OneAPI/sub2api backend (chatcmpl-)", Fix: "id_rewrite"}
	case strings.HasPrefix(id, "msg_01"):
		return CheckResult{Name: "backend_type", Pass: true, Expected: "msg_01 或 msg_ 前缀", Actual: "msg_01 前缀", Detail: "official format (msg_01)"}
	case strings.HasPrefix(id, "msg_") && len(id) >= 26:
		return CheckResult{Name: "backend_type", Pass: true, Expected: "msg_01 或 msg_ 前缀", Actual: "msg_ 前缀", Detail: "Anthropic format (msg_)"}
	default:
		return CheckResult{Name: "backend_type", Pass: false, Expected: "msg_01 或 msg_ 前缀", Actual: truncate(id, 20), Detail: "unknown id prefix: " + truncate(id, 20), Fix: "id_rewrite"}
	}
}

var cfRayRe = regexp.MustCompile(`^[0-9a-f]{16}-[A-Z]{3}$`)

func checkCfRayFormat(headers http.Header) CheckResult {
	if hasXRatelimitHeaders(headers) {
		return CheckResult{Name: "cf_ray_format", Pass: true, Expected: "Cf-Ray 头（托管渠道可选）", Actual: "托管渠道，跳过检查", Detail: "managed-channel headers (Cf-Ray not expected)"}
	}
	ray := headers.Get("Cf-Ray")
	if ray == "" {
		return CheckResult{Name: "cf_ray_format", Pass: false, Expected: "hex{16}-IATA 格式 Cf-Ray", Actual: "Cf-Ray 头缺失", Detail: "no Cf-Ray header", Fix: "headers_fake"}
	}
	if cfRayRe.MatchString(ray) {
		return CheckResult{Name: "cf_ray_format", Pass: true, Expected: "hex{16}-IATA 格式", Actual: ray, Detail: "Cf-Ray format OK: " + ray}
	}
	return CheckResult{Name: "cf_ray_format", Pass: false, Expected: "hex{16}-IATA 格式", Actual: truncate(ray, 30), Detail: "Cf-Ray format invalid: " + truncate(ray, 30), Fix: "headers_fake"}
}

// checkHiddenPrompt detects injected system prompts by analyzing input_tokens.
// A bare "hi" with NO system prompt should use ~8-10 input_tokens.
// If input_tokens > 20, the upstream likely injects a hidden system prompt.
func checkHiddenPrompt(body map[string]any) CheckResult {
	usage, _ := body["usage"].(map[string]any)
	if usage == nil {
		return CheckResult{Name: "hidden_prompt", Pass: false, Expected: "input_tokens ≤ 20", Actual: "无 usage 对象", Detail: "no usage object"}
	}
	inputTok := fingerprint.IntVal(usage, "input_tokens")
	actualStr := fmt.Sprintf("input_tokens=%d", inputTok)
	// "hi" with no system prompt = ~8 tokens (message overhead + content)
	// Threshold 20 allows some variance but catches any injected system prompt
	if inputTok <= 20 {
		return CheckResult{Name: "hidden_prompt", Pass: true, Expected: "input_tokens ≤ 20", Actual: actualStr,
			Detail: fmt.Sprintf("input_tokens=%d (clean, no hidden prompt)", inputTok)}
	}
	return CheckResult{Name: "hidden_prompt", Pass: false, Expected: "input_tokens ≤ 20", Actual: actualStr,
		Detail: fmt.Sprintf("input_tokens=%d (expected ≤20 for bare 'hi'), likely hidden system prompt injected", inputTok)}
}

// checkTokenBudget detects hidden prompt injection in requests that include our
// known system prompt. mini_probe sends fullSystem() + "hi" + max_tokens=1.
// Official channels report input_tokens=36 (opus-4-6) or 58 (opus-4-7).
// If input_tokens greatly exceeds the expected budget, the upstream has injected
// additional hidden content on top of our system prompt.
func checkTokenBudget(body map[string]any, model string) CheckResult {
	usage, _ := body["usage"].(map[string]any)
	if usage == nil {
		return CheckResult{Name: "token_budget", Pass: false, Expected: "input_tokens ≤ 80", Actual: "无 usage 对象", Detail: "no usage object"}
	}
	inputTok := fingerprint.IntVal(usage, "input_tokens")
	actualStr := fmt.Sprintf("input_tokens=%d", inputTok)
	// Reference budgets from clean channels:
	//   opus-4-6 / sonnet-4-6 : 36
	//   opus-4-7              : 58
	// Allow generous margin (+20) to absorb minor tokenizer/version drift.
	maxBudget := 80
	expectedStr := fmt.Sprintf("input_tokens ≤ %d", maxBudget)
	if inputTok <= maxBudget {
		return CheckResult{Name: "token_budget", Pass: true, Expected: expectedStr, Actual: actualStr,
			Detail: fmt.Sprintf("input_tokens=%d within budget (≤%d)", inputTok, maxBudget)}
	}
	return CheckResult{Name: "token_budget", Pass: false, Expected: expectedStr, Actual: actualStr,
		Detail: fmt.Sprintf("input_tokens=%d exceeds budget %d — likely extra hidden prompt injected on top of system prompt", inputTok, maxBudget)}
}

// checkServiceTier checks if usage contains a service_tier field.
// Official API always includes service_tier; proxies often strip it.
func checkServiceTier(body map[string]any) CheckResult {
	usage, _ := body["usage"].(map[string]any)
	if usage == nil {
		return CheckResult{Name: "service_tier", Pass: false, Expected: "存在 service_tier 字段", Actual: "无 usage 对象", Detail: "no usage object"}
	}
	if _, ok := usage["service_tier"]; !ok {
		return CheckResult{Name: "service_tier", Pass: false, Expected: "存在 service_tier 字段", Actual: "service_tier 缺失", Detail: "service_tier missing from usage"}
	}
	return CheckResult{Name: "service_tier", Pass: true, Expected: "存在 service_tier 字段", Actual: "存在 service_tier 字段", Detail: "service_tier present"}
}

// checkServerHeader verifies the Server response header value.
// Console: "cloudflare"; Azure/managed: absent (acceptable); Proxy: "nginx" etc.
func checkServerHeader(headers http.Header) CheckResult {
	if hasXRatelimitHeaders(headers) {
		// Azure/managed channels often have no Server header — that's fine
		return CheckResult{Name: "server_header", Pass: true, Expected: "cloudflare（托管渠道可选）", Actual: "托管渠道，跳过检查", Detail: "managed-channel (Server optional)"}
	}
	server := headers.Get("Server")
	if server == "" {
		return CheckResult{Name: "server_header", Pass: false, Expected: "Server: cloudflare", Actual: "Server 头缺失", Detail: "no Server header", Fix: "headers_fake"}
	}
	if strings.Contains(strings.ToLower(server), "cloudflare") {
		return CheckResult{Name: "server_header", Pass: true, Expected: "cloudflare", Actual: "cloudflare", Detail: "Server=cloudflare OK"}
	}
	return CheckResult{Name: "server_header", Pass: false, Expected: "cloudflare", Actual: truncate(server, 30),
		Detail: "Server=" + truncate(server, 30) + " (expected cloudflare)", Fix: "headers_fake"}
}

// checkSignatureTypeLeak detects the non-standard "signature_type" field in thinking blocks.
// Official API never includes this field; some proxies add it.
func checkSignatureTypeLeak(body map[string]any) CheckResult {
	content, _ := body["content"].([]any)
	for _, cb := range content {
		m, ok := cb.(map[string]any)
		if !ok {
			continue
		}
		t, _ := m["type"].(string)
		if t == "thinking" {
			if _, has := m["signature_type"]; has {
				return CheckResult{Name: "signature_type_leak", Pass: false, Expected: "无 signature_type 字段", Actual: "存在 signature_type 字段",
					Detail: "thinking block has non-standard signature_type field"}
			}
		}
	}
	return CheckResult{Name: "signature_type_leak", Pass: true, Expected: "无 signature_type 字段", Actual: "无 signature_type 字段", Detail: "no signature_type leak"}
}

// checkCookieDomain validates Set-Cookie contains correct domain for Anthropic API.
func checkCookieDomain(headers http.Header) CheckResult {
	if hasXRatelimitHeaders(headers) {
		return CheckResult{Name: "cookie_domain", Pass: true, Expected: "anthropic.com 域（托管渠道可选）", Actual: "托管渠道，跳过检查", Detail: "managed-channel headers (Set-Cookie not expected)"}
	}
	cookie := headers.Get("Set-Cookie")
	if cookie == "" {
		return CheckResult{Name: "cookie_domain", Pass: false, Expected: "Set-Cookie 含 anthropic.com", Actual: "Set-Cookie 头缺失", Detail: "no Set-Cookie header", Fix: "headers_fake"}
	}
	if strings.Contains(cookie, "anthropic.com") {
		return CheckResult{Name: "cookie_domain", Pass: true, Expected: "anthropic.com 域", Actual: "anthropic.com 域", Detail: "cookie domain OK"}
	}
	return CheckResult{Name: "cookie_domain", Pass: false, Expected: "anthropic.com 域", Actual: "非 anthropic.com 域", Detail: "cookie missing anthropic.com domain", Fix: "headers_fake"}
}
