package channeltest

import (
	"errors"
	"fmt"
	"strings"
)

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

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// apiErrorChecks generates check results when a probe encounters an API error.
// It produces one api_error check and marks all expected checks as skipped.
func apiErrorChecks(probe *Probe, apiErr *APIError) []CheckResult {
	actual := fmt.Sprintf("HTTP %d", apiErr.StatusCode)
	if apiErr.ErrorType != "" {
		actual += ": " + apiErr.ErrorType
	}
	detail := apiErr.Message
	if detail == "" {
		detail = apiErr.Error()
	}

	checks := []CheckResult{{
		Name:     "api_error",
		Pass:     false,
		Expected: "HTTP 200",
		Actual:   actual,
		Detail:   detail,
	}}
	for _, name := range probe.Checks {
		checks = append(checks, CheckResult{
			Name:     name,
			Pass:     false,
			Expected: "正常响应",
			Actual:   fmt.Sprintf("跳过 (HTTP %d)", apiErr.StatusCode),
			Detail:   fmt.Sprintf("skipped: %s", apiErr.ErrorType),
		})
	}
	return checks
}

// asAPIError extracts *APIError from an error chain, returns nil if not an APIError.
func asAPIError(err error) *APIError {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr
	}
	return nil
}
