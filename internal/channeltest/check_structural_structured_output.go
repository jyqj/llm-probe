package channeltest

import (
	"fmt"
	"strings"

	"encoding/json"
)

func checkStructuredJSONValid(body map[string]any) CheckResult {
	text := collectResponseText(body)
	if text == "" {
		return CheckResult{Name: "structured_json_valid", Pass: false,
			Expected: "合法 JSON 输出", Actual: "无文本内容",
			Detail: "no text content", Fix: "body_rewrite"}
	}
	var parsed any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return CheckResult{Name: "structured_json_valid", Pass: false,
			Expected: "合法 JSON 输出", Actual: "JSON 解析失败: " + truncate(err.Error(), 40),
			Detail: "response is not valid JSON: " + truncate(err.Error(), 60), Fix: "body_rewrite"}
	}
	return CheckResult{Name: "structured_json_valid", Pass: true,
		Expected: "合法 JSON 输出", Actual: "JSON 合法",
		Detail: "valid JSON response"}
}

func checkStructuredSchemaMatch(body map[string]any) CheckResult {
	text := collectResponseText(body)
	if text == "" {
		return CheckResult{Name: "structured_schema_match", Pass: false,
			Expected: "{title, desc} 对象", Actual: "无文本内容",
			Detail: "no text content", Fix: "body_rewrite"}
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(text), &obj); err != nil {
		return CheckResult{Name: "structured_schema_match", Pass: false,
			Expected: "{title, desc} 对象", Actual: "非 JSON 对象",
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
			Expected: "{title, desc} 两个字段", Actual: "缺少 " + strings.Join(missing, ", "),
			Detail: "missing required fields: " + strings.Join(missing, ", "), Fix: "body_rewrite"}
	}
	if title == "" || desc == "" {
		return CheckResult{Name: "structured_schema_match", Pass: false,
			Expected: "title/desc 非空", Actual: "title 或 desc 为空",
			Detail: "title or desc is empty", Fix: "body_rewrite"}
	}
	extra := 0
	for k := range obj {
		if k != "title" && k != "desc" {
			extra++
		}
	}
	if extra > 0 {
		return CheckResult{Name: "structured_schema_match", Pass: false,
			Expected: "仅 {title, desc}", Actual: fmt.Sprintf("%d 个多余字段", extra),
			Detail: fmt.Sprintf("schema has %d extra fields (additionalProperties should be false)", extra), Fix: "body_rewrite"}
	}
	return CheckResult{Name: "structured_schema_match", Pass: true,
		Expected: "{title, desc}", Actual: "{title, desc}",
		Detail: "schema match: {title, desc} OK"}
}

func checkStructuredStopReason(body map[string]any) CheckResult {
	sr, _ := body["stop_reason"].(string)
	if sr == "" {
		return CheckResult{Name: "structured_stop_reason", Pass: false,
			Expected: "end_turn / refusal", Actual: "空/null",
			Detail: "stop_reason is empty/null", Fix: "body_rewrite"}
	}
	if sr == "end_turn" {
		return CheckResult{Name: "structured_stop_reason", Pass: true,
			Expected: "end_turn", Actual: sr, Detail: "stop_reason=end_turn OK"}
	}
	if sr == "refusal" {
		return CheckResult{Name: "structured_stop_reason", Pass: true,
			Expected: "end_turn / refusal", Actual: sr, Detail: "stop_reason=refusal (safety)"}
	}
	return CheckResult{Name: "structured_stop_reason", Pass: false,
		Expected: "end_turn", Actual: sr,
		Detail: "stop_reason=" + sr + " expected end_turn for structured output", Fix: "body_rewrite"}
}
