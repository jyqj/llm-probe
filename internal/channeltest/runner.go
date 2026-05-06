package channeltest

import (
	"net/http"
	"time"
)

// Runner sends test requests to a target API and analyzes responses.
type Runner struct {
	HTTPClient *http.Client
}

// NewRunner creates a channel-test runner.
func NewRunner() *Runner {
	return &Runner{
		HTTPClient: &http.Client{Timeout: 180 * time.Second},
	}
}
