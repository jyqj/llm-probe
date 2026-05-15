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
			Expected: "{name, title, desc} 对象", Actual: "无文本内容",
			Detail: "no text content", Fix: "body_rewrite"}
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(text), &obj); err != nil {
		return CheckResult{Name: "structured_schema_match", Pass: false,
			Expected: "{name, title, desc} 对象", Actual: "非 JSON 对象",
			Detail: "not a JSON object", Fix: "body_rewrite"}
	}
	required := []string{"name", "title", "desc"}
	var missing []string
	for _, k := range required {
		v, ok := obj[k].(string)
		if !ok || v == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return CheckResult{Name: "structured_schema_match", Pass: false,
			Expected: "{name, title, desc} 三个字段", Actual: "缺少 " + strings.Join(missing, ", "),
			Detail: "missing required fields: " + strings.Join(missing, ", "), Fix: "body_rewrite"}
	}
	extra := 0
	for k := range obj {
		if k != "name" && k != "title" && k != "desc" {
			extra++
		}
	}
	if extra > 0 {
		return CheckResult{Name: "structured_schema_match", Pass: false,
			Expected: "仅 {name, title, desc}", Actual: fmt.Sprintf("%d 个多余字段", extra),
			Detail: fmt.Sprintf("schema has %d extra fields (additionalProperties should be false)", extra), Fix: "body_rewrite"}
	}
	return CheckResult{Name: "structured_schema_match", Pass: true,
		Expected: "JSON 字段与 schema 匹配", Actual: "JSON 字段与 schema 匹配",
		Detail: "schema match: {name, title, desc} OK"}
}

func checkStructuredNameCorrect(body map[string]any, expectedName string) CheckResult {
	text := collectResponseText(body)
	if text == "" {
		return CheckResult{Name: "structured_name_correct", Pass: false,
			Expected: "包含 \"" + expectedName + "\"", Actual: "无文本内容",
			Detail: "no text content", Fix: "body_rewrite"}
	}
	if strings.Contains(text, expectedName) {
		return CheckResult{Name: "structured_name_correct", Pass: true,
			Expected: "包含 \"" + expectedName + "\"", Actual: "包含 \"" + expectedName + "\"",
			Detail: "name correct: " + expectedName}
	}
	return CheckResult{Name: "structured_name_correct", Pass: false,
		Expected: "包含 \"" + expectedName + "\"", Actual: truncate(text, 60),
		Detail: "expected name " + expectedName + " not found in output", Fix: "body_rewrite"}
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
