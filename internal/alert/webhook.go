package alert

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// WebhookSender sends alert events via HTTP POST.
type WebhookSender struct {
	dest   WebhookDest
	client *http.Client
}

// NewWebhookSender creates a webhook sender.
func NewWebhookSender(dest WebhookDest) WebhookSender {
	return WebhookSender{
		dest:   dest,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Name returns the webhook destination name.
func (w WebhookSender) Name() string { return w.dest.Name }

// WebhookPayload is the JSON body sent to webhook URLs.
type WebhookPayload struct {
	AlertName  string      `json:"alert_name"`
	Severity   Severity    `json:"severity"`
	Status     EventStatus `json:"status"`
	Message    string      `json:"message"`
	TargetID   string      `json:"target_id"`
	Model      string      `json:"model"`
	Score      float64     `json:"score,omitempty"`
	Grade      string      `json:"grade,omitempty"`
	FiredAt    time.Time   `json:"fired_at"`
	ResolvedAt *time.Time  `json:"resolved_at,omitempty"`
}

// Send posts an alert event to the webhook URL.
func (w WebhookSender) Send(ev *Event) error {
	payload := WebhookPayload{
		AlertName:  ev.RuleName,
		Severity:   ev.Severity,
		Status:     ev.Status,
		Message:    ev.Message,
		TargetID:   ev.TargetID,
		Model:      ev.Model,
		Score:      ev.Score,
		Grade:      ev.Grade,
		FiredAt:    ev.FiredAt,
		ResolvedAt: ev.ResolvedAt,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}

	req, err := http.NewRequest("POST", w.dest.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range w.dest.Headers {
		req.Header.Set(k, v)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned %d", resp.StatusCode)
	}
	return nil
}
