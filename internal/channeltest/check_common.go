package channeltest

import "strings"

// collectResponseText extracts all text content from a response body.
func collectResponseText(body map[string]any) string {
	content, _ := body["content"].([]any)
	var parts []string
	for _, cb := range content {
		m, ok := cb.(map[string]any)
		if !ok {
			continue
		}
		t, _ := m["type"].(string)
		if t == "text" {
			if txt, _ := m["text"].(string); txt != "" {
				parts = append(parts, txt)
			}
		}
	}
	return strings.Join(parts, "\n")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
