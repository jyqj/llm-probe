package convert

import (
	"testing"

	"bedrock-gateway/internal/types"
)

func TestMessagesToConverse(t *testing.T) {
	req := &types.MessagesRequest{
		Model:     "claude-opus-4-6",
		MaxTokens: 1024,
		System:    "You are a helpful assistant.",
		Messages: []types.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	conv, err := MessagesToConverse(req)
	if err != nil {
		t.Fatalf("MessagesToConverse failed: %v", err)
	}

	if len(conv.System) != 1 {
		t.Errorf("system blocks = %d, want 1", len(conv.System))
	}

	if len(conv.Messages) != 1 {
		t.Errorf("messages = %d, want 1", len(conv.Messages))
	}

	if conv.Messages[0].Role != "user" {
		t.Errorf("role = %s, want user", conv.Messages[0].Role)
	}

	if conv.InferenceConfig == nil || *conv.InferenceConfig.MaxTokens != 1024 {
		t.Error("max_tokens not set correctly")
	}
}

func TestConvertSystemArray(t *testing.T) {
	system := []any{
		map[string]any{"type": "text", "text": "block1"},
		map[string]any{"type": "text", "text": "block2", "cache_control": map[string]any{"type": "ephemeral"}},
	}

	blocks, err := convertSystem(system)
	if err != nil {
		t.Fatalf("convertSystem failed: %v", err)
	}

	if len(blocks) != 2 {
		t.Errorf("blocks = %d, want 2", len(blocks))
	}

	if blocks[1].CachePoint == nil {
		t.Error("cache_control should create CachePoint")
	}
}

func TestConvertContentBlock(t *testing.T) {
	tests := []struct {
		name  string
		block map[string]any
		check func(*types.ConverseContentBlock) bool
	}{
		{
			name:  "text block",
			block: map[string]any{"type": "text", "text": "hello"},
			check: func(cb *types.ConverseContentBlock) bool { return cb.Text != nil && *cb.Text == "hello" },
		},
		{
			name: "image block",
			block: map[string]any{
				"type": "image",
				"source": map[string]any{
					"media_type": "image/png",
					"data":       "base64data",
				},
			},
			check: func(cb *types.ConverseContentBlock) bool { return cb.Image != nil && cb.Image.Format == "png" },
		},
		{
			name: "tool_use block",
			block: map[string]any{
				"type":  "tool_use",
				"id":    "toolu_123",
				"name":  "test_tool",
				"input": map[string]any{"key": "value"},
			},
			check: func(cb *types.ConverseContentBlock) bool {
				return cb.ToolUse != nil && cb.ToolUse.Name == "test_tool"
			},
		},
		{
			name: "tool_result block",
			block: map[string]any{
				"type":        "tool_result",
				"tool_use_id": "toolu_123",
				"content":     "result text",
			},
			check: func(cb *types.ConverseContentBlock) bool {
				return cb.ToolResult != nil && cb.ToolResult.ToolUseID == "toolu_123"
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cb, err := convertContentBlock(tc.block)
			if err != nil {
				t.Fatalf("convertContentBlock failed: %v", err)
			}
			if !tc.check(cb) {
				t.Error("check failed")
			}
		})
	}
}

func TestConvertToolChoice(t *testing.T) {
	tests := []struct {
		name  string
		input any
		check func(*types.ConverseToolChoice) bool
	}{
		{
			name:  "auto string",
			input: "auto",
			check: func(tc *types.ConverseToolChoice) bool { return tc.Auto != nil },
		},
		{
			name:  "any string",
			input: "any",
			check: func(tc *types.ConverseToolChoice) bool { return tc.Any != nil },
		},
		{
			name:  "tool object",
			input: map[string]any{"type": "tool", "name": "my_tool"},
			check: func(tc *types.ConverseToolChoice) bool { return tc.Tool != nil && tc.Tool.Name == "my_tool" },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			choice, err := convertToolChoice(tc.input)
			if err != nil {
				t.Fatalf("convertToolChoice failed: %v", err)
			}
			if choice == nil || !tc.check(choice) {
				t.Error("check failed")
			}
		})
	}
}

func TestMediaTypeToFormat(t *testing.T) {
	tests := []struct {
		mediaType string
		want      string
	}{
		{"image/jpeg", "jpeg"},
		{"image/png", "png"},
		{"image/gif", "gif"},
		{"image/webp", "webp"},
		{"unknown/type", "png"},
	}

	for _, tc := range tests {
		got := mediaTypeToFormat(tc.mediaType)
		if got != tc.want {
			t.Errorf("mediaTypeToFormat(%q) = %s, want %s", tc.mediaType, got, tc.want)
		}
	}
}
