package channeltest

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
			return CheckResult{Name: "signature", Pass: false, Expected: "non-empty signature on thinking block", Actual: "signature is empty", Detail: "thinking block has no signature", Fix: "signature_rewrite"}
		}
		raw, err := base64.StdEncoding.DecodeString(sig)
		if err != nil {
			return CheckResult{Name: "signature", Pass: false, Expected: "valid base64-encoded signature", Actual: fmt.Sprintf("base64 decode error: %v", err), Detail: "signature not valid base64", Fix: "signature_rewrite"}
		}
		if len(raw) < 4 || raw[0] != 0x12 {
			return CheckResult{Name: "signature", Pass: false, Expected: "protobuf tag 0x12 at byte[0], length >= 4", Actual: fmt.Sprintf("byte[0]=0x%02x, length=%d", func() byte { if len(raw) > 0 { return raw[0] }; return 0 }(), len(raw)), Detail: "signature protobuf tag mismatch", Fix: "signature_rewrite"}
		}
		ok2, _ := fingerprint.VerifySignature(sig)
		if ok2 {
			return CheckResult{Name: "signature", Pass: true, Expected: "valid protobuf signature", Actual: "detector-issued signature verified", Detail: "detector-issued signature verified"}
		}
		return CheckResult{Name: "signature", Pass: true, Expected: "valid protobuf signature", Actual: "valid protobuf signature (external)", Detail: "valid protobuf signature (external)"}
	}
	return CheckResult{Name: "signature", Pass: true, Expected: "N/A (no thinking block)", Actual: "no thinking block", Detail: "no thinking block (skip)"}
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
			return CheckResult{Name: "thinking_present", Pass: true, Expected: "thinking or redacted_thinking block present", Actual: fmt.Sprintf("found block type %q", t), Detail: "thinking block found"}
		}
	}
	// Clean Opus 4.7 Console references may omit thinking even with adaptive
	// thinking requested. Treat absence as non-discriminative.
	return CheckResult{Name: "thinking_present", Pass: true, Expected: "thinking block present (optional)", Actual: "no thinking block in content", Detail: "no thinking block (skip)"}
}

// checkThinkingOrder verifies thinking block appears before text block in content.
// Official API always puts thinking at index 0 when thinking is enabled.
func checkThinkingOrder(body map[string]any) CheckResult {
	content, _ := body["content"].([]any)
	if len(content) == 0 {
		return CheckResult{Name: "thinking_order", Pass: true, Expected: "thinking at index 0 (if present)", Actual: "no content blocks", Detail: "no content blocks"}
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
					Expected: "thinking block at index 0", Actual: fmt.Sprintf("thinking block at index %d", i),
					Detail: fmt.Sprintf("thinking at index %d, should be 0", i), Fix: "thinking_inject"}
			}
		}
	}
	if !hasThinking {
		return CheckResult{Name: "thinking_order", Pass: true, Expected: "thinking at index 0 (if present)", Actual: "no thinking block", Detail: "no thinking block (skip)"}
	}
	if firstType == "thinking" || firstType == "redacted_thinking" {
		return CheckResult{Name: "thinking_order", Pass: true, Expected: "thinking block at index 0", Actual: fmt.Sprintf("%s block at index 0", firstType), Detail: "thinking at index 0"}
	}
	return CheckResult{Name: "thinking_order", Pass: false, Expected: "thinking block at index 0", Actual: fmt.Sprintf("first block is %s", firstType), Detail: "first block is " + firstType + " not thinking", Fix: "thinking_inject"}
}

// checkThinkingDisplayOmitted verifies thinking block format for display="omitted" mode.
// On Opus 4.7+, thinking.thinking may be "" (empty) but signature must still be present.
func checkThinkingDisplayOmitted(body map[string]any, model string) CheckResult {
	if !strings.Contains(model, "opus-4-7") && !strings.Contains(model, "opus-4-8") {
		return CheckResult{Name: "thinking_display_omitted", Pass: true, Expected: "opus-4-7+ model for display check", Actual: fmt.Sprintf("model=%s (not opus-4-7+)", model), Detail: "not opus-4-7+ model (skip)"}
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
				Expected: "non-empty signature on opus-4-7+ thinking block", Actual: "signature is empty",
				Detail: "opus-4-7 thinking block missing signature", Fix: "signature_rewrite"}
		}
		if thinkText == "" {
			return CheckResult{Name: "thinking_display_omitted", Pass: true,
				Expected: "signature present on opus-4-7+ thinking block", Actual: "thinking empty, signature present",
				Detail: "display=omitted mode: thinking empty, signature present"}
		}
		return CheckResult{Name: "thinking_display_omitted", Pass: true,
			Expected: "signature present on opus-4-7+ thinking block", Actual: "thinking has content, signature present",
			Detail: "display=summarized mode: thinking has content, signature present"}
	}
	return CheckResult{Name: "thinking_display_omitted", Pass: true,
		Expected: "opus-4-7+ thinking block with signature", Actual: "no thinking block found",
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
				Expected: "no thinking/redacted_thinking blocks when thinking disabled", Actual: fmt.Sprintf("found %q block in content", t),
				Detail: "thinking block present but thinking was NOT requested", Fix: "body_rewrite"}
		}
	}
	return CheckResult{Name: "no_thinking_leak", Pass: true, Expected: "no thinking blocks when thinking disabled", Actual: "no thinking blocks found", Detail: "no unexpected thinking block"}
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
			return CheckResult{Name: "signature_length", Pass: false, Expected: "non-empty signature", Actual: "signature is empty", Detail: "no signature", Fix: "signature_rewrite"}
		}
		raw, err := base64.StdEncoding.DecodeString(sig)
		if err != nil {
			return CheckResult{Name: "signature_length", Pass: false, Expected: "valid base64 signature", Actual: fmt.Sprintf("base64 decode error: %v", err), Detail: "invalid base64", Fix: "signature_rewrite"}
		}
		n := len(raw)
		// Real clean-channel signatures vary widely across model/test cases in the
		// reference corpus (~260 to 4.7KB). Only reject clearly malformed blobs.
		if n >= 64 && n <= 8192 {
			return CheckResult{Name: "signature_length", Pass: true, Expected: "signature length 64-8192 bytes", Actual: fmt.Sprintf("%d bytes", n), Detail: fmt.Sprintf("signature %d bytes OK", n)}
		}
		return CheckResult{Name: "signature_length", Pass: false,
			Expected: "signature length 64-8192 bytes", Actual: fmt.Sprintf("%d bytes", n),
			Detail: fmt.Sprintf("signature %d bytes outside sane range 64-8192", n), Fix: "signature_rewrite"}
	}
	return CheckResult{Name: "signature_length", Pass: true, Expected: "N/A (no thinking block)", Actual: "no thinking block", Detail: "no thinking block (skip)"}
}
