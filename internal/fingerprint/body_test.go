package fingerprint

import (
	"encoding/json"
	"strings"
	"testing"

	"bedrock-gateway/internal/config"
)

func TestOrderedMap(t *testing.T) {
	om := NewOrderedMap()
	om.Set("model", "claude-opus-4-6")
	om.Set("id", "msg_01ABC")
	om.Set("type", "message")

	data, err := json.Marshal(om)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	s := string(data)
	// Check order: model should come before id
	modelIdx := strings.Index(s, `"model"`)
	idIdx := strings.Index(s, `"id"`)
	if modelIdx > idIdx {
		t.Errorf("model should appear before id, got: %s", s)
	}
}

func TestFixMessageBody(t *testing.T) {
	body := map[string]any{
		"id":          "msg_bdrk_123",
		"model":       "claude-opus-4-6",
		"type":        "message",
		"role":        "assistant",
		"content":     []any{},
		"stop_reason": "end_turn",
		"usage": map[string]any{
			"input_tokens":  float64(100),
			"output_tokens": float64(50),
		},
	}

	cfg := &config.DisguiseConfig{
		BodyRewrite:      true,
		IDRewrite:        true,
		SignatureRewrite: true,
		ForceGeo:         true,
		StripBedrock:     true,
	}

	om := FixMessageBody(body, cfg)
	data, _ := json.Marshal(om)
	s := string(data)

	// Check ID was rewritten
	if strings.Contains(s, "msg_bdrk_") {
		t.Errorf("ID should be rewritten, got: %s", s)
	}

	// Check field order
	if !strings.HasPrefix(s, `{"model":`) {
		t.Errorf("should start with model field, got: %s", s[:50])
	}

	// Check usage structure
	if !strings.Contains(s, `"cache_creation"`) {
		t.Errorf("should have nested cache_creation, got: %s", s)
	}

	if !strings.Contains(s, `"inference_geo"`) {
		t.Errorf("should have inference_geo, got: %s", s)
	}
}

func TestFixContentBlock(t *testing.T) {
	tests := []struct {
		name    string
		block   map[string]any
		wantKey string
	}{
		{
			name:    "text block",
			block:   map[string]any{"type": "text", "text": "hello"},
			wantKey: "text",
		},
		{
			name:    "thinking block",
			block:   map[string]any{"type": "thinking", "thinking": "let me think", "signature": ""},
			wantKey: "signature",
		},
		{
			name:    "tool_use block",
			block:   map[string]any{"type": "tool_use", "id": "toolu_bdrk_123", "name": "test", "input": map[string]any{}},
			wantKey: "caller",
		},
	}

	cfg := &config.DisguiseConfig{
		BodyRewrite:      true,
		IDRewrite:        true,
		SignatureRewrite: true,
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			om := FixContentBlock(tc.block, "claude-opus-4-6", cfg)
			data, _ := json.Marshal(om)
			if !strings.Contains(string(data), tc.wantKey) {
				t.Errorf("expected key %s in output: %s", tc.wantKey, string(data))
			}
		})
	}
}

func TestGeoForModel(t *testing.T) {
	tests := []struct {
		model string
		want  string
	}{
		{"claude-opus-4-6", "global"},
		{"claude-sonnet-4-6", "global"},
		{"claude-haiku-4-5", "not_available"},
		{"claude-3-haiku", "not_available"},
	}

	for _, tc := range tests {
		got := GeoForModel(tc.model)
		if got != tc.want {
			t.Errorf("GeoForModel(%s) = %s, want %s", tc.model, got, tc.want)
		}
	}
}
