package channeltest

import (
	"fmt"

	"encoding/base64"
)

func checkWebSearchResult(body map[string]any) CheckResult {
	content, _ := body["content"].([]any)
	for _, cb := range content {
		m, ok := cb.(map[string]any)
		if !ok {
			continue
		}
		t, _ := m["type"].(string)
		if t == "web_search_tool_result" {
			tid, _ := m["tool_use_id"].(string)
			if tid != "" && !srvToolIDRe.MatchString(tid) {
				return CheckResult{Name: "web_search_result", Pass: false,
					Expected: "srvtoolu_01 格式", Actual: truncate(tid, 20),
					Detail: "web_search_tool_result tool_use_id not srvtoolu_01 format: " + truncate(tid, 20), Fix: "id_rewrite"}
			}
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
							Expected: "encrypted_content 非空", Actual: "空",
							Detail: "web_search_result missing encrypted_content", Fix: "signature_rewrite"}
					}
					_, err := base64.StdEncoding.DecodeString(ec)
					if err != nil {
						return CheckResult{Name: "web_search_result", Pass: false,
							Expected: "合法 base64 encrypted_content", Actual: "base64 解码失败",
							Detail: "encrypted_content not valid base64", Fix: "signature_rewrite"}
					}
				}
			}
			return CheckResult{Name: "web_search_result", Pass: true,
				Expected: "web_search_tool_result 结构完整", Actual: "结构完整",
				Detail: "web_search_tool_result structure OK"}
		}
	}
	return CheckResult{Name: "web_search_result", Pass: true,
		Expected: "web_search_tool_result", Actual: "无 (skip)",
		Detail: "no web_search_tool_result (skip)"}
}

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
		return CheckResult{Name: "tool_stop_reason", Pass: true,
			Expected: "tool_use / end_turn (含工具块)", Actual: sr,
			Detail: "tool content present with stop_reason=" + sr}
	}
	if sr == "" {
		return CheckResult{Name: "tool_stop_reason", Pass: false,
			Expected: "tool_use / end_turn", Actual: "空",
			Detail: "stop_reason empty for forced tool request", Fix: "body_rewrite"}
	}
	return CheckResult{Name: "tool_stop_reason", Pass: false,
		Expected: "tool_use (含工具块)", Actual: fmt.Sprintf("%s (tool=%v)", sr, hasTool),
		Detail: "stop_reason=" + sr + " without expected tool content", Fix: "body_rewrite"}
}

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
		return CheckResult{Name: "server_tool_type", Pass: true,
			Expected: "server_tool_use", Actual: "server_tool_use",
			Detail: "web_search uses server_tool_use"}
	}
	if hasPlainToolUse {
		return CheckResult{Name: "server_tool_type", Pass: false,
			Expected: "server_tool_use", Actual: "tool_use (降级)",
			Detail: "web_search uses plain tool_use instead of server_tool_use", Fix: "body_rewrite"}
	}
	return CheckResult{Name: "server_tool_type", Pass: true,
		Expected: "server_tool_use", Actual: "无 web_search (skip)",
		Detail: "no web_search tool (skip)"}
}

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
		return CheckResult{Name: "citations_present", Pass: true,
			Expected: "citations 数组", Actual: "无 web_search (skip)",
			Detail: "no web_search result (skip)"}
	}
	if hasCitations {
		return CheckResult{Name: "citations_present", Pass: true,
			Expected: "citations 数组", Actual: "citations 存在",
			Detail: "citations present in text blocks"}
	}
	return CheckResult{Name: "citations_present", Pass: false,
		Expected: "text 块含 citations", Actual: "无 citations",
		Detail: "web_search result present but no text blocks with citations", Fix: "body_rewrite"}
}

func checkServerToolUsage(body map[string]any) CheckResult {
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
		return CheckResult{Name: "server_tool_usage", Pass: true,
			Expected: "server_tool_use 统计", Actual: "无 server tool (skip)",
			Detail: "no server tool in response (skip)"}
	}
	usage, _ := body["usage"].(map[string]any)
	if usage == nil {
		return CheckResult{Name: "server_tool_usage", Pass: false,
			Expected: "usage.server_tool_use 对象", Actual: "无 usage",
			Detail: "no usage object", Fix: "body_rewrite"}
	}
	stu, ok := usage["server_tool_use"].(map[string]any)
	if !ok {
		return CheckResult{Name: "server_tool_usage", Pass: false,
			Expected: "usage.server_tool_use 对象", Actual: "字段缺失",
			Detail: "server tool used but usage.server_tool_use missing", Fix: "body_rewrite"}
	}
	if _, ok := stu["web_search_requests"]; !ok {
		return CheckResult{Name: "server_tool_usage", Pass: false,
			Expected: "web_search_requests 字段", Actual: "字段缺失",
			Detail: "server_tool_use missing web_search_requests", Fix: "body_rewrite"}
	}
	return CheckResult{Name: "server_tool_usage", Pass: true,
		Expected: "server_tool_use 统计完整", Actual: fmt.Sprintf("%v", stu),
		Detail: fmt.Sprintf("server_tool_use present: %v", stu)}
}

// checkStopSequenceNull is defined in check_structural_body.go
