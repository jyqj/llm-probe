package channeltest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (p *Runner) send(targetBase, targetKey string, body []byte) (*http.Response, error) {
	url := strings.TrimRight(targetBase, "/") + "/v1/messages"
	ctx := p.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	for attempt := 0; attempt < 3; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
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
		if resp.StatusCode == 429 && attempt < 2 {
			retryAfter := 15 * time.Second
			if s := resp.Header.Get("Retry-After"); s != "" {
				if n, err := strconv.Atoi(s); err == nil {
					retryAfter = time.Duration(n) * time.Second
				}
			}
			resp.Body.Close()
			select {
			case <-time.After(retryAfter):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			continue
		}
		if resp.StatusCode != 200 {
			b, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			p.recordExchange(body, b, resp.StatusCode)
			return nil, fmt.Errorf("upstream %d: %s", resp.StatusCode, truncate(string(b), 200))
		}
		if p.recording {
			p.recordExchange(body, nil, 200)
		}
		return resp, nil
	}
	return nil, fmt.Errorf("upstream 429: rate limited after retries")
}

// recordStreamResult updates the last exchange's response with the merged SSE result.
func (p *Runner) recordStreamResult(merged map[string]any) {
	if !p.recording || merged == nil {
		return
	}
	raw, _ := json.Marshal(merged)
	p.mu.Lock()
	defer p.mu.Unlock()
	if n := len(p.exchanges); n > 0 {
		p.exchanges[n-1].Response = json.RawMessage(raw)
	}
}

func (p *Runner) sendReadJSON(targetBase, targetKey string, body []byte) (map[string]any, error) {
	_, j, err := p.sendReadRaw(targetBase, targetKey, body)
	return j, err
}

func (p *Runner) sendReadRaw(targetBase, targetKey string, body []byte) ([]byte, map[string]any, error) {
	resp, err := p.send(targetBase, targetKey, body)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read: %w", err)
	}
	p.recordExchange(body, raw, resp.StatusCode)
	var j map[string]any
	if err := json.Unmarshal(raw, &j); err != nil {
		return raw, nil, fmt.Errorf("parse: %w", err)
	}
	return raw, j, nil
}
