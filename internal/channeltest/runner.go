package channeltest

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Exchange captures one API request-response pair.
type Exchange struct {
	Request  json.RawMessage `json:"request"`
	Response json.RawMessage `json:"response,omitempty"`
	Status   int             `json:"status"`
}

// Runner sends test requests to a target API and analyzes responses.
type Runner struct {
	HTTPClient *http.Client
	recording  bool
	mu         sync.Mutex
	exchanges  []Exchange
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
		HTTPClient: p.HTTPClient,
		recording:  true,
	}
}

// recordExchange appends a request/response pair if recording is active.
func (p *Runner) recordExchange(req []byte, resp []byte, status int) {
	if !p.recording {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.exchanges = append(p.exchanges, Exchange{
		Request:  json.RawMessage(req),
		Response: json.RawMessage(resp),
		Status:   status,
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
