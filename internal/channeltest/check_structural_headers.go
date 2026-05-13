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

func checkHeaders(headers http.Header) CheckResult {
	if hasXRatelimitHeaders(headers) {
		return CheckResult{Name: "headers", Pass: true,
			Expected: "速率限制头存在", Actual: "X-Ratelimit 头齐全",
			Detail: "managed-channel X-Ratelimit headers present"}
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
		return CheckResult{Name: "headers", Pass: true,
			Expected: fmt.Sprintf("13/13 速率限制头"), Actual: "13/13 齐全",
			Detail: "all Anthropic ratelimit headers present"}
	}
	return CheckResult{Name: "headers", Pass: false,
		Expected: fmt.Sprintf("13/13 速率限制头"), Actual: fmt.Sprintf("缺少 %d/%d", missing, len(required)),
		Detail: fmt.Sprintf("missing %d/%d ratelimit headers", missing, len(required)), Fix: "headers_fake"}
}

func checkCfHeaders(headers http.Header) CheckResult {
	if hasXRatelimitHeaders(headers) {
		return CheckResult{Name: "cf_headers", Pass: true,
			Expected: "Cloudflare 头或托管头", Actual: "托管渠道 (无需 Cloudflare)",
			Detail: "managed-channel headers (no Cloudflare expected)"}
	}

	var missing []string
	if headers.Get("Cf-Ray") == "" {
		missing = append(missing, "Cf-Ray")
	}
	server := headers.Get("Server")
	if server == "" {
		missing = append(missing, "Server")
	}
	cookie := headers.Get("Set-Cookie")
	if cookie == "" || !strings.Contains(cookie, "_cfuvid") {
		missing = append(missing, "Set-Cookie(_cfuvid)")
	}
	if len(missing) == 0 {
		return CheckResult{Name: "cf_headers", Pass: true,
			Expected: "Cf-Ray + Server + _cfuvid", Actual: "均存在",
			Detail: "Cloudflare-style headers present"}
	}
	return CheckResult{Name: "cf_headers", Pass: false,
		Expected: "Cf-Ray + Server + _cfuvid", Actual: "缺少: " + strings.Join(missing, ", "),
		Detail: "missing: " + strings.Join(missing, ", "), Fix: "headers_fake"}
}

func checkServerTiming(headers http.Header) CheckResult {
	if hasXRatelimitHeaders(headers) {
		return CheckResult{Name: "server_timing", Pass: true,
			Expected: "Server-Timing 或托管头", Actual: "托管渠道 (可选)",
			Detail: "managed-channel headers (Server-Timing optional)"}
	}
	if envoy := headers.Get("X-Envoy-Upstream-Service-Time"); envoy != "" {
		return CheckResult{Name: "server_timing", Pass: true,
			Expected: "Server-Timing / X-Envoy 头", Actual: "X-Envoy=" + truncate(envoy, 20),
			Detail: "X-Envoy-Upstream-Service-Time present: " + truncate(envoy, 40)}
	}
	st := headers.Get("Server-Timing")
	if st == "" {
		return CheckResult{Name: "server_timing", Pass: false,
			Expected: "Server-Timing / X-Envoy 头", Actual: "均不存在",
			Detail: "no Server-Timing/X-Envoy header", Fix: "headers_fake"}
	}
	if strings.Contains(st, "x-originResponse") {
		return CheckResult{Name: "server_timing", Pass: true,
			Expected: "Server-Timing 含 x-originResponse", Actual: truncate(st, 40),
			Detail: "Server-Timing OK: " + truncate(st, 40)}
	}
	return CheckResult{Name: "server_timing", Pass: false,
		Expected: "Server-Timing 含 x-originResponse", Actual: truncate(st, 40),
		Detail: "Server-Timing format unexpected: " + truncate(st, 40), Fix: "headers_fake"}
}
