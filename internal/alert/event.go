package alert

import "time"

// EventStatus represents the state of an alert event.
type EventStatus string

const (
	EventFiring   EventStatus = "firing"
	EventResolved EventStatus = "resolved"
)

// Event is a single alert occurrence.
type Event struct {
	ID         string      `json:"id"`
	RuleName   string      `json:"rule_name"`
	Severity   Severity    `json:"severity"`
	Status     EventStatus `json:"status"`
	TargetID   string      `json:"target_id"`
	Target     string      `json:"target"`
	Model      string      `json:"model"`
	CheckType  string      `json:"check_type,omitempty"`
	Message    string      `json:"message"`
	Score      float64     `json:"score,omitempty"`
	Grade      string      `json:"grade,omitempty"`
	FiredAt    time.Time   `json:"fired_at"`
	ResolvedAt *time.Time  `json:"resolved_at,omitempty"`
	Notified   bool        `json:"notified"`
	Silenced   bool        `json:"silenced"`
}
