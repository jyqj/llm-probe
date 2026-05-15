package channeltest

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Exchange captures one API request-response pair.
type Exchange struct {
	Request         json.RawMessage `json:"request"`
	Response        json.RawMessage `json:"response,omitempty"`
	Status          int             `json:"status"`
	ResponseHeaders http.Header     `json:"response_headers,omitempty"`
}

// Runner sends test requests to a target API and analyzes responses.
type Runner struct {
	HTTPClient   *http.Client
	KeywordStore *KeywordStore
	recording    bool
	profile      string // set per-clone via withRecorder; never set globally
	ctx          context.Context
	mu           sync.Mutex
	exchanges    []Exchange
}

// NewRunner creates a channel-test runner.
func NewRunner() *Runner {
	return &Runner{
		HTTPClient: &http.Client{Timeout: 180 * time.Second},
	}
}

// withRecorder creates a Runner clone that records all exchanges.
func (p *Runner) withRecorder() *Runner {
	return &Runner{
		HTTPClient:   p.HTTPClient,
		KeywordStore: p.KeywordStore,
		recording:    true,
		profile:      p.profile,
		ctx:          p.ctx,
	}
}

// recordExchange appends a request/response pair if recording is active.
func (p *Runner) recordExchange(req []byte, resp []byte, status int, headers http.Header) {
	if !p.recording {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.exchanges = append(p.exchanges, Exchange{
		Request:         json.RawMessage(req),
		Response:        json.RawMessage(resp),
		Status:          status,
		ResponseHeaders: headers,
	})
}

// getExchanges returns recorded exchanges.
func (p *Runner) getExchanges() []Exchange {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]Exchange, len(p.exchanges))
	copy(out, p.exchanges)
	return out
}
