package channeltest

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"detector-service/internal/channeltest/data"
)

// billingBlock returns cctest system[0] — billing header without cache_control.
func billingBlock() map[string]any {
	cch := randomHex(3)
	return map[string]any{
		"type": "text",
		"text": fmt.Sprintf("x-anthropic-billing-header: cc_version=2.1.107.3fe; cc_entrypoint=cli; cch=%s;", cch),
	}
}

// fullSystem returns the cctest 3-block system prompt:
// [0] billing (no cache_control)
// [1] "You are Claude Code..." (cache_control=ephemeral)
// [2] Full instruction text (cache_control=ephemeral) — from embedded data.
func fullSystem() []any {
	return []any{
		billingBlock(),
		map[string]any{
			"type":          "text",
			"text":          "You are Claude Code, Anthropic's official CLI for Claude.",
			"cache_control": map[string]any{"type": "ephemeral"},
		},
		map[string]any{
			"type":          "text",
			"text":          data.SystemPrompt,
			"cache_control": map[string]any{"type": "ephemeral"},
		},
	}
}

// genMetadata returns cctest metadata with random device_id, account_uuid, session_id.
func genMetadata() map[string]any {
	return map[string]any{
		"user_id": fmt.Sprintf(
			`{"device_id":"%s","account_uuid":"%s","session_id":"%s"}`,
			randomHex(32), randomUUID(), randomUUID()),
	}
}

func umsg(content string) map[string]any {
	return map[string]any{"role": "user", "content": content}
}

func toJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func randomUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0F) | 0x40
	b[8] = (b[8] & 0x3F) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
