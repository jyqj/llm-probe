package stream

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// SSEWriter writes Server-Sent Events to an HTTP response.
// Used for the Anthropic Messages API streaming format.
type SSEWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

// NewSSEWriter creates a new SSE writer and sets the appropriate headers.
func NewSSEWriter(w http.ResponseWriter) (*SSEWriter, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	return &SSEWriter{w: w, flusher: flusher}, nil
}

// WriteEvent writes a single SSE event.
func (s *SSEWriter) WriteEvent(eventType string, data any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal event data: %w", err)
	}

	_, err = fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", eventType, jsonData)
	if err != nil {
		return fmt.Errorf("write event: %w", err)
	}

	s.flusher.Flush()
	return nil
}

// WriteRawEvent writes a raw SSE line.
func (s *SSEWriter) WriteRawEvent(eventType string, rawJSON []byte) error {
	_, err := fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", eventType, rawJSON)
	if err != nil {
		return fmt.Errorf("write event: %w", err)
	}
	s.flusher.Flush()
	return nil
}

// WritePing writes a ping event.
func (s *SSEWriter) WritePing() error {
	return s.WriteEvent("ping", map[string]string{"type": "ping"})
}

// Close writes the final data: [DONE] marker if needed.
func (s *SSEWriter) Close() {
	// Anthropic API does not use [DONE], it ends after message_stop event
	s.flusher.Flush()
}
