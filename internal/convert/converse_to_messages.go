package convert

import (
	"bedrock-gateway/internal/types"
)

// ConverseResponseToMessages converts a Bedrock Converse response to Anthropic Messages format.
func ConverseResponseToMessages(resp *types.ConverseResponse, model string) *types.MessagesResponse {
	out := &types.MessagesResponse{
		ID:          GenerateMessageID(),
		Type:        "message",
		Role:        "assistant",
		Model:       model,
		StopReason:  converseStopReasonToMessages(resp.StopReason),
		StopDetails: nil, // Bedrock Converse doesn't provide this
		ContextManagement: &types.ContextManagement{
			AppliedEdits: []any{},
		},
	}

	// Convert content blocks
	if resp.Output != nil && resp.Output.Message != nil {
		for _, cb := range resp.Output.Message.Content {
			out.Content = append(out.Content, converseContentBlockToMessages(cb))
		}
	}
	if out.Content == nil {
		out.Content = []types.ContentBlock{}
	}

	// Convert usage
	if resp.Usage != nil {
		out.Usage = &types.MessagesUsage{
			InputTokens:              resp.Usage.InputTokens,
			OutputTokens:             resp.Usage.OutputTokens,
			CacheCreationInputTokens: resp.Usage.CacheWriteInputTokens,
			CacheReadInputTokens:     resp.Usage.CacheReadInputTokens,
			CacheCreation: &types.CacheCreationDetail{
				Ephemeral5mInputTokens: 0,
				Ephemeral1hInputTokens: 0,
			},
			ServiceTier:  "standard",
			InferenceGeo: "not_available",
			Speed:        "standard",
		}
	}

	return out
}

func converseContentBlockToMessages(cb types.ConverseContentBlock) types.ContentBlock {
	if cb.Text != nil {
		return types.ContentBlock{Type: "text", Text: *cb.Text}
	}

	if cb.ToolUse != nil {
		return types.ContentBlock{
			Type:   "tool_use",
			ID:     cb.ToolUse.ToolUseID,
			Name:   cb.ToolUse.Name,
			Input:  cb.ToolUse.Input,
			Caller: &types.CallerInfo{Type: "direct"},
		}
	}

	if cb.ReasoningContent != nil {
		if cb.ReasoningContent.ReasoningText != nil {
			return types.ContentBlock{
				Type:      "thinking",
				Thinking:  cb.ReasoningContent.ReasoningText.Text,
				Signature: cb.ReasoningContent.ReasoningText.Signature,
			}
		}
		if cb.ReasoningContent.RedactedContent != nil {
			return types.ContentBlock{
				Type: "redacted_thinking",
				Data: *cb.ReasoningContent.RedactedContent,
			}
		}
	}

	if cb.Image != nil {
		return types.ContentBlock{
			Type: "image",
			Source: &types.ImageSource{
				Type:      "base64",
				MediaType: formatToMediaType(cb.Image.Format),
				Data:      cb.Image.Source.Bytes,
			},
		}
	}

	return types.ContentBlock{Type: "text", Text: ""}
}

func converseStopReasonToMessages(reason string) string {
	switch reason {
	case types.StopReasonEndTurn:
		return "end_turn"
	case types.StopReasonMaxTokens:
		return "max_tokens"
	case types.StopReasonToolUse:
		return "tool_use"
	case types.StopReasonStopSequence:
		return "stop_sequence"
	default:
		return reason
	}
}

func formatToMediaType(format string) string {
	switch format {
	case "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	default:
		return "image/png"
	}
}
