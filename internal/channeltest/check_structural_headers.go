package channeltest

import (
	"fmt"
	"strings"

	"net/http"
)

func hasXRatelimitHeaders(headers http.Header) bool {
	return headers.Get("X-Ratelimit-Limit-Requests") != "" &&
		headers.Get("X-Ratelimit-Limit-Tokens") != "" &&
		headers.Get("X-Ratelimit-Remaining-Requests") != "" &&
		headers.Get("X-Ratelimit-Remaining-Tokens") != ""
}

// checkUsageStructure verifies usage has cache_creation nested object and proper fields.
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
