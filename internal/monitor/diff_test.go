package monitor

import (
	"testing"

	"detector-service/internal/channeltest"
)

func TestDiffChannelCountsRegressionsAndImprovements(t *testing.T) {
	baseline := &channeltest.Report{
		Score: &channeltest.ScoreReport{TotalScore: 90},
		Checks: []channeltest.CheckResult{
			{ProbeID: "p1", Name: "a", Pass: true},
			{ProbeID: "p1", Name: "b", Pass: false},
			{ProbeID: "p2", Name: "c", Pass: true},
		},
	}
	current := &channeltest.Report{
		Score: &channeltest.ScoreReport{TotalScore: 70},
		Checks: []channeltest.CheckResult{
			{ProbeID: "p1", Name: "a", Pass: false},
			{ProbeID: "p1", Name: "b", Pass: true},
			{ProbeID: "p2", Name: "c", Pass: true},
			{ProbeID: "p3", Name: "d", Pass: false},
		},
	}

	diff := DiffChannel(baseline, current)
	if diff == nil {
		t.Fatal("expected diff")
	}
	if diff.OverlappingChecks != 3 {
		t.Fatalf("overlap=%d, want 3", diff.OverlappingChecks)
	}
	if diff.DeviatedChecks != 2 {
		t.Fatalf("deviated=%d, want 2", diff.DeviatedChecks)
	}
	if diff.RegressedChecks != 1 {
		t.Fatalf("regressed=%d, want 1", diff.RegressedChecks)
	}
	if diff.ImprovedChecks != 1 {
		t.Fatalf("improved=%d, want 1", diff.ImprovedChecks)
	}
	if diff.ScoreDelta != -20 {
		t.Fatalf("score delta=%v, want -20", diff.ScoreDelta)
	}
}
