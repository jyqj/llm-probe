package monitor

import (
	"detector-service/internal/channeltest"
	"detector-service/internal/intelligence"
)

// DiffReport holds comparison results between a run and a baseline.
type DiffReport struct {
	BaselineID   string            `json:"baseline_id"`
	BaselineName string            `json:"baseline_name"`
	Channel      *ChannelDiff      `json:"channel,omitempty"`
	Intelligence *IntelligenceDiff `json:"intelligence,omitempty"`
}

// ChannelDiff compares channel check results.
type ChannelDiff struct {
	BaselineScore     float64         `json:"baseline_score"`
	CurrentScore      float64         `json:"current_score"`
	ScoreDelta        float64         `json:"score_delta"`
	OverlappingChecks int             `json:"overlapping_checks"`
	DeviatedChecks    int             `json:"deviated_checks"`
	RegressedChecks   int             `json:"regressed_checks"`
	ImprovedChecks    int             `json:"improved_checks"`
	CheckDiffs        []CheckDiffItem `json:"check_diffs,omitempty"`
}

// CheckDiffItem records one check's baseline vs current result.
type CheckDiffItem struct {
	ProbeID      string `json:"probe_id,omitempty"`
	Name         string `json:"name"`
	BaselinePass bool   `json:"baseline_pass"`
	CurrentPass  bool   `json:"current_pass"`
	Deviated     bool   `json:"deviated"`
}

// IntelligenceDiff compares intelligence task results.
type IntelligenceDiff struct {
	BaselineTaskCount     int            `json:"baseline_task_count"`
	CurrentTaskCount      int            `json:"current_task_count"`
	OverlappingTasks      int            `json:"overlapping_tasks"`
	AggregateDeviation    float64        `json:"aggregate_deviation"`
	AbsAggregateDeviation float64        `json:"abs_aggregate_deviation"`
	TaskDiffs             []TaskDiffItem `json:"task_diffs,omitempty"`
}

// TaskDiffItem records one task's baseline vs current score.
type TaskDiffItem struct {
	TaskID        string  `json:"task_id"`
	BaselineScore float64 `json:"baseline_score"`
	CurrentScore  float64 `json:"current_score"`
	Deviation     float64 `json:"deviation"`
}

// DiffChannel compares a current channel report against a baseline.
func DiffChannel(baseline, current *channeltest.Report) *ChannelDiff {
	if baseline == nil || current == nil {
		return nil
	}
	diff := &ChannelDiff{}
	if baseline.Score != nil {
		diff.BaselineScore = baseline.Score.TotalScore
	}
	if current.Score != nil {
		diff.CurrentScore = current.Score.TotalScore
	}
	diff.ScoreDelta = diff.CurrentScore - diff.BaselineScore

	baseChecks := make(map[string]bool)
	for _, c := range baseline.Checks {
		baseChecks[c.ProbeID+":"+c.Name] = c.Pass
	}
	for _, c := range current.Checks {
		key := c.ProbeID + ":" + c.Name
		bp, exists := baseChecks[key]
		if !exists {
			continue
		}
		deviated := bp != c.Pass
		diff.OverlappingChecks++
		if deviated {
			diff.DeviatedChecks++
		}
		if bp && !c.Pass {
			diff.RegressedChecks++
		}
		if !bp && c.Pass {
			diff.ImprovedChecks++
		}
		diff.CheckDiffs = append(diff.CheckDiffs, CheckDiffItem{
			ProbeID:      c.ProbeID,
			Name:         c.Name,
			BaselinePass: bp,
			CurrentPass:  c.Pass,
			Deviated:     deviated,
		})
	}
	return diff
}

// DiffIntelligence compares a current intelligence report against a baseline.
func DiffIntelligence(baseline, current *intelligence.RunReport) *IntelligenceDiff {
	if baseline == nil || current == nil {
		return nil
	}
	baseScores := make(map[string]float64)
	for _, r := range baseline.Results {
		if r.Score != nil {
			baseScores[r.Task.TaskID] = *r.Score
		}
	}

	diff := &IntelligenceDiff{
		BaselineTaskCount: len(baseline.Results),
		CurrentTaskCount:  len(current.Results),
	}
	var aggDev, absAggDev float64
	for _, r := range current.Results {
		bs, ok := baseScores[r.Task.TaskID]
		if !ok {
			continue
		}
		currentScore := 0.0
		if r.Score != nil {
			currentScore = *r.Score
		} else if r.Error == "" {
			continue
		}
		diff.OverlappingTasks++
		dev := currentScore - bs
		aggDev += dev
		if dev < 0 {
			absAggDev -= dev
		} else {
			absAggDev += dev
		}
		diff.TaskDiffs = append(diff.TaskDiffs, TaskDiffItem{
			TaskID:        r.Task.TaskID,
			BaselineScore: bs,
			CurrentScore:  currentScore,
			Deviation:     dev,
		})
	}
	diff.AggregateDeviation = aggDev
	diff.AbsAggregateDeviation = absAggDev
	return diff
}
