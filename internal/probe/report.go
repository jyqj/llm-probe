package probe

import (
	"fmt"
	"strings"
	"time"

	"bedrock-gateway/internal/config"
)

// CheckResult represents the outcome of a single fingerprint check.
type CheckResult struct {
	Name   string `json:"name"`
	Pass   bool   `json:"pass"`
	Detail string `json:"detail"`
	Fix    string `json:"fix,omitempty"` // DisguiseConfig field to enable if failed
}

// ProbeReport is the full result of a probe suite run.
type ProbeReport struct {
	Target      string               `json:"target"`
	Model       string               `json:"model"`
	Timestamp   time.Time            `json:"timestamp"`
	ElapsedMs   int64                `json:"elapsed_ms"`
	Checks      []CheckResult        `json:"checks"`
	Recommended config.DisguiseConfig `json:"recommended"`
	Summary     string               `json:"summary"`
	Score       *ScoreReport         `json:"score,omitempty"`
}

// RecommendConfig builds a minimal DisguiseConfig that only enables
// the features whose corresponding checks failed.
func RecommendConfig(checks []CheckResult) config.DisguiseConfig {
	cfg := config.DisguiseConfig{Enabled: true}
	for _, c := range checks {
		if c.Pass || c.Fix == "" {
			continue
		}
		switch c.Fix {
		case "IDRewrite":
			cfg.IDRewrite = true
		case "SignatureRewrite":
			cfg.SignatureRewrite = true
		case "BodyRewrite":
			cfg.BodyRewrite = true
		case "HeadersFake":
			cfg.HeadersFake = true
		case "StripDone":
			cfg.StripDone = true
		case "StripContainer":
			cfg.StripContainer = true
		case "StripBedrock":
			cfg.StripBedrock = true
		case "ForceGeo":
			cfg.ForceGeo = true
		case "ThinkingInject":
			cfg.ThinkingInject = true
		case "SmallProbeZero":
			cfg.SmallProbeZero = true
		case "CacheFake":
			cfg.CacheFake = true
		}
	}
	return cfg
}

// BuildSummary generates a human-readable summary from check results.
func BuildSummary(checks []CheckResult) string {
	passed, total := 0, len(checks)
	var fixes []string
	seen := map[string]bool{}
	for _, c := range checks {
		if c.Pass {
			passed++
		} else if c.Fix != "" && !seen[c.Fix] {
			fixes = append(fixes, c.Fix)
			seen[c.Fix] = true
		}
	}
	if len(fixes) == 0 {
		return fmt.Sprintf("%d/%d checks passed. No disguise needed.", passed, total)
	}
	return fmt.Sprintf("%d/%d checks passed. Recommend enabling: %s",
		passed, total, strings.Join(fixes, ", "))
}

// BuildSummaryWithScore generates a summary that includes score and grade.
func BuildSummaryWithScore(checks []CheckResult, score *ScoreReport) string {
	if score == nil {
		return BuildSummary(checks)
	}
	base := BuildSummary(checks)
	return fmt.Sprintf("[%s %.1f] %s — %s", score.Grade, score.TotalScore, score.VerdictLabel, base)
}
