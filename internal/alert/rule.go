package alert

import "time"

// Severity levels for alert rules.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

// Rule defines when an alert should fire.
type Rule struct {
	Name        string   `json:"name" yaml:"name"`
	Metric      string   `json:"metric" yaml:"metric"`
	Op          string   `json:"op" yaml:"op"`
	Value       float64  `json:"value" yaml:"value"`
	CheckName   string   `json:"check,omitempty" yaml:"check,omitempty"`
	Consecutive int      `json:"consecutive" yaml:"consecutive"`
	Severity    Severity `json:"severity" yaml:"severity"`
	Cooldown    string   `json:"cooldown,omitempty" yaml:"cooldown,omitempty"`
}

// CooldownDuration parses the cooldown string into a duration.
func (r *Rule) CooldownDuration() time.Duration {
	if r.Cooldown == "" {
		return 30 * time.Minute
	}
	d, err := time.ParseDuration(r.Cooldown)
	if err != nil {
		return 30 * time.Minute
	}
	return d
}

// AlertConfig holds global alert settings.
type AlertConfig struct {
	Enabled  bool          `json:"enabled" yaml:"enabled"`
	Cooldown string        `json:"cooldown,omitempty" yaml:"cooldown,omitempty"`
	Rules    []Rule        `json:"rules" yaml:"rules"`
	Webhooks []WebhookDest `json:"webhooks,omitempty" yaml:"webhooks,omitempty"`
}

// WebhookDest is a notification target.
type WebhookDest struct {
	Name    string            `json:"name" yaml:"name"`
	URL     string            `json:"url" yaml:"url"`
	Headers map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
}

// DefaultRules returns a sensible set of default alert rules.
func DefaultRules() []Rule {
	return []Rule{
		{
			Name:        "channel_score_low",
			Metric:      "channel.score",
			Op:          "<",
			Value:       80,
			Consecutive: 2,
			Severity:    SeverityWarning,
			Cooldown:    "30m",
		},
		{
			Name:        "channel_score_critical",
			Metric:      "channel.score",
			Op:          "<",
			Value:       50,
			Consecutive: 1,
			Severity:    SeverityCritical,
			Cooldown:    "15m",
		},
		{
			Name:        "channel_backend_type_fail",
			Metric:      "channel.check",
			CheckName:   "backend_type",
			Op:          "==",
			Value:       0,
			Consecutive: 1,
			Severity:    SeverityCritical,
			Cooldown:    "15m",
		},
		{
			Name:        "intelligence_deviation_high",
			Metric:      "intelligence.deviation",
			Op:          ">",
			Value:       4,
			Consecutive: 1,
			Severity:    SeverityWarning,
			Cooldown:    "30m",
		},
		{
			Name:        "intelligence_error_rate_high",
			Metric:      "intelligence.error_rate",
			Op:          ">",
			Value:       50,
			Consecutive: 1,
			Severity:    SeverityCritical,
			Cooldown:    "15m",
		},
		{
			Name:        "intelligence_score_delta_high",
			Metric:      "intelligence.score_delta",
			Op:          ">",
			Value:       20,
			Consecutive: 1,
			Severity:    SeverityWarning,
			Cooldown:    "30m",
		},
		{
			Name:        "intelligence_overlap_too_low",
			Metric:      "intelligence.overlap",
			Op:          "<",
			Value:       50,
			Consecutive: 1,
			Severity:    SeverityInfo,
			Cooldown:    "1h",
		},
	}
}
