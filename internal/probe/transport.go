package probe

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (p *Prober) send(targetBase, targetKey string, body []byte) (*http.Response, error) {
	url := strings.TrimRight(targetBase, "/") + "/v1/messages"
	req, err := http.NewRequest("POST", url, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("create: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", targetKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("anthropic-beta", "interleaved-thinking-2025-05-14")

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send: %w", err)
	}
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("upstream %d: %s", resp.StatusCode, truncate(string(b), 200))
	}
	return resp, nil
}

// sendReadJSON sends a non-stream request and returns parsed JSON.
func (p *Prober) sendReadJSON(targetBase, targetKey string, body []byte) (map[string]any, error) {
	_, j, err := p.sendReadRaw(targetBase, targetKey, body)
	return j, err
}

// sendReadRaw sends a non-stream request and returns both raw bytes and parsed JSON.
func (p *Prober) sendReadRaw(targetBase, targetKey string, body []byte) ([]byte, map[string]any, error) {
	resp, err := p.send(targetBase, targetKey, body)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read: %w", err)
	}
	var j map[string]any
	if err := json.Unmarshal(raw, &j); err != nil {
		return raw, nil, fmt.Errorf("parse: %w", err)
	}
	return raw, j, nil
}
