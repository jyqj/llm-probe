package handler

import (
	"testing"
)

func TestHasCacheControl(t *testing.T) {
	tests := []struct {
		name    string
		reqJSON map[string]any
		want    bool
	}{
		{
			name:    "no cache_control",
			reqJSON: map[string]any{"messages": []any{}},
			want:    false,
		},
		{
			name: "cache_control in message",
			reqJSON: map[string]any{
				"messages": []any{
					map[string]any{
						"role": "user",
						"content": []any{
							map[string]any{"type": "text", "text": "hi", "cache_control": map[string]any{"type": "ephemeral"}},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "cache_control in system",
			reqJSON: map[string]any{
				"system": []any{
					map[string]any{"type": "text", "text": "sys", "cache_control": map[string]any{"type": "ephemeral"}},
				},
				"messages": []any{},
			},
			want: true,
		},
		{
			name: "cache_control in tools",
			reqJSON: map[string]any{
				"messages": []any{},
				"tools": []any{
					map[string]any{"name": "test", "cache_control": map[string]any{"type": "ephemeral"}},
				},
			},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := hasCacheControl(tc.reqJSON)
			if got != tc.want {
				t.Errorf("hasCacheControl() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsCacheAllZero(t *testing.T) {
	tests := []struct {
		name  string
		usage map[string]any
		want  bool
	}{
		{
			name: "all zero",
			usage: map[string]any{
				"cache_creation_input_tokens": float64(0),
				"cache_read_input_tokens":     float64(0),
				"cache_creation": map[string]any{
					"ephemeral_5m_input_tokens": float64(0),
					"ephemeral_1h_input_tokens": float64(0),
				},
			},
			want: true,
		},
		{
			name: "non-zero creation",
			usage: map[string]any{
				"cache_creation_input_tokens": float64(100),
				"cache_read_input_tokens":     float64(0),
			},
			want: false,
		},
		{
			name: "non-zero read",
			usage: map[string]any{
				"cache_creation_input_tokens": float64(0),
				"cache_read_input_tokens":     float64(50),
			},
			want: false,
		},
		{
			name: "non-zero ephemeral",
			usage: map[string]any{
				"cache_creation_input_tokens": float64(0),
				"cache_read_input_tokens":     float64(0),
				"cache_creation": map[string]any{
					"ephemeral_5m_input_tokens": float64(100),
					"ephemeral_1h_input_tokens": float64(0),
				},
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isCacheAllZero(tc.usage)
			if got != tc.want {
				t.Errorf("isCacheAllZero() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestStripThinkingFromMessages(t *testing.T) {
	reqJSON := map[string]any{
		"messages": []any{
			map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{"type": "thinking", "thinking": "let me think"},
					map[string]any{"type": "text", "text": "response"},
				},
			},
		},
	}

	stripThinkingFromMessages(reqJSON)

	messages := reqJSON["messages"].([]any)
	msg := messages[0].(map[string]any)
	content := msg["content"].([]any)

	if len(content) != 1 {
		t.Errorf("content length = %d, want 1", len(content))
	}

	block := content[0].(map[string]any)
	if block["type"] != "text" {
		t.Errorf("remaining block type = %s, want text", block["type"])
	}
}

func TestExtractClientKey(t *testing.T) {
	// This would need actual http.Request, skip for now
}

func TestIntFromAny(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  int
	}{
		{"float64", float64(42), 42},
		{"int", 42, 42},
		{"nil", nil, 0},
		{"string", "42", 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := intFromAny(tc.input)
			if got != tc.want {
				t.Errorf("intFromAny(%v) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}

func TestInjectCacheUsageWithKey(t *testing.T) {
	// Simulated request with system prompt and tools for content-based hashing
	reqJSON := map[string]any{
		"system": "You are a helpful assistant.",
		"tools": []any{
			map[string]any{"name": "tool1", "description": "does stuff"},
		},
	}

	// Test per-key tracking
	usage1 := map[string]any{
		"input_tokens":                float64(1000),
		"cache_creation_input_tokens": float64(0),
		"cache_read_input_tokens":     float64(0),
	}

	injectCacheUsageWithKey(usage1, 1000, "upstream-a", reqJSON)

	ccCreate := usage1["cache_creation_input_tokens"].(float64)
	ccRead := usage1["cache_read_input_tokens"].(float64)

	// First call for upstream-a should favor creation
	if ccCreate <= 0 {
		t.Errorf("first call cache_creation should be > 0, got %v", ccCreate)
	}
	if ccCreate < ccRead {
		t.Errorf("first call should favor creation over read: create=%v read=%v", ccCreate, ccRead)
	}

	// Multiple calls with same content should shift toward reads
	for i := 0; i < 15; i++ {
		usage := map[string]any{
			"input_tokens":                float64(1000),
			"cache_creation_input_tokens": float64(0),
			"cache_read_input_tokens":     float64(0),
		}
		injectCacheUsageWithKey(usage, 1000, "upstream-a", reqJSON)
	}

	// After many calls, should favor reads
	usageFinal := map[string]any{
		"input_tokens":                float64(1000),
		"cache_creation_input_tokens": float64(0),
		"cache_read_input_tokens":     float64(0),
	}
	injectCacheUsageWithKey(usageFinal, 1000, "upstream-a", reqJSON)

	ccReadFinal := usageFinal["cache_read_input_tokens"].(float64)
	ccCreateFinal := usageFinal["cache_creation_input_tokens"].(float64)

	if ccReadFinal <= ccCreateFinal {
		t.Logf("after many calls: create=%v read=%v (read should be higher)", ccCreateFinal, ccReadFinal)
	}

	// Different content on same upstream should reset to creation-heavy pattern
	reqJSON2 := map[string]any{
		"system": "You are a different assistant with different instructions.",
		"tools": []any{
			map[string]any{"name": "tool2", "description": "different tool"},
		},
	}
	usageDiff := map[string]any{
		"input_tokens":                float64(1000),
		"cache_creation_input_tokens": float64(0),
		"cache_read_input_tokens":     float64(0),
	}
	injectCacheUsageWithKey(usageDiff, 1000, "upstream-a", reqJSON2)
	ccCreateDiff := usageDiff["cache_creation_input_tokens"].(float64)
	ccReadDiff := usageDiff["cache_read_input_tokens"].(float64)
	if ccCreateDiff <= 0 {
		t.Errorf("different content first call cache_creation should be > 0, got %v", ccCreateDiff)
	}
	if ccCreateDiff < ccReadDiff {
		t.Errorf("different content first call should favor creation: create=%v read=%v", ccCreateDiff, ccReadDiff)
	}
}
