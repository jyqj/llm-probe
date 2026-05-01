package handler

import (
	"fmt"
	"time"
)

// modelIdentity maps model names to (display_name, knowledge_cutoff).
// NOTE: Update cutoff dates when new model versions are released.
var modelIdentity = map[string][2]string{
	"claude-opus-4-7":   {"Claude Opus 4.7", "January 2026"},
	"claude-opus-4-6":   {"Claude Opus 4.6", "July 2025"},
	"claude-opus-4-5":   {"Claude Opus 4.5", "March 2025"},
	"claude-sonnet-4-6": {"Claude Sonnet 4.6", "July 2025"},
	"claude-sonnet-4-5": {"Claude Sonnet 4.5", "March 2025"},
	"claude-haiku-4-5":  {"Claude Haiku 4.5", "February 2025"},
}

// identityPrompt returns the identity system prompt for a model.
func identityPrompt(model string) string {
	if info, ok := modelIdentity[model]; ok {
		return fmt.Sprintf("You are %s by Anthropic (knowledge cutoff: %s). Today's date is %s.",
			info[0], info[1], time.Now().Format("January 2, 2006"))
	}
	// Prefix match for date-suffixed models like claude-haiku-4-5-20251001
	for prefix, info := range modelIdentity {
		if len(model) > len(prefix) && model[:len(prefix)] == prefix {
			return fmt.Sprintf("You are %s by Anthropic (knowledge cutoff: %s). Today's date is %s.",
				info[0], info[1], time.Now().Format("January 2, 2006"))
		}
	}
	// Fallback: use generic identity without exposing raw model name
	return fmt.Sprintf("You are Claude by Anthropic. Today's date is %s.",
		time.Now().Format("January 2, 2006"))
}

// injectIdentity prepends an identity system prompt into the request.
// Returns estimated token count of injected text.
func injectIdentity(reqJSON map[string]any, model string) int {
	text := identityPrompt(model)

	existing := reqJSON["system"]
	switch v := existing.(type) {
	case nil:
		reqJSON["system"] = []any{map[string]any{"type": "text", "text": text}}
	case string:
		if v == "" {
			reqJSON["system"] = []any{map[string]any{"type": "text", "text": text}}
		} else {
			reqJSON["system"] = text + "\n\n" + v
		}
	case []any:
		reqJSON["system"] = append([]any{map[string]any{"type": "text", "text": text}}, v...)
	}
	return len(text) / 4
}

// estimateInputTokens roughly estimates the original input tokens.
func estimateInputTokens(reqJSON map[string]any) int {
	totalChars := 0
	messages, _ := reqJSON["messages"].([]any)
	for _, msg := range messages {
		m, ok := msg.(map[string]any)
		if !ok {
			continue
		}
		switch c := m["content"].(type) {
		case string:
			totalChars += len(c)
		case []any:
			for _, block := range c {
				if b, ok := block.(map[string]any); ok {
					if text, _ := b["text"].(string); text != "" {
						totalChars += len(text)
					}
				}
			}
		}
	}
	// Also count system prompt
	switch s := reqJSON["system"].(type) {
	case string:
		totalChars += len(s)
	case []any:
		for _, block := range s {
			if b, ok := block.(map[string]any); ok {
				if text, _ := b["text"].(string); text != "" {
					totalChars += len(text)
				}
			}
		}
	}
	est := totalChars/4 + 6
	if est < 8 {
		est = 8
	}
	return est
}
