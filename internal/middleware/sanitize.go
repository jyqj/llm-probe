package middleware

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"bedrock-gateway/internal/config"
	"bedrock-gateway/internal/types"
)

// SanitizeResult holds the sanitization outcome.
type SanitizeResult struct {
	Modified     bool     // whether the request was modified
	Warnings     []string // non-fatal warnings
	Blocked      bool     // whether the request should be rejected
	BlockReason  string   // reason for blocking
}

// Sanitizer validates and cleans incoming requests.
type Sanitizer struct {
	cfg config.SanitizeConfig
}

// NewSanitizer creates a new Sanitizer.
func NewSanitizer(cfg config.SanitizeConfig) *Sanitizer {
	return &Sanitizer{cfg: cfg}
}

// SanitizeMessagesRequest checks and cleans an Anthropic Messages API request.
func (s *Sanitizer) SanitizeMessagesRequest(req *types.MessagesRequest) *SanitizeResult {
	result := &SanitizeResult{}

	// 1. Check system prompt injection
	if s.cfg.BlockSystemPromptInjection {
		s.checkSystemPrompt(req, result)
	}

	// 2. Validate and fix output_config structure
	s.fixOutputConfig(req, result)

	// 3. Validate messages structure
	s.validateMessages(req, result)

	// 4. Normalize model name
	s.normalizeModel(req, result)

	// 5. Validate inference parameters
	s.validateInferenceParams(req, result)

	// 6. Validate thinking config
	s.validateThinking(req, result)

	return result
}

// checkSystemPrompt detects potentially injected system prompts.
func (s *Sanitizer) checkSystemPrompt(req *types.MessagesRequest, result *SanitizeResult) {
	if req.System == nil {
		return
	}

	var systemText string

	switch v := req.System.(type) {
	case string:
		systemText = v
	case []any:
		// Array of system blocks - extract text
		for _, block := range v {
			if m, ok := block.(map[string]any); ok {
				if t, ok := m["text"].(string); ok {
					systemText += t + "\n"
				}
			}
		}
	}

	if systemText == "" {
		return
	}

	// Check against allowed hashes if configured
	if len(s.cfg.AllowedSystemPromptHashes) > 0 {
		hash := fmt.Sprintf("%x", sha256.Sum256([]byte(strings.TrimSpace(systemText))))
		allowed := false
		for _, h := range s.cfg.AllowedSystemPromptHashes {
			if h == hash {
				allowed = true
				break
			}
		}
		if !allowed {
			result.Blocked = true
			result.BlockReason = fmt.Sprintf("system prompt hash %s not in allowlist", hash[:16])
			return
		}
	}

	// Heuristic detection of injection patterns
	injectionPatterns := []string{
		"ignore previous instructions",
		"ignore all previous",
		"disregard previous",
		"forget your instructions",
		"you are now",
		"new instructions:",
		"override:",
		"system override",
		"jailbreak",
		"DAN mode",
	}

	lowerSystem := strings.ToLower(systemText)
	for _, pattern := range injectionPatterns {
		if strings.Contains(lowerSystem, strings.ToLower(pattern)) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("potential injection pattern detected in system prompt: %q", pattern))
		}
	}
}

// fixOutputConfig handles output_config by injecting JSON schema constraints
// into the system prompt, since most upstream providers don't support output_config natively.
func (s *Sanitizer) fixOutputConfig(req *types.MessagesRequest, result *SanitizeResult) {
	if req.OutputConfig == nil {
		return
	}
	if req.OutputConfig.Format == nil {
		req.OutputConfig = nil
		return
	}

	if req.OutputConfig.Format.Type != "json_schema" || req.OutputConfig.Format.Schema == nil {
		req.OutputConfig = nil
		return
	}

	// Serialize the schema to inject into system prompt
	schemaBytes, err := json.Marshal(req.OutputConfig.Format.Schema)
	if err != nil {
		result.Warnings = append(result.Warnings, "failed to marshal output_config schema")
		req.OutputConfig = nil
		return
	}

	// Build the injection text
	schemaPrompt := "You must respond with ONLY a valid JSON object (no markdown, no explanation, no extra text) that strictly conforms to this JSON Schema:\n" + string(schemaBytes)

	// Inject into system prompt
	switch v := req.System.(type) {
	case nil:
		req.System = schemaPrompt
	case string:
		req.System = v + "\n\n" + schemaPrompt
	case []any:
		// Array of system blocks — append a new text block
		req.System = append(v, map[string]any{
			"type": "text",
			"text": schemaPrompt,
		})
	default:
		// Unknown format, overwrite
		req.System = schemaPrompt
	}

	// Remove output_config so it's not sent to upstream (which would ignore or 502)
	req.OutputConfig = nil
	result.Modified = true
	result.Warnings = append(result.Warnings, "output_config.json_schema injected into system prompt")
}

// validateMessages checks message structure integrity.
func (s *Sanitizer) validateMessages(req *types.MessagesRequest, result *SanitizeResult) {
	if len(req.Messages) == 0 {
		result.Blocked = true
		result.BlockReason = "messages array is empty"
		return
	}

	// First message must be from user
	if req.Messages[0].Role != "user" {
		result.Warnings = append(result.Warnings, "first message should be from user")
	}

	// Check for proper role alternation (some providers break this)
	for i := 1; i < len(req.Messages); i++ {
		if req.Messages[i].Role == req.Messages[i-1].Role {
			// Consecutive same-role messages - this is allowed in some cases
			// (e.g., tool_result after assistant), but flag it
			if req.Messages[i].Role == "user" && req.Messages[i-1].Role == "user" {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("consecutive user messages at index %d and %d", i-1, i))
			}
		}
	}

	// Normalize content: if content is a bare string, wrap it
	for i := range req.Messages {
		if str, ok := req.Messages[i].Content.(string); ok {
			req.Messages[i].Content = []map[string]any{
				{"type": "text", "text": str},
			}
			result.Modified = true
		}
	}
}

// normalizeModel cleans up the model field.
func (s *Sanitizer) normalizeModel(req *types.MessagesRequest, result *SanitizeResult) {
	if req.Model == "" {
		result.Warnings = append(result.Warnings, "model field is empty")
	}
	// Strip any provider prefix that some channels add
	req.Model = strings.TrimPrefix(req.Model, "anthropic/")
	req.Model = strings.TrimPrefix(req.Model, "bedrock/")
}

// validateInferenceParams checks parameter ranges.
func (s *Sanitizer) validateInferenceParams(req *types.MessagesRequest, result *SanitizeResult) {
	if req.Temperature != nil {
		t := *req.Temperature
		if t < 0 {
			zero := 0.0
			req.Temperature = &zero
			result.Modified = true
		} else if t > 1.0 {
			one := 1.0
			req.Temperature = &one
			result.Modified = true
			result.Warnings = append(result.Warnings, "temperature clamped to 1.0")
		}
	}

	if req.TopP != nil {
		p := *req.TopP
		if p < 0 || p > 1.0 {
			req.TopP = nil
			result.Modified = true
			result.Warnings = append(result.Warnings, "invalid top_p removed")
		}
	}

	if req.MaxTokens <= 0 {
		req.MaxTokens = 4096
		result.Modified = true
		result.Warnings = append(result.Warnings, "max_tokens set to default 4096")
	}
}

// validateThinking validates extended thinking config.
func (s *Sanitizer) validateThinking(req *types.MessagesRequest, result *SanitizeResult) {
	if req.Thinking == nil {
		return
	}

	if req.Thinking.Type == "enabled" {
		// Extended thinking requires no temperature or temperature=1
		if req.Temperature != nil && *req.Temperature != 1.0 {
			one := 1.0
			req.Temperature = &one
			result.Modified = true
			result.Warnings = append(result.Warnings, "temperature set to 1.0 for extended thinking")
		}

		if req.Thinking.BudgetTokens <= 0 {
			req.Thinking.BudgetTokens = 1024
			result.Modified = true
		}
	}
}

// SanitizeConverseRequest checks and cleans a Bedrock Converse request.
func (s *Sanitizer) SanitizeConverseRequest(req *types.ConverseRequest) *SanitizeResult {
	result := &SanitizeResult{}

	// Check system prompt injection in Converse format
	if s.cfg.BlockSystemPromptInjection && len(req.System) > 0 {
		var systemText string
		for _, block := range req.System {
			if block.Text != nil {
				systemText += *block.Text + "\n"
			}
		}
		if systemText != "" {
			// Reuse the same injection detection
			tempReq := &types.MessagesRequest{System: systemText}
			s.checkSystemPrompt(tempReq, result)
		}
	}

	// Validate messages
	if len(req.Messages) == 0 {
		result.Blocked = true
		result.BlockReason = "messages array is empty"
		return result
	}

	// Validate inference config ranges
	if req.InferenceConfig != nil {
		if req.InferenceConfig.Temperature != nil {
			t := *req.InferenceConfig.Temperature
			if t < 0 || t > 1.0 {
				clamped := max(0, min(1.0, t))
				req.InferenceConfig.Temperature = &clamped
				result.Modified = true
			}
		}
	}

	return result
}

// ToJSON is a helper to serialize for logging.
func (r *SanitizeResult) ToJSON() string {
	b, _ := json.Marshal(r)
	return string(b)
}
