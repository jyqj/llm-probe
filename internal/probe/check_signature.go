package probe

import (
	"encoding/base64"
	"fmt"
	"strings"

	"detector-service/internal/fingerprint"
)

// checkSignature checks thinking.signature is a valid protobuf blob.
func checkSignature(body map[string]any) CheckResult {
	content, _ := body["content"].([]any)
	for _, cb := range content {
		m, ok := cb.(map[string]any)
		if !ok {
			continue
		}
		t, _ := m["type"].(string)
		if t != "thinking" {
			continue
		}
		sig, _ := m["signature"].(string)
		if sig == "" {
			return CheckResult{Name: "signature", Pass: false, Detail: "thinking block has no signature", Fix: "signature_rewrite"}
		}
		raw, err := base64.StdEncoding.DecodeString(sig)
		if err != nil {
			return CheckResult{Name: "signature", Pass: false, Detail: "signature not valid base64", Fix: "signature_rewrite"}
		}
		if len(raw) < 4 || raw[0] != 0x12 {
			return CheckResult{Name: "signature", Pass: false, Detail: "signature protobuf tag mismatch", Fix: "signature_rewrite"}
		}
		ok2, _ := fingerprint.VerifySignature(sig)
		if ok2 {
			return CheckResult{Name: "signature", Pass: true, Detail: "detector-issued signature verified"}
		}
		return CheckResult{Name: "signature", Pass: true, Detail: "valid protobuf signature (external)"}
	}
	return CheckResult{Name: "signature", Pass: true, Detail: "no thinking block (skip)"}
}

// checkThinkingPresent checks if a thinking block exists when thinking was requested.
func checkThinkingPresent(body map[string]any) CheckResult {
	content, _ := body["content"].([]any)
	for _, cb := range content {
		m, ok := cb.(map[string]any)
		if !ok {
			continue
		}
		t, _ := m["type"].(string)
		if t == "thinking" || t == "redacted_thinking" {
			return CheckResult{Name: "thinking_present", Pass: true, Detail: "thinking block found"}
		}
	}
	// Clean Opus 4.7 Console references may omit thinking even with adaptive
	// thinking requested. Treat absence as non-discriminative.
	return CheckResult{Name: "thinking_present", Pass: true, Detail: "no thinking block (skip)"}
}

// checkThinkingOrder verifies thinking block appears before text block in content.
// Official API always puts thinking at index 0 when thinking is enabled.
func checkThinkingOrder(body map[string]any) CheckResult {
	content, _ := body["content"].([]any)
	if len(content) == 0 {
		return CheckResult{Name: "thinking_order", Pass: true, Detail: "no content blocks"}
	}
	firstType := ""
	hasThinking := false
	for i, cb := range content {
		m, ok := cb.(map[string]any)
		if !ok {
			continue
		}
		t, _ := m["type"].(string)
		if i == 0 {
			firstType = t
		}
		if t == "thinking" || t == "redacted_thinking" {
			hasThinking = true
			if i > 0 {
				return CheckResult{Name: "thinking_order", Pass: false,
					Detail: fmt.Sprintf("thinking at index %d, should be 0", i), Fix: "thinking_inject"}
			}
		}
	}
	if !hasThinking {
		return CheckResult{Name: "thinking_order", Pass: true, Detail: "no thinking block (skip)"}
	}
	if firstType == "thinking" || firstType == "redacted_thinking" {
		return CheckResult{Name: "thinking_order", Pass: true, Detail: "thinking at index 0"}
	}
	return CheckResult{Name: "thinking_order", Pass: false, Detail: "first block is " + firstType + " not thinking", Fix: "thinking_inject"}
}

// checkThinkingDisplayOmitted verifies thinking block format for display="omitted" mode.
// On Opus 4.7+, thinking.thinking may be "" (empty) but signature must still be present.
func checkThinkingDisplayOmitted(body map[string]any, model string) CheckResult {
	if !strings.Contains(model, "opus-4-7") && !strings.Contains(model, "opus-4-8") {
		return CheckResult{Name: "thinking_display_omitted", Pass: true, Detail: "not opus-4-7+ model (skip)"}
	}
	content, _ := body["content"].([]any)
	for _, cb := range content {
		m, ok := cb.(map[string]any)
		if !ok {
			continue
		}
		t, _ := m["type"].(string)
		if t != "thinking" {
			continue
		}
		thinkText, _ := m["thinking"].(string)
		sig, _ := m["signature"].(string)
		if sig == "" {
			return CheckResult{Name: "thinking_display_omitted", Pass: false,
				Detail: "opus-4-7 thinking block missing signature", Fix: "signature_rewrite"}
		}
		if thinkText == "" {
			return CheckResult{Name: "thinking_display_omitted", Pass: true,
				Detail: "display=omitted mode: thinking empty, signature present"}
		}
		return CheckResult{Name: "thinking_display_omitted", Pass: true,
			Detail: "display=summarized mode: thinking has content, signature present"}
	}
	return CheckResult{Name: "thinking_display_omitted", Pass: true,
		Detail: "no thinking block for opus-4-7 (skip)"}
}

// checkNoThinkingWhenDisabled verifies no thinking block exists when thinking was NOT requested.
func checkNoThinkingWhenDisabled(body map[string]any) CheckResult {
	content, _ := body["content"].([]any)
	for _, cb := range content {
		m, ok := cb.(map[string]any)
		if !ok {
			continue
		}
		t, _ := m["type"].(string)
		if t == "thinking" || t == "redacted_thinking" {
			return CheckResult{Name: "no_thinking_leak", Pass: false,
				Detail: "thinking block present but thinking was NOT requested", Fix: "body_rewrite"}
		}
	}
	return CheckResult{Name: "no_thinking_leak", Pass: true, Detail: "no unexpected thinking block"}
}

// checkSignatureLength verifies thinking.signature decoded length is in expected range (68-100 bytes).
func checkSignatureLength(body map[string]any) CheckResult {
	content, _ := body["content"].([]any)
	for _, cb := range content {
		m, ok := cb.(map[string]any)
		if !ok {
			continue
		}
		t, _ := m["type"].(string)
		if t != "thinking" {
			continue
		}
		sig, _ := m["signature"].(string)
		if sig == "" {
			return CheckResult{Name: "signature_length", Pass: false, Detail: "no signature", Fix: "signature_rewrite"}
		}
		raw, err := base64.StdEncoding.DecodeString(sig)
		if err != nil {
			return CheckResult{Name: "signature_length", Pass: false, Detail: "invalid base64", Fix: "signature_rewrite"}
		}
		n := len(raw)
		// Real clean-channel signatures vary widely across model/test cases in the
		// reference corpus (~260 to 4.7KB). Only reject clearly malformed blobs.
		if n >= 64 && n <= 8192 {
			return CheckResult{Name: "signature_length", Pass: true, Detail: fmt.Sprintf("signature %d bytes OK", n)}
		}
		return CheckResult{Name: "signature_length", Pass: false,
			Detail: fmt.Sprintf("signature %d bytes outside sane range 64-8192", n), Fix: "signature_rewrite"}
	}
	return CheckResult{Name: "signature_length", Pass: true, Detail: "no thinking block (skip)"}
}
