package alert

import (
	"fmt"
	"time"

	"detector-service/internal/monitor"
)

// Evaluator checks monitor runs against alert rules.
type Evaluator struct {
	rules []Rule
	store *Store
}

// NewEvaluator creates an alert evaluator.
func NewEvaluator(rules []Rule, store *Store) *Evaluator {
	return &Evaluator{rules: rules, store: store}
}

// SetRules replaces the active rule set.
func (e *Evaluator) SetRules(rules []Rule) {
	e.rules = rules
}

// Evaluate checks a monitor run against all rules and returns new events.
func (e *Evaluator) Evaluate(run *monitor.MonitorRun, state *monitor.HealthState) []*Event {
	var events []*Event
	for _, rule := range e.rules {
		ev := e.evaluateRule(rule, run, state)
		if ev != nil {
			events = append(events, ev)
		}
		if run.Status == monitor.StatusOK {
			e.resolveIfNeeded(rule, run)
		}
	}
	return events
}

func (e *Evaluator) evaluateRule(rule Rule, run *monitor.MonitorRun, state *monitor.HealthState) *Event {
	if !e.ruleMatches(rule, run) {
		return nil
	}
	if state != nil && rule.Consecutive > 0 && state.ConsecFails < rule.Consecutive {
		return nil
	}

	dedupKey := fmt.Sprintf("%s:%s:%s", rule.Name, run.TargetID, run.Model)
	if e.store.IsInCooldown(dedupKey, rule.CooldownDuration()) {
		return nil
	}

	ev := &Event{
		ID:       fmt.Sprintf("%d", time.Now().UnixNano()),
		RuleName: rule.Name,
		Severity: rule.Severity,
		Status:   EventFiring,
		TargetID: run.TargetID,
		Target:   run.Model,
		Model:    run.Model,
		Score:    run.Score,
		Grade:    run.Grade,
		FiredAt:  time.Now(),
		Message:  e.buildMessage(rule, run),
	}
	e.store.RecordEvent(ev)
	return ev
}

func (e *Evaluator) ruleMatches(rule Rule, run *monitor.MonitorRun) bool {
	switch rule.Metric {
	case "channel.score":
		return compareFloat(run.Score, rule.Op, rule.Value)
	case "channel.check":
		if run.Report == nil || rule.CheckName == "" {
			return false
		}
		for _, c := range run.Report.Checks {
			if c.Name == rule.CheckName && !c.Pass {
				return true
			}
		}
		return false
	case "channel.status":
		switch rule.Op {
		case "==":
			return string(run.Status) == fmt.Sprintf("%v", rule.Value)
		case "!=":
			return string(run.Status) != fmt.Sprintf("%v", rule.Value)
		}
		return false
	case "intelligence.deviation":
		if run.BaselineDiff == nil || run.BaselineDiff.Intelligence == nil {
			return false
		}
		dev := run.BaselineDiff.Intelligence.AggregateDeviation
		if dev < 0 {
			dev = -dev
		}
		return compareFloat(dev, rule.Op, rule.Value)
	case "intelligence.error_rate":
		if run.IntelligenceReport == nil || run.IntelligenceReport.TaskTotal == 0 {
			return false
		}
		rate := float64(run.IntelligenceReport.TaskErrors) / float64(run.IntelligenceReport.TaskTotal) * 100
		return compareFloat(rate, rule.Op, rule.Value)
	case "intelligence.pass_rate":
		if run.IntelligenceReport == nil || run.IntelligenceReport.PassRate == nil {
			return false
		}
		return compareFloat(*run.IntelligenceReport.PassRate, rule.Op, rule.Value)
	case "intelligence.score_delta":
		if run.BaselineDiff == nil || run.BaselineDiff.Intelligence == nil {
			return false
		}
		return compareFloat(run.BaselineDiff.Intelligence.AbsAggregateDeviation, rule.Op, rule.Value)
	case "intelligence.overlap":
		if run.BaselineDiff == nil || run.BaselineDiff.Intelligence == nil {
			return false
		}
		diff := run.BaselineDiff.Intelligence
		if diff.BaselineTaskCount == 0 {
			return false
		}
		overlapPct := float64(diff.OverlappingTasks) / float64(diff.BaselineTaskCount) * 100
		return compareFloat(overlapPct, rule.Op, rule.Value)
	default:
		return false
	}
}

func (e *Evaluator) resolveIfNeeded(rule Rule, run *monitor.MonitorRun) {
	dedupKey := fmt.Sprintf("%s:%s:%s", rule.Name, run.TargetID, run.Model)
	e.store.ResolveEvent(dedupKey)
}

func (e *Evaluator) buildMessage(rule Rule, run *monitor.MonitorRun) string {
	switch rule.Metric {
	case "channel.score":
		return fmt.Sprintf("[%s] 渠道分数 %.0f %s %.0f (grade=%s, model=%s)",
			rule.Severity, run.Score, rule.Op, rule.Value, run.Grade, run.Model)
	case "channel.check":
		return fmt.Sprintf("[%s] 检查项 %s 失败 (model=%s, score=%.0f)",
			rule.Severity, rule.CheckName, run.Model, run.Score)
	case "intelligence.deviation":
		dev := 0.0
		if run.BaselineDiff != nil && run.BaselineDiff.Intelligence != nil {
			dev = run.BaselineDiff.Intelligence.AggregateDeviation
		}
		return fmt.Sprintf("[%s] 智商偏差 %.1f %s %.1f (model=%s, escalated=%v)",
			rule.Severity, dev, rule.Op, rule.Value, run.Model, run.Escalated)
	case "intelligence.error_rate":
		rate := 0.0
		if run.IntelligenceReport != nil && run.IntelligenceReport.TaskTotal > 0 {
			rate = float64(run.IntelligenceReport.TaskErrors) / float64(run.IntelligenceReport.TaskTotal) * 100
		}
		return fmt.Sprintf("[%s] 智商错误率 %.0f%% %s %.0f%% (model=%s)",
			rule.Severity, rate, rule.Op, rule.Value, run.Model)
	case "intelligence.pass_rate":
		pr := 0.0
		if run.IntelligenceReport != nil && run.IntelligenceReport.PassRate != nil {
			pr = *run.IntelligenceReport.PassRate
		}
		return fmt.Sprintf("[%s] 智商通过率 %.1f%% %s %.1f%% (model=%s)",
			rule.Severity, pr, rule.Op, rule.Value, run.Model)
	case "intelligence.score_delta":
		delta := 0.0
		if run.BaselineDiff != nil && run.BaselineDiff.Intelligence != nil {
			delta = run.BaselineDiff.Intelligence.AbsAggregateDeviation
		}
		return fmt.Sprintf("[%s] 智商分数偏差 %.1f %s %.1f (model=%s)",
			rule.Severity, delta, rule.Op, rule.Value, run.Model)
	case "intelligence.overlap":
		overlap := 0
		if run.BaselineDiff != nil && run.BaselineDiff.Intelligence != nil {
			overlap = run.BaselineDiff.Intelligence.OverlappingTasks
		}
		return fmt.Sprintf("[%s] 智商重叠题目 %d 过低 (model=%s)",
			rule.Severity, overlap, run.Model)
	default:
		return fmt.Sprintf("[%s] %s triggered (model=%s)", rule.Severity, rule.Name, run.Model)
	}
}

func compareFloat(actual float64, op string, threshold float64) bool {
	switch op {
	case "<":
		return actual < threshold
	case "<=":
		return actual <= threshold
	case ">":
		return actual > threshold
	case ">=":
		return actual >= threshold
	case "==":
		return actual == threshold
	case "!=":
		return actual != threshold
	default:
		return false
	}
}
