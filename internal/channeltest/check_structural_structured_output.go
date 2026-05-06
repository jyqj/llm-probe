package channeltest

import (
	"fmt"
	"strings"

	"encoding/json"
)

// checkStructuredJSONValid verifies the response text is valid JSON when output_config is used.
func checkStructuredJSONValid(body map[string]any) CheckResult {
	text := collectResponseText(body)
	if text == "" {
		return CheckResult{Name: "structured_json_valid", Pass: false, Detail: "no text content", Fix: "body_rewrite"}
	}
	var parsed any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return CheckResult{Name: "structured_json_valid", Pass: false,
			Detail: "response is not valid JSON: " + truncate(err.Error(), 60), Fix: "body_rewrite"}
	}
	return CheckResult{Name: "structured_json_valid", Pass: true, Detail: "valid JSON response"}
}

// checkStructuredSchemaMatch verifies the JSON response matches the expected schema {title: string, desc: string}.
// checkStructuredSchemaMatch verifies the JSON response matches the expected schema {title: string, desc: string}.
func checkStructuredSchemaMatch(body map[string]any) CheckResult {
	text := collectResponseText(body)
	if text == "" {
		return CheckResult{Name: "structured_schema_match", Pass: false, Detail: "no text content", Fix: "body_rewrite"}
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(text), &obj); err != nil {
		return CheckResult{Name: "structured_schema_match", Pass: false,
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
			Detail: "missing required fields: " + strings.Join(missing, ", "), Fix: "body_rewrite"}
	}
	if title == "" || desc == "" {
		return CheckResult{Name: "structured_schema_match", Pass: false,
			Detail: "title or desc is empty", Fix: "body_rewrite"}
	}
	// Check no extra fields (additionalProperties: false)
	extra := 0
	for k := range obj {
		if k != "title" && k != "desc" {
			extra++
		}
	}
	if extra > 0 {
		return CheckResult{Name: "structured_schema_match", Pass: false,
			Detail: fmt.Sprintf("schema has %d extra fields (additionalProperties should be false)", extra), Fix: "body_rewrite"}
	}
	return CheckResult{Name: "structured_schema_match", Pass: true,
		Detail: "schema match: {title, desc} OK"}
}

// checkStructuredStopReason verifies stop_reason is "end_turn" when structured output is used.
// With output_config json_schema, stop_reason should be "end_turn" (not "max_tokens").
// "refusal" is also a valid structured output stop_reason (safety refusal).
// checkStructuredStopReason verifies stop_reason is "end_turn" when structured output is used.
// With output_config json_schema, stop_reason should be "end_turn" (not "max_tokens").
// "refusal" is also a valid structured output stop_reason (safety refusal).
func checkStructuredStopReason(body map[string]any) CheckResult {
	sr, _ := body["stop_reason"].(string)
	if sr == "" {
		return CheckResult{Name: "structured_stop_reason", Pass: false,
			Detail: "stop_reason is empty/null", Fix: "body_rewrite"}
	}
	if sr == "end_turn" {
		return CheckResult{Name: "structured_stop_reason", Pass: true, Detail: "stop_reason=end_turn OK"}
	}
	if sr == "refusal" {
		return CheckResult{Name: "structured_stop_reason", Pass: true, Detail: "stop_reason=refusal (safety)"}
	}
	return CheckResult{Name: "structured_stop_reason", Pass: false,
		Detail: "stop_reason=" + sr + " expected end_turn for structured output", Fix: "body_rewrite"}
}

// checkMessageStartUsage verifies the message_start event contains input-side usage fields.
// Official streaming: message_start.usage has input_tokens, cache_creation_input_tokens, cache_read_input_tokens.
