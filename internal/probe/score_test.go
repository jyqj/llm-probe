package probe

import (
	"testing"
)

func TestCalculateScore(t *testing.T) {
	tests := []struct {
		name        string
		checks      []CheckResult
		mode        string
		wantGradeGT string // grade should be at least this
		wantVerdict string
	}{
		{
			name: "all pass",
			checks: []CheckResult{
				{Name: "id_format", Pass: true},
				{Name: "backend_type", Pass: true},
				{Name: "inference_geo", Pass: true},
				{Name: "usage_structure", Pass: true},
				{Name: "headers", Pass: true},
				{Name: "signature", Pass: true},
				{Name: "sse_done", Pass: true},
			},
			mode:        "quick",
			wantGradeGT: "A",
			wantVerdict: "official",
		},
		{
			name: "backend type fail",
			checks: []CheckResult{
				{Name: "id_format", Pass: false},
				{Name: "backend_type", Pass: false, Detail: "Bedrock backend (msg_bdrk_)"},
				{Name: "inference_geo", Pass: true},
			},
			mode:        "quick",
			wantVerdict: "non_official",
		},
		{
			name: "mixed results",
			checks: []CheckResult{
				{Name: "id_format", Pass: true},
				{Name: "backend_type", Pass: true},
				{Name: "headers", Pass: false},
				{Name: "sse_done", Pass: false},
				{Name: "cache_fake", Pass: false},
			},
			mode:        "full",
			wantGradeGT: "D", // only 2/5 critical checks pass, grade is lower
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := CalculateScore(tc.checks, tc.mode)

			if tc.wantVerdict != "" && report.Verdict != tc.wantVerdict {
				t.Errorf("verdict = %s, want %s", report.Verdict, tc.wantVerdict)
			}

			if tc.wantGradeGT != "" {
				gradeOrder := map[string]int{"F": 0, "D": 1, "C": 2, "B": 3, "A": 4, "A+": 5}
				if gradeOrder[report.Grade] < gradeOrder[tc.wantGradeGT] {
					t.Errorf("grade = %s, want >= %s", report.Grade, tc.wantGradeGT)
				}
			}

			if report.ChecksTotal != len(tc.checks) {
				t.Errorf("checks_total = %d, want %d", report.ChecksTotal, len(tc.checks))
			}
		})
	}
}

func TestMapGrade(t *testing.T) {
	tests := []struct {
		score float64
		want  string
	}{
		{100, "A+"},
		{95, "A+"},
		{90, "A"},
		{85, "B"},
		{75, "C"},
		{60, "D"},
		{40, "F"},
	}

	for _, tc := range tests {
		grade, _ := mapGrade(tc.score)
		if grade != tc.want {
			t.Errorf("mapGrade(%v) = %s, want %s", tc.score, grade, tc.want)
		}
	}
}

func TestCategoryMapping(t *testing.T) {
	// Verify all important checks are categorized
	important := []string{
		"id_format", "backend_type", "headers", "signature",
		"sse_done", "usage_structure", "thinking_present",
	}

	for _, name := range important {
		if _, ok := checkCategoryMap[name]; !ok {
			t.Errorf("check %q is not categorized", name)
		}
	}
}
