package monitor

import (
	"time"

	"detector-service/internal/channeltest"
	"detector-service/internal/intelligence"
)

// HealthState tracks the current health of a target+model pair.
type HealthState struct {
	TargetID    string    `json:"target_id"`
	Model       string    `json:"model"`
	Status      Status    `json:"status"`
	Score       float64   `json:"score"`
	Grade       string    `json:"grade"`
	LastCheck   time.Time `json:"last_check"`
	LastChange  time.Time `json:"last_change"`
	ConsecFails int       `json:"consec_fails"`
	ConsecOK    int       `json:"consec_ok"`
}

// MonitorRun records a single monitor execution.
type MonitorRun struct {
	ID                  string                  `json:"id"`
	TargetID            string                  `json:"target_id"`
	Model               string                  `json:"model"`
	CheckType           string                  `json:"check_type,omitempty"`
	Status              Status                  `json:"status"`
	Score               float64                 `json:"score"`
	Grade               string                  `json:"grade"`
	Report              *channeltest.Report     `json:"report,omitempty"`
	Error               string                  `json:"error,omitempty"`
	IntelligenceReport  *intelligence.RunReport `json:"intelligence_report,omitempty"`
	IntelligenceError   string                  `json:"intelligence_error,omitempty"`
	BaselineDiff        *DiffReport             `json:"baseline_diff,omitempty"`
	Escalated           bool                    `json:"escalated,omitempty"`
	EscalationReason    string                  `json:"escalation_reason,omitempty"`
	ChannelSurface      string                  `json:"channel_surface,omitempty"`
	IntelligenceSurface string                  `json:"intelligence_surface,omitempty"`
	StartedAt           time.Time               `json:"started_at"`
	ElapsedMs           int64                   `json:"elapsed_ms"`
	PrevState           Status                  `json:"prev_state"`
	Changed             bool                    `json:"changed"`
}

// StatusFromScore derives health status from a channel test score.
func StatusFromScore(score *channeltest.ScoreReport) Status {
	if score == nil {
		return StatusUnknown
	}
	switch {
	case score.TotalScore >= 80:
		return StatusOK
	case score.TotalScore >= 50:
		return StatusWarning
	default:
		return StatusCritical
	}
}

// Transition computes the new HealthState after a monitor run.
func (h *HealthState) Transition(run *MonitorRun) {
	prev := h.Status
	h.Status = run.Status
	h.Score = run.Score
	h.Grade = run.Grade
	h.LastCheck = run.StartedAt

	if prev != run.Status {
		h.LastChange = run.StartedAt
		run.Changed = true
	}
	run.PrevState = prev

	if run.Status == StatusOK {
		h.ConsecFails = 0
		h.ConsecOK++
	} else {
		h.ConsecOK = 0
		h.ConsecFails++
	}
}
