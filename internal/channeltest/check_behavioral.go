package channeltest

import (
	"encoding/json"
	"strings"
)

// checkTagReplay verifies the response contains the echoed tag from the request.
// cctest sends a random tag and expects the LLM to echo it back.
func checkTagReplay(body map[string]any, tag string) CheckResult {
	if tag == "" {
		return CheckResult{Name: "tag_replay", Pass: true, Expected: "tag echoed back", Actual: "no tag sent (skip)", Detail: "no tag to check (skip)"}
	}
	content, _ := body["content"].([]any)
	for _, cb := range content {
		m, ok := cb.(map[string]any)
		if !ok {
			continue
		}
		t, _ := m["type"].(string)
		if t == "text" {
			text, _ := m["text"].(string)
			if strings.Contains(text, tag) {
				return CheckResult{Name: "tag_replay", Pass: true, Expected: "response contains tag: " + tag, Actual: "tag found in response", Detail: "tag found in response: " + tag}
			}
		}
	}
	return CheckResult{Name: "tag_replay", Pass: false, Expected: "response contains tag: " + tag, Actual: "tag not found", Detail: "tag NOT found in response: " + tag}
}

// checkIdentityResponse verifies the response content mentions Claude or Anthropic.
// cctest sends identity probe questions to check if the LLM knows what it is.
// Also handles structured JSON output (output_config json_schema) where the text
// content is a JSON string containing title/desc fields.
func checkIdentityResponse(body map[string]any) CheckResult {
	text := collectResponseText(body)
	lower := strings.ToLower(text)
	if strings.Contains(lower, "claude") || strings.Contains(lower, "anthropic") {
		return CheckResult{Name: "identity_response", Pass: true, Expected: "mentions Claude or Anthropic", Actual: "found Claude/Anthropic in response", Detail: "response mentions Claude/Anthropic"}
	}
	// Try parsing as JSON (structured output from output_config json_schema)
	if text != "" {
		var structured map[string]any
		if json.Unmarshal([]byte(text), &structured) == nil {
			combined := ""
			for _, v := range structured {
				if s, ok := v.(string); ok {
					combined += " " + s
				}
			}
			combinedLower := strings.ToLower(combined)
			if strings.Contains(combinedLower, "claude") || strings.Contains(combinedLower, "anthropic") {
				return CheckResult{Name: "identity_response", Pass: true, Expected: "mentions Claude or Anthropic", Actual: "found Claude/Anthropic in structured output", Detail: "structured output mentions Claude/Anthropic"}
			}
		}
	}
	if text == "" {
		return CheckResult{Name: "identity_response", Pass: false, Expected: "mentions Claude or Anthropic", Actual: "empty response (no text)", Detail: "no text content in response"}
	}
	return CheckResult{Name: "identity_response", Pass: false, Expected: "mentions Claude or Anthropic", Actual: "neither Claude nor Anthropic found", Detail: "response does not mention Claude or Anthropic"}
}

// checkPoisonAnswer verifies the answer to the classic poison bottle problem.
// 1000 bottles, 1 poisoned, 24h, answer = 10 mice (binary encoding).
func checkPoisonAnswer(body map[string]any) CheckResult {
	text := collectResponseText(body)
	if text == "" {
		return CheckResult{Name: "poison_answer", Pass: false, Expected: "answer contains standalone 10", Actual: "empty response (no text)", Detail: "no text content"}
	}
	// Match standalone "10" — avoid false positives from "100", "1000", etc.
	// Use word-boundary-like check: "10" must not be adjacent to other digits.
	for i := 0; i < len(text); i++ {
		if text[i] != '1' {
			continue
		}
		if i+1 >= len(text) || text[i+1] != '0' {
			continue
		}
		// Check char after "10" is not a digit
		if i+2 < len(text) && text[i+2] >= '0' && text[i+2] <= '9' {
			continue
		}
		// Check char before "1" is not a digit
		if i > 0 && text[i-1] >= '0' && text[i-1] <= '9' {
			continue
		}
		return CheckResult{Name: "poison_answer", Pass: true, Expected: "answer contains standalone 10", Actual: "found standalone 10 in response", Detail: "contains correct answer (10 mice)"}
	}
	return CheckResult{Name: "poison_answer", Pass: false, Expected: "answer contains standalone 10", Actual: "standalone 10 not found", Detail: "standalone 10 not found in response"}
}

// checkLogicAnswer verifies the answer to the 3-switch puzzle.
// The correct method involves turning switches on, waiting, then checking heat.
func checkLogicAnswer(body map[string]any) CheckResult {
	text := collectResponseText(body)
	lower := strings.ToLower(text)
	if text == "" {
		return CheckResult{Name: "logic_answer", Pass: false, Expected: "mentions heat/warm method", Actual: "empty response (no text)", Detail: "no text content"}
	}
	// The answer involves heat/temperature/warm — checking if bulb is warm
	heatKeywords := []string{"热", "温度", "摸", "warm", "heat", "hot", "touch"}
	for _, kw := range heatKeywords {
		if strings.Contains(lower, kw) {
			return CheckResult{Name: "logic_answer", Pass: true, Expected: "mentions heat/warm method", Actual: "found keyword: " + kw, Detail: "contains heat/warm method"}
		}
	}
	return CheckResult{Name: "logic_answer", Pass: false, Expected: "mentions heat/warm method", Actual: "no heat/warm keywords found", Detail: "heat method not found in response"}
}

// checkIdentityNoLeak verifies the response does not leak internal codenames.
// Known internal identifiers: kiro, warp, 0z, sn, antigravity.
func checkIdentityNoLeak(body map[string]any) CheckResult {
	text := strings.ToLower(collectResponseText(body))
	if text == "" {
		return CheckResult{Name: "identity_no_leak", Pass: true, Expected: "no internal codename claims", Actual: "no text content (skip)", Detail: "no text content (skip)"}
	}
	leaks := []string{"kiro", "warp", "0z", "antigravity"}
	negations := []string{"不是", "并非", "没有", "不存在", "我不是", "not ", "not a", "no ", "isn't", "not running"}
	claimPhrases := []string{"我是", "i am", "i'm", "as ", "running on", "运行在", "运行于", "powered by"}
	for _, kw := range leaks {
		idx := strings.Index(text, kw)
		for idx >= 0 {
			start := idx - 40
			if start < 0 {
				start = 0
			}
			end := idx + len(kw) + 40
			if end > len(text) {
				end = len(text)
			}
			window := text[start:end]
			negated := false
			for _, n := range negations {
				if strings.Contains(window, n) {
					negated = true
					break
				}
			}
			if !negated {
				for _, phrase := range claimPhrases {
					if strings.Contains(window, phrase) {
						return CheckResult{Name: "identity_no_leak", Pass: false,
							Expected: "no internal codename claims", Actual: "claimed codename: " + kw,
							Detail: "claimed/leaked internal codename: " + kw}
					}
				}
			}
			next := strings.Index(text[idx+len(kw):], kw)
			if next < 0 {
				break
			}
			idx = idx + len(kw) + next
		}
	}
	return CheckResult{Name: "identity_no_leak", Pass: true, Expected: "no internal codename claims", Actual: "no codename claim detected", Detail: "no internal codename claim"}
}

// checkIdentityPlatform verifies the response does not claim to run on non-Anthropic platforms.
func checkIdentityPlatform(body map[string]any) CheckResult {
	text := strings.ToLower(collectResponseText(body))
	if text == "" {
		return CheckResult{Name: "identity_platform", Pass: true, Expected: "no non-Anthropic platform claims", Actual: "no text content (skip)", Detail: "no text content (skip)"}
	}
	platforms := []struct{ kw, label string }{
		{"bedrock", "AWS Bedrock"},
		{"vertex", "Google Vertex"},
		{"azure", "Azure"},
		{"openrouter", "OpenRouter"},
	}
	// Only flag if the model CLAIMS to be running on these platforms
	// (not just mentioning them in general discussion)
	claimPhrases := []string{
		"运行在", "运行于", "部署在", "hosted on", "running on", "deployed on",
		"i am running", "i run on", "i'm on", "powered by",
	}
	for _, p := range platforms {
		if !strings.Contains(text, p.kw) {
			continue
		}
		for _, phrase := range claimPhrases {
			// Check if platform keyword is near a claim phrase
			pidx := strings.Index(text, p.kw)
			for _, cp := range []int{pidx - 30, pidx + len(p.kw)} {
				if cp < 0 {
					cp = 0
				}
				end := cp + 40
				if end > len(text) {
					end = len(text)
				}
				window := text[cp:end]
				if strings.Contains(window, phrase) {
					return CheckResult{Name: "identity_platform", Pass: false,
						Expected: "no non-Anthropic platform claims", Actual: "claims to run on " + p.label,
						Detail: "claims to run on " + p.label}
				}
			}
		}
	}
	return CheckResult{Name: "identity_platform", Pass: true, Expected: "no non-Anthropic platform claims", Actual: "no platform claim detected", Detail: "no non-Anthropic platform claims"}
}

// checkToolForcedCompliance verifies a tool_use content block exists when tool_choice was forced.
func checkToolForcedCompliance(body map[string]any) CheckResult {
	content, _ := body["content"].([]any)
	for _, cb := range content {
		m, ok := cb.(map[string]any)
		if !ok {
			continue
		}
		t, _ := m["type"].(string)
		if t == "tool_use" || t == "server_tool_use" {
			return CheckResult{Name: "tool_forced_compliance", Pass: true, Expected: "tool_use block in response", Actual: "tool_use block present", Detail: "tool_use block present"}
		}
	}
	return CheckResult{Name: "tool_forced_compliance", Pass: false,
		Expected: "tool_use block in response", Actual: "no tool_use block found",
		Detail: "no tool_use block found (tool_choice was forced)"}
}
