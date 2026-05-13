package monitor

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// Status represents the health state of a monitored target.
type Status string

const (
	StatusUnknown  Status = "unknown"
	StatusOK       Status = "ok"
	StatusWarning  Status = "warning"
	StatusCritical Status = "critical"
)

// Target is a monitored API endpoint.
type Target struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	BaseURL   string        `json:"base_url"`
	APIKey    string        `json:"-"`
	Models    []string      `json:"models"`
	Interval  time.Duration `json:"interval"`
	Jitter    time.Duration `json:"jitter,omitempty"`
	Enabled   bool          `json:"enabled"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`

	CheckType             string  `json:"check_type"`
	IntelligenceDataset   string  `json:"intelligence_dataset,omitempty"`
	IntelligenceLimit     int     `json:"intelligence_limit,omitempty"`
	IntelligenceMaxLimit  int     `json:"intelligence_max_limit,omitempty"`
	IntelligenceThreshold float64 `json:"intelligence_threshold,omitempty"`
	BaselineID            string  `json:"baseline_id,omitempty"`
	Effort                string  `json:"effort,omitempty"`
	ThinkingMode          string  `json:"thinking_mode,omitempty"`
	MaxTokens             int     `json:"max_tokens,omitempty"`
}

func validCheckType(ct string) bool {
	return ct == "" || ct == "channel" || ct == "intelligence" || ct == "both"
}

// TargetCreateRequest is the API input for creating a target.
type TargetCreateRequest struct {
	Name                  string   `json:"name"`
	BaseURL               string   `json:"base_url"`
	APIKey                string   `json:"api_key"`
	Models                []string `json:"models"`
	Interval              string   `json:"interval"`
	Jitter                string   `json:"jitter,omitempty"`
	Enabled               *bool    `json:"enabled,omitempty"`
	CheckType             string   `json:"check_type,omitempty"`
	IntelligenceDataset   string   `json:"intelligence_dataset,omitempty"`
	IntelligenceLimit     *int     `json:"intelligence_limit,omitempty"`
	IntelligenceMaxLimit  *int     `json:"intelligence_max_limit,omitempty"`
	IntelligenceThreshold *float64 `json:"intelligence_threshold,omitempty"`
	BaselineID            string   `json:"baseline_id,omitempty"`
	Effort                string   `json:"effort,omitempty"`
	ThinkingMode          string   `json:"thinking_mode,omitempty"`
	MaxTokens             *int     `json:"max_tokens,omitempty"`
}

// TargetUpdateRequest is the API input for updating a target.
type TargetUpdateRequest struct {
	Name                  *string  `json:"name,omitempty"`
	BaseURL               *string  `json:"base_url,omitempty"`
	APIKey                *string  `json:"api_key,omitempty"`
	Models                []string `json:"models,omitempty"`
	Interval              *string  `json:"interval,omitempty"`
	Jitter                *string  `json:"jitter,omitempty"`
	Enabled               *bool    `json:"enabled,omitempty"`
	CheckType             *string  `json:"check_type,omitempty"`
	IntelligenceDataset   *string  `json:"intelligence_dataset,omitempty"`
	IntelligenceLimit     *int     `json:"intelligence_limit,omitempty"`
	IntelligenceMaxLimit  *int     `json:"intelligence_max_limit,omitempty"`
	IntelligenceThreshold *float64 `json:"intelligence_threshold,omitempty"`
	BaselineID            *string  `json:"baseline_id,omitempty"`
	Effort                *string  `json:"effort,omitempty"`
	ThinkingMode          *string  `json:"thinking_mode,omitempty"`
	MaxTokens             *int     `json:"max_tokens,omitempty"`
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
