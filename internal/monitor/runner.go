package monitor

import (
	"context"
	"log/slog"
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
	return r.RunTargetCtx(context.Background(), target, model)
}

// RunTargetCtx executes a monitor check with context for cancellation/timeout.
func (r *MonitorRunner) RunTargetCtx(ctx context.Context, target *Target, model string) *MonitorRun {
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
		r.runChannel(ctx, target, model, run)
	case "intelligence":
		r.runIntelligence(ctx, target, model, run)
	case "both":
		r.runChannel(ctx, target, model, run)
		if ctx.Err() == nil {
			r.runIntelligence(ctx, target, model, run)
		}
	}

	run.ElapsedMs = time.Since(start).Milliseconds()
	if target.BaselineID != "" && r.baselineStore != nil {
		if baseline := r.baselineStore.Get(target.BaselineID); baseline != nil {
			run.BaselineDiff = r.computeDiff(baseline, run)
		}
	}
	run.Status = r.deriveStatus(run)

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
	return r.RunAllCtx(context.Background(), target)
}

// RunAllCtx executes monitor checks for all models with context.
func (r *MonitorRunner) RunAllCtx(ctx context.Context, target *Target) []*MonitorRun {
	var runs []*MonitorRun
	for _, model := range target.Models {
		if ctx.Err() != nil {
			break
		}
		runs = append(runs, r.RunTargetCtx(ctx, target, model))
	}
	return runs
}

func (r *MonitorRunner) runChannel(ctx context.Context, target *Target, model string, run *MonitorRun) {
	report, err := r.channelRunner.RunMonitor(target.BaseURL, target.APIKey, model)
	if err != nil {
		run.Error = err.Error()
		r.logger.Warn("monitor channel run failed",
			"target", target.Name, "model", model, "error", err)
		return
	}
	run.ChannelSurface = "monitor"
	run.Report = report
	if report.Score != nil {
		run.Score = report.Score.TotalScore
		run.Grade = report.Score.Grade
	}
	if !channelReportOK(report) {
		full, err := r.channelRunner.Run(target.BaseURL, target.APIKey, model, 2)
		if err != nil {
			run.Error = err.Error()
			r.logger.Warn("monitor full channel escalation failed",
				"target", target.Name, "model", model, "error", err)
			return
		}
		run.Escalated = true
		run.EscalationReason = "channel monitor surface reported non-ok"
		run.ChannelSurface = "full"
		run.Report = full
		if full.Score != nil {
			run.Score = full.Score.TotalScore
			run.Grade = full.Score.Grade
		}
	}
}

func (r *MonitorRunner) runIntelligence(ctx context.Context, target *Target, model string, run *MonitorRun) {
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
	threshold := target.IntelligenceThreshold
	if threshold <= 0 {
		threshold = 1.0
	}

	report, err := r.intelligenceRunner.Run(ctx, ds, intelligence.RunRequest{
		TargetBase:     target.BaseURL,
		TargetKey:      target.APIKey,
		Model:          model,
		Limit:          limit,
		ImportantFirst: true,
		Effort:         target.Effort,
		ThinkingMode:   target.ThinkingMode,
		MaxTokens:      target.MaxTokens,
	})
	if err != nil {
		run.IntelligenceError = err.Error()
		return
	}
	run.IntelligenceSurface = "important"

	needsEscalation := false
	reason := ""
	if target.BaselineID != "" && r.baselineStore != nil {
		if baseline := r.baselineStore.Get(target.BaselineID); baseline != nil && baseline.IntelligenceReport != nil {
			diff := DiffIntelligence(baseline.IntelligenceReport, report)
			if diff != nil {
				for _, td := range diff.TaskDiffs {
					if td.Deviation < -threshold {
						needsEscalation = true
						reason = "intelligence score below official baseline threshold"
						break
					}
				}
			}
		}
	}

	if needsEscalation && (maxLimit == 0 || maxLimit > limit) {
		run.Escalated = true
		run.EscalationReason = reason
		r.logger.Info("intelligence escalation triggered",
			"target", target.Name, "model", model,
			"initial", limit, "escalating_to", maxLimit)
		escalated, err := r.intelligenceRunner.Run(ctx, ds, intelligence.RunRequest{
			TargetBase:     target.BaseURL,
			TargetKey:      target.APIKey,
			Model:          model,
			Limit:          maxLimit,
			ImportantFirst: true,
			Effort:         target.Effort,
			ThinkingMode:   target.ThinkingMode,
			MaxTokens:      target.MaxTokens,
		})
		if err == nil {
			report = escalated
			if maxLimit == 0 {
				run.IntelligenceSurface = "full"
			} else {
				run.IntelligenceSurface = "expanded"
			}
		}
	}

	run.IntelligenceReport = report
}

func channelReportOK(report *channeltest.Report) bool {
	if report == nil || report.Score == nil {
		return false
	}
	return StatusFromScore(report.Score) == StatusOK
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
	if run.BaselineDiff != nil && run.BaselineDiff.Intelligence != nil {
		diff := run.BaselineDiff.Intelligence
		if diff.OverlappingTasks == 0 {
			return StatusUnknown
		}
		avgDrop := -diff.AggregateDeviation / float64(diff.OverlappingTasks)
		if avgDrop >= 1.0 {
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
