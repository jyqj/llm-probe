package convert

import (
	"encoding/json"
	"fmt"

	"bedrock-gateway/internal/types"
)

// MessagesToConverse converts an Anthropic Messages API request to a Bedrock Converse request.
func MessagesToConverse(req *types.MessagesRequest) (*types.ConverseRequest, error) {
	out := &types.ConverseRequest{}

	// Convert system prompt
	if req.System != nil {
		system, err := convertSystem(req.System)
		if err != nil {
			return nil, fmt.Errorf("convert system: %w", err)
		}
		out.System = system
	}

	// Convert messages
	for i, msg := range req.Messages {
		cm, err := convertMessage(msg)
		if err != nil {
			return nil, fmt.Errorf("convert message[%d]: %w", i, err)
		}
		out.Messages = append(out.Messages, *cm)
	}

	// Convert inference config
	out.InferenceConfig = &types.InferenceConfig{
		MaxTokens: &req.MaxTokens,
	}
	if req.Temperature != nil {
		out.InferenceConfig.Temperature = req.Temperature
	}
	if req.TopP != nil {
		out.InferenceConfig.TopP = req.TopP
	}
	if len(req.StopSequences) > 0 {
		out.InferenceConfig.StopSequences = req.StopSequences
	}

	// Convert tools
	if len(req.Tools) > 0 {
		tc, err := convertTools(req.Tools, req.ToolChoice)
		if err != nil {
			return nil, fmt.Errorf("convert tools: %w", err)
		}
		out.ToolConfig = tc
	}

	// Convert additional fields (thinking, output_config, top_k, etc.)
	// These go into additionalModelRequestFields for Converse API
	additional := make(map[string]any)

	if req.TopK != nil {
		additional["top_k"] = *req.TopK
	}

	if req.Thinking != nil {
		additional["thinking"] = req.Thinking
	}

	if req.OutputConfig != nil {
		additional["output_config"] = req.OutputConfig
	}

	if len(additional) > 0 {
		out.AdditionalModelRequestFields = additional
	}

	return out, nil
}

func convertSystem(system any) ([]types.ConverseSystemBlock, error) {
	switch v := system.(type) {
	case string:
		return []types.ConverseSystemBlock{{Text: &v}}, nil

	case []any:
		var blocks []types.ConverseSystemBlock
		for _, item := range v {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if text, ok := m["text"].(string); ok {
				block := types.ConverseSystemBlock{Text: &text}
				// Handle cache_control
				if cc, ok := m["cache_control"]; ok {
					if ccMap, ok := cc.(map[string]any); ok {
						if ccType, ok := ccMap["type"].(string); ok && ccType == "ephemeral" {
							block.CachePoint = &types.CachePointBlock{Type: "default"}
						}
					}
				}
				blocks = append(blocks, block)
			}
		}
		return blocks, nil

	default:
		data, _ := json.Marshal(system)
		return nil, fmt.Errorf("unsupported system type: %s", string(data))
	}
}

func convertMessage(msg types.Message) (*types.ConverseMessage, error) {
	cm := &types.ConverseMessage{Role: msg.Role}

	switch content := msg.Content.(type) {
	case string:
		cm.Content = []types.ConverseContentBlock{{Text: &content}}

	case []any:
		for _, item := range content {
			block, ok := item.(map[string]any)
			if !ok {
				continue
			}
			cb, err := convertContentBlock(block)
			if err != nil {
				return nil, err
			}
			if cb != nil {
				cm.Content = append(cm.Content, *cb)
			}
		}

	default:
		// Try to re-marshal and re-parse as []ContentBlock
		data, err := json.Marshal(content)
		if err != nil {
			return nil, fmt.Errorf("marshal content: %w", err)
		}
		var blocks []map[string]any
		if err := json.Unmarshal(data, &blocks); err != nil {
			return nil, fmt.Errorf("content is neither string nor array: %w", err)
		}
		for _, block := range blocks {
			cb, err := convertContentBlock(block)
			if err != nil {
				return nil, err
			}
			if cb != nil {
				cm.Content = append(cm.Content, *cb)
			}
		}
	}

	return cm, nil
}

func convertContentBlock(block map[string]any) (*types.ConverseContentBlock, error) {
	blockType, _ := block["type"].(string)

	switch blockType {
	case "text":
		text, _ := block["text"].(string)
		cb := &types.ConverseContentBlock{Text: &text}
		// Handle cache_control -> cachePoint
		if _, ok := block["cache_control"]; ok {
			cb.CachePoint = &types.CachePointBlock{Type: "default"}
		}
		return cb, nil

	case "image":
		source, ok := block["source"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("image block missing source")
		}
		mediaType, _ := source["media_type"].(string)
		data, _ := source["data"].(string)

		// Map media type to format
		format := mediaTypeToFormat(mediaType)

		return &types.ConverseContentBlock{
			Image: &types.ConverseImageBlock{
				Format: format,
				Source: &types.ConverseSource{Bytes: data},
			},
		}, nil

	case "tool_use":
		id, _ := block["id"].(string)
		name, _ := block["name"].(string)
		input := block["input"]

		return &types.ConverseContentBlock{
			ToolUse: &types.ConverseToolUseBlock{
				ToolUseID: id,
				Name:      name,
				Input:     input,
			},
		}, nil

	case "tool_result":
		toolUseID, _ := block["tool_use_id"].(string)
		isError, _ := block["is_error"].(bool)

		tr := &types.ConverseToolResultBlock{
			ToolUseID: toolUseID,
		}
		if isError {
			tr.Status = "error"
		} else {
			tr.Status = "success"
		}

		// Convert tool_result content
		switch c := block["content"].(type) {
		case string:
			tr.Content = []types.ConverseContentBlock{{Text: &c}}
		case []any:
			for _, item := range c {
				if m, ok := item.(map[string]any); ok {
					cb, err := convertContentBlock(m)
					if err != nil {
						return nil, err
					}
					if cb != nil {
						tr.Content = append(tr.Content, *cb)
					}
				}
			}
		}

		return &types.ConverseContentBlock{ToolResult: tr}, nil

	case "document":
		doc, ok := block["source"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("document block missing source")
		}
		mediaType, _ := doc["media_type"].(string)
		data, _ := doc["data"].(string)
		name, _ := block["name"].(string)

		return &types.ConverseContentBlock{
			Document: &types.ConverseDocumentBlock{
				Format: docMediaTypeToFormat(mediaType),
				Name:   name,
				Source: &types.ConverseSource{Bytes: data},
			},
		}, nil

	case "thinking":
		thinking, _ := block["thinking"].(string)
		signature, _ := block["signature"].(string)
		return &types.ConverseContentBlock{
			ReasoningContent: &types.ReasoningContentBlock{
				ReasoningText: &types.ReasoningTextBlock{
					Text:      thinking,
					Signature: signature,
				},
			},
		}, nil

	case "redacted_thinking":
		data, _ := block["data"].(string)
		return &types.ConverseContentBlock{
			ReasoningContent: &types.ReasoningContentBlock{
				RedactedContent: &data,
			},
		}, nil

	default:
		// Unknown block type - pass through as text if possible
		if text, ok := block["text"].(string); ok {
			return &types.ConverseContentBlock{Text: &text}, nil
		}
		return nil, fmt.Errorf("unsupported content block type: %s", blockType)
	}
}

func convertTools(tools []types.Tool, toolChoice any) (*types.ToolConfig, error) {
	tc := &types.ToolConfig{}

	for _, tool := range tools {
		tc.Tools = append(tc.Tools, types.ConverseToolDef{
			ToolSpec: &types.ToolSpec{
				Name:        tool.Name,
				Description: tool.Description,
				InputSchema: &types.ToolInputSchema{JSON: tool.InputSchema},
			},
		})
	}

	// Convert tool_choice
	if toolChoice != nil {
		choice, err := convertToolChoice(toolChoice)
		if err != nil {
			return nil, err
		}
		tc.ToolChoice = choice
	}

	return tc, nil
}

func convertToolChoice(tc any) (*types.ConverseToolChoice, error) {
	switch v := tc.(type) {
	case string:
		switch v {
		case "auto":
			return &types.ConverseToolChoice{Auto: &struct{}{}}, nil
		case "any":
			return &types.ConverseToolChoice{Any: &struct{}{}}, nil
		case "none":
			return nil, nil // Converse API doesn't have "none", omit toolChoice
		default:
			return nil, fmt.Errorf("unknown tool_choice string: %s", v)
		}

	case map[string]any:
		t, _ := v["type"].(string)
		switch t {
		case "auto":
			return &types.ConverseToolChoice{Auto: &struct{}{}}, nil
		case "any":
			return &types.ConverseToolChoice{Any: &struct{}{}}, nil
		case "tool":
			name, _ := v["name"].(string)
			return &types.ConverseToolChoice{
				Tool: &types.ConverseToolChoiceTool{Name: name},
			}, nil
		default:
			return &types.ConverseToolChoice{Auto: &struct{}{}}, nil
		}

	default:
		return &types.ConverseToolChoice{Auto: &struct{}{}}, nil
	}
}

func mediaTypeToFormat(mediaType string) string {
	switch mediaType {
	case "image/jpeg":
		return "jpeg"
	case "image/png":
		return "png"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	default:
		return "png"
	}
}

func docMediaTypeToFormat(mediaType string) string {
	switch mediaType {
	case "application/pdf":
		return "pdf"
	case "text/csv":
		return "csv"
	case "text/html":
		return "html"
	case "text/plain":
		return "txt"
	case "text/markdown":
		return "md"
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return "docx"
	case "application/msword":
		return "doc"
	default:
		return "txt"
	}
}
