package channeltest

import (
	"fmt"
	"strings"
	"time"
)

// CheckResult represents the outcome of a single fingerprint check.
type CheckResult struct {
	ProbeID  string `json:"probe_id,omitempty"`
	Name     string `json:"name"`
	Label    string `json:"label,omitempty"`
	Pass     bool   `json:"pass"`
	Expected string `json:"expected,omitempty"`
	Actual   string `json:"actual,omitempty"`
	Detail   string `json:"detail"`
	Fix      Fix    `json:"fix,omitempty"`
}

// ProbeError records an API error encountered by a specific probe.
type ProbeError struct {
	ProbeID    string `json:"probe_id"`
	StatusCode int    `json:"status_code"`
	ErrorType  string `json:"error_type"`
	Message    string `json:"message"`
	Category   string `json:"category"`
}

// CheckComparison records one check's baseline vs current result.
type CheckComparison struct {
	ProbeID        string `json:"probe_id,omitempty"`
	Name           string `json:"name"`
	Label          string `json:"label,omitempty"`
	Category       string `json:"category,omitempty"`
	BaselinePass   bool   `json:"baseline_pass"`
	CurrentPass    bool   `json:"current_pass"`
	Deviated       bool   `json:"deviated"`
	BaselineDetail string `json:"baseline_detail,omitempty"`
	CurrentDetail  string `json:"current_detail,omitempty"`
}

// ChannelBaselineComparison holds the result of comparing a channel run against a baseline.
type ChannelBaselineComparison struct {
	BaselineID        string            `json:"baseline_id"`
	BaselineName      string            `json:"baseline_name,omitempty"`
	BaselineProfile   string            `json:"baseline_profile,omitempty"`
	BaselineScore     float64           `json:"baseline_score"`
	CurrentScore      float64           `json:"current_score"`
	RelativeScore     float64           `json:"relative_score"`
	ScoreDelta        float64           `json:"score_delta"`
	OverlappingChecks int               `json:"overlapping_checks"`
	DeviatedChecks    int               `json:"deviated_checks"`
	CheckComparisons  []CheckComparison `json:"check_comparisons,omitempty"`
}

// Report is the full result of a channel test suite run.
type Report struct {
	ID            string           `json:"id"`
	RunGroup      string           `json:"run_group,omitempty"`
	ChannelName   string           `json:"channel_name,omitempty"`
	Target        string           `json:"target"`
	Model         string           `json:"model"`
	Timestamp     time.Time        `json:"timestamp"`
	ElapsedMs     int64            `json:"elapsed_ms"`
	Checks        []CheckResult    `json:"checks"`
	ProbeResults  []ProbeResult    `json:"probe_results,omitempty"`
	Billing       *BillingEstimate `json:"billing,omitempty"`
	Recommended   Recommendation   `json:"recommended"`
	Summary       string           `json:"summary"`
	Score         *ScoreReport     `json:"score,omitempty"`
	SkippedProbes []string         `json:"skipped_probes,omitempty"`
	RunProfile    string           `json:"run_profile,omitempty"`
	Profile              string           `json:"profile,omitempty"`
	Errors               []ProbeError     `json:"errors,omitempty"`
	IdentifiedChannels   []ChannelHit     `json:"identified_channels,omitempty"`
	BaselineComparison   *ChannelBaselineComparison `json:"baseline_comparison,omitempty"`
}

// CompareToBaseline computes a check-level comparison between this report and a baseline.
func (r *Report) CompareToBaseline(baselineID, baselineName string, baseline *Report) {
	if baseline == nil || r == nil {
		return
	}
	comp := &ChannelBaselineComparison{
		BaselineID:   baselineID,
		BaselineName: baselineName,
	}
	if baseline.Profile != "" {
		comp.BaselineProfile = baseline.Profile
	}
	if baseline.Score != nil {
		comp.BaselineScore = baseline.Score.TotalScore
	}
	if r.Score != nil {
		comp.CurrentScore = r.Score.TotalScore
	}
	comp.ScoreDelta = comp.CurrentScore - comp.BaselineScore
	if comp.BaselineScore > 0 {
		comp.RelativeScore = round2(comp.CurrentScore / comp.BaselineScore * 100)
	} else if comp.CurrentScore > 0 {
		comp.RelativeScore = 100
	}

	baseChecks := make(map[string]CheckResult)
	for _, c := range baseline.Checks {
		baseChecks[c.ProbeID+":"+c.Name] = c
	}
	for _, c := range r.Checks {
		key := c.ProbeID + ":" + c.Name
		bc, ok := baseChecks[key]
		if !ok {
			continue
		}
		comp.OverlappingChecks++
		deviated := bc.Pass != c.Pass
		if deviated {
			comp.DeviatedChecks++
		}
		cat := ""
		if catKey, exists := checkCategoryMap[c.Name]; exists {
			cat = string(catKey)
		}
		comp.CheckComparisons = append(comp.CheckComparisons, CheckComparison{
			ProbeID:        c.ProbeID,
			Name:           c.Name,
			Label:          c.Label,
			Category:       cat,
			BaselinePass:   bc.Pass,
			CurrentPass:    c.Pass,
			Deviated:       deviated,
			BaselineDetail: bc.Detail,
			CurrentDetail:  c.Detail,
		})
	}

	r.BaselineComparison = comp
}

// InjectLabels fills in the Label field from the check registry for all results.
func InjectLabels(checks []CheckResult) {
	for i := range checks {
		if checks[i].Label == "" {
			checks[i].Label = labelForCheck(checks[i].Name)
		}
	}
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
