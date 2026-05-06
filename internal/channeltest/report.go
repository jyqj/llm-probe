package channeltest

import (
	"fmt"
	"strings"
	"time"
)

// CheckResult represents the outcome of a single fingerprint check.
type CheckResult struct {
	Name   string          `json:"name"`
	Pass   bool            `json:"pass"`
	Detail string          `json:"detail"`
	Fix    Fix `json:"fix,omitempty"` // neutral remediation item to enable if failed
}

// Report is the full result of a channel test suite run.
type Report struct {
	ID          string                     `json:"id"`
	Target      string                     `json:"target"`
	Model       string                     `json:"model"`
	Timestamp   time.Time                  `json:"timestamp"`
	ElapsedMs   int64                      `json:"elapsed_ms"`
	Checks      []CheckResult              `json:"checks"`
	Recommended Recommendation `json:"recommended"`
	Summary     string                     `json:"summary"`
	Score       *ScoreReport               `json:"score,omitempty"`
}

// RecommendFixes builds a minimal neutral recommendation from failed checks.
func RecommendFixes(checks []CheckResult) Recommendation {
	rec := Recommendation{Enabled: true}
	seen := map[Fix]bool{}
	for _, c := range checks {
		if c.Pass {
			continue
		}
		fix := c.Fix
		if fix == "" {
			fix = defaultFixForCheck(c.Name)
		}
		if fix == "" || seen[fix] {
			continue
		}
		rec.Fixes = append(rec.Fixes, fix)
		seen[fix] = true
	}
	return rec
}

// BuildSummary generates a human-readable summary from check results.
func BuildSummary(checks []CheckResult) string {
	passed, total := 0, len(checks)
	var fixes []string
	seen := map[Fix]bool{}
	for _, c := range checks {
		if c.Pass {
			passed++
			continue
		}
		fix := c.Fix
		if fix == "" {
			fix = defaultFixForCheck(c.Name)
		}
		if fix != "" && !seen[fix] {
			fixes = append(fixes, string(fix))
			seen[fix] = true
		}
	}
	if len(fixes) == 0 {
		return fmt.Sprintf("%d/%d checks passed. No issues detected.", passed, total)
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
