package channeltest

import (
	"fmt"

	"encoding/base64"
)

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
