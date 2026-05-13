package monitor

import (
	"context"
	"log/slog"
	"math"
	"time"

	"detector-service/internal/channeltest"
	"detector-service/internal/intelligence"
)

// RunCallback is called after each monitor run with the result.
type RunCallback func(run *MonitorRun)

// MonitorRunner executes monitor checks against targets.
type MonitorRunner struct {
	channelRunner      *channeltest.Runner
	intelligenceRunner *intelligence.Runner
	registry           *intelligence.Registry
	baselineStore      *BaselineStore
	store              *Store
	logger             *slog.Logger
	onRun              RunCallback
}

// NewRunner creates a monitor runner.
func NewRunner(
	channelRunner *channeltest.Runner,
	intelligenceRunner *intelligence.Runner,
	registry *intelligence.Registry,
	baselineStore *BaselineStore,
	store *Store,
	logger *slog.Logger,
	onRun RunCallback,
) *MonitorRunner {
	return &MonitorRunner{
		channelRunner:      channelRunner,
		intelligenceRunner: intelligenceRunner,
		registry:           registry,
		baselineStore:      baselineStore,
		store:              store,
		logger:             logger,
		onRun:              onRun,
	}
}

// RunTarget executes a monitor check for one target+model pair.
func (r *MonitorRunner) RunTarget(target *Target, model string) *MonitorRun {
	start := time.Now()
	run := &MonitorRun{
		TargetID:  target.ID,
		Model:     model,
		CheckType: target.CheckType,
		StartedAt: start,
	}

	checkType := target.CheckType
	if checkType == "" {
		checkType = "channel"
	}

	switch checkType {
	case "channel":
		r.runChannel(target, model, run)
	case "intelligence":
		r.runIntelligence(target, model, run)
	case "both":
		r.runChannel(target, model, run)
		r.runIntelligence(target, model, run)
	}

	run.ElapsedMs = time.Since(start).Milliseconds()
	run.Status = r.deriveStatus(run)

	if target.BaselineID != "" && r.baselineStore != nil {
		if baseline := r.baselineStore.Get(target.BaselineID); baseline != nil {
			run.BaselineDiff = r.computeDiff(baseline, run)
		}
	}

	r.store.RecordRun(run)
	r.logger.Info("monitor run completed",
		"target", target.Name, "model", model,
		"check_type", checkType, "status", run.Status,
		"score", run.Score, "grade", run.Grade,
		"escalated", run.Escalated,
		"elapsed_ms", run.ElapsedMs)

	if r.onRun != nil {
		r.onRun(run)
	}
	return run
}

// RunAll executes monitor checks for all models of a target.
func (r *MonitorRunner) RunAll(target *Target) []*MonitorRun {
	var runs []*MonitorRun
	for _, model := range target.Models {
		runs = append(runs, r.RunTarget(target, model))
	}
	return runs
}

func (r *MonitorRunner) runChannel(target *Target, model string, run *MonitorRun) {
	report, err := r.channelRunner.RunMonitor(target.BaseURL, target.APIKey, model)
	if err != nil {
		run.Error = err.Error()
		r.logger.Warn("monitor channel run failed",
			"target", target.Name, "model", model, "error", err)
		return
	}
	run.Report = report
	if report.Score != nil {
		run.Score = report.Score.TotalScore
		run.Grade = report.Score.Grade
	}
}

func (r *MonitorRunner) runIntelligence(target *Target, model string, run *MonitorRun) {
	if r.intelligenceRunner == nil || r.registry == nil {
		run.IntelligenceError = "intelligence runner not configured"
		return
	}

	dsName := target.IntelligenceDataset
	if dsName == "" {
		names := r.registry.List()
		if len(names) == 0 {
			run.IntelligenceError = "no datasets available"
			return
		}
		dsName = names[0]
	}
	ds, ok := r.registry.Get(dsName)
	if !ok {
		run.IntelligenceError = "dataset not found: " + dsName
		return
	}

	limit := target.IntelligenceLimit
	if limit <= 0 {
		limit = 3
	}
	maxLimit := target.IntelligenceMaxLimit
	if maxLimit <= 0 {
		maxLimit = 10
	}
	threshold := target.IntelligenceThreshold
	if threshold <= 0 {
		threshold = 1.0
	}

	ctx := context.Background()
	report, err := r.intelligenceRunner.Run(ctx, ds, intelligence.RunRequest{
		TargetBase:   target.BaseURL,
		TargetKey:    target.APIKey,
		Model:        model,
		Limit:        limit,
		Effort:       target.Effort,
		ThinkingMode: target.ThinkingMode,
		MaxTokens:    target.MaxTokens,
	})
	if err != nil {
		run.IntelligenceError = err.Error()
		return
	}

	needsEscalation := false
	if target.BaselineID != "" && r.baselineStore != nil {
		if baseline := r.baselineStore.Get(target.BaselineID); baseline != nil && baseline.IntelligenceReport != nil {
			diff := DiffIntelligence(baseline.IntelligenceReport, report)
			if diff != nil {
				for _, td := range diff.TaskDiffs {
					if math.Abs(td.Deviation) > threshold {
						needsEscalation = true
						break
					}
				}
			}
		}
	}

	if needsEscalation && maxLimit > limit {
		run.Escalated = true
		r.logger.Info("intelligence escalation triggered",
			"target", target.Name, "model", model,
			"initial", limit, "escalating_to", maxLimit)
		escalated, err := r.intelligenceRunner.Run(ctx, ds, intelligence.RunRequest{
			TargetBase:   target.BaseURL,
			TargetKey:    target.APIKey,
			Model:        model,
			Limit:        maxLimit,
			Effort:       target.Effort,
			ThinkingMode: target.ThinkingMode,
			MaxTokens:    target.MaxTokens,
		})
		if err == nil {
			report = escalated
		}
	}

	run.IntelligenceReport = report
}

func (r *MonitorRunner) deriveStatus(run *MonitorRun) Status {
	if run.Error != "" && run.IntelligenceError != "" {
		return StatusCritical
	}

	channelStatus := StatusUnknown
	if run.Report != nil {
		channelStatus = StatusFromScore(run.Report.Score)
	} else if run.Error != "" {
		channelStatus = StatusCritical
	}

	intelligenceStatus := StatusUnknown
	if run.IntelligenceReport != nil {
		intelligenceStatus = r.intelligenceStatus(run)
	} else if run.IntelligenceError != "" {
		intelligenceStatus = StatusCritical
	}

	checkType := run.CheckType
	if checkType == "" {
		checkType = "channel"
	}

	switch checkType {
	case "channel":
		return channelStatus
	case "intelligence":
		return intelligenceStatus
	default:
		order := map[Status]int{StatusOK: 0, StatusUnknown: 1, StatusWarning: 2, StatusCritical: 3}
		if order[channelStatus] > order[intelligenceStatus] {
			return channelStatus
		}
		return intelligenceStatus
	}
}

func (r *MonitorRunner) intelligenceStatus(run *MonitorRun) Status {
	report := run.IntelligenceReport
	if report == nil {
		return StatusUnknown
	}
	if report.TaskErrors > 0 && report.TaskErrors >= report.TaskTotal/2 {
		return StatusCritical
	}
	if run.BaselineDiff != nil && run.BaselineDiff.Intelligence != nil {
		absDev := math.Abs(run.BaselineDiff.Intelligence.AggregateDeviation)
		if absDev >= 4.0 {
			return StatusWarning
		}
	}
	return StatusOK
}

func (r *MonitorRunner) computeDiff(baseline *Baseline, run *MonitorRun) *DiffReport {
	diff := &DiffReport{
		BaselineID:   baseline.ID,
		BaselineName: baseline.Name,
	}
	diff.Channel = DiffChannel(baseline.ChannelReport, run.Report)
	diff.Intelligence = DiffIntelligence(baseline.IntelligenceReport, run.IntelligenceReport)
	return diff
}
