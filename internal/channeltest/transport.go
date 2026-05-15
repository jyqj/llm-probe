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

// APIError represents a structured error from the upstream API.
type APIError struct {
	StatusCode int    `json:"status_code"`
	ErrorType  string `json:"error_type"`
	Message    string `json:"message"`
	Category   string `json:"category"`
	RawBody    string `json:"raw_body,omitempty"`
}

func (e *APIError) Error() string {
	if e.ErrorType != "" {
		return fmt.Sprintf("HTTP %d (%s): %s", e.StatusCode, e.ErrorType, e.Message)
	}
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

func classifyAPIError(statusCode int, errorType string) string {
	switch {
	case statusCode == 401 || errorType == "authentication_error":
		return "auth"
	case statusCode == 403 || errorType == "permission_error":
		return "forbidden"
	case statusCode == 429 || errorType == "rate_limit_error":
		return "rate_limit"
	case statusCode == 529 || errorType == "overloaded_error":
		return "overloaded"
	case statusCode == 400 || errorType == "invalid_request_error":
		return "client"
	case statusCode >= 500:
		return "server"
	default:
		return "unknown"
	}
}

func parseAPIError(statusCode int, body []byte) *APIError {
	apiErr := &APIError{
		StatusCode: statusCode,
		RawBody:    truncate(string(body), 500),
	}
	var envelope struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &envelope) == nil && envelope.Error.Type != "" {
		apiErr.ErrorType = envelope.Error.Type
		apiErr.Message = envelope.Error.Message
	} else {
		apiErr.Message = truncate(string(body), 200)
	}
	apiErr.Category = classifyAPIError(statusCode, apiErr.ErrorType)
	return apiErr
}

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
			p.recordExchange(body, b, resp.StatusCode, resp.Header)
			return nil, parseAPIError(resp.StatusCode, b)
		}
		if p.recording {
			p.recordExchange(body, nil, 200, resp.Header)
		}
		return resp, nil
	}
	return nil, &APIError{
		StatusCode: 429,
		ErrorType:  "rate_limit_error",
		Message:    "rate limited after retries",
		Category:   "rate_limit",
	}
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
	// Fill response into the exchange that send() already created (avoid duplicates).
	if p.recording {
		p.mu.Lock()
		if n := len(p.exchanges); n > 0 {
			p.exchanges[n-1].Response = json.RawMessage(raw)
			p.exchanges[n-1].Status = resp.StatusCode
		}
		p.mu.Unlock()
	}
	var j map[string]any
	if err := json.Unmarshal(raw, &j); err != nil {
		return raw, nil, fmt.Errorf("parse: %w", err)
	}
	return raw, j, nil
}
