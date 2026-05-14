package channeltest

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ReportSink receives completed channel test reports for external consumption.
type ReportSink func(upstreamBase string, report *Report)

// StoreEntry holds cached channel test result for one upstream.
type StoreEntry struct {
	Report   *Report
	TestedAt time.Time
	Running  bool // true while async channel test is running
}

// StorePersist holds callbacks for SQLite persistence, injected by the caller
// to avoid an import cycle between channeltest and persist.
type StorePersist struct {
	DB         *sql.DB
	LogErr     func(op string, err error)
	Save       func(db *sql.DB, r *Report) error
	Delete     func(db *sql.DB, id string) error
	UpdateName func(db *sql.DB, id, name string) error
	Load       func(db *sql.DB) ([]*Report, error)
}

func (p *StorePersist) logErr(op string, err error) {
	if err != nil && p != nil && p.LogErr != nil {
		p.LogErr(op, err)
	}
}

// Store caches per-upstream channel test results only.
type Store struct {
	mu         sync.RWMutex
	entries    map[string]*StoreEntry // key = upstream base URL
	history    []*Report
	maxHistory int // 0 = unlimited
	runner     *Runner
	logger     *slog.Logger
	autoRun    bool
	onReport   ReportSink
	persist    *StorePersist
}

// NewStore creates a channel test store.
func NewStore(runner *Runner, logger *slog.Logger, autoRun bool, onReport ReportSink, p *StorePersist, maxHistory int) *Store {
	s := &Store{
		entries:    make(map[string]*StoreEntry),
		maxHistory: maxHistory,
		runner:     runner,
		logger:     logger,
		autoRun:    autoRun,
		onReport:   onReport,
		persist:    p,
	}

	// Load persisted history (newest-first from DB, reverse to oldest-first for in-memory slice).
	if p != nil && p.Load != nil {
		if loaded, err := p.Load(p.DB); err == nil && len(loaded) > 0 {
			for i, j := 0, len(loaded)-1; i < j; i, j = i+1, j-1 {
				loaded[i], loaded[j] = loaded[j], loaded[i]
			}
			s.history = loaded
		}
	}

	return s
}

// EnsureRun triggers async channel test for an upstream when no cached report exists.
func (s *Store) EnsureRun(upstreamBase, upstreamKey, model string) {
	s.mu.RLock()
	entry, ok := s.entries[upstreamBase]
	hasReport := ok && entry.Report != nil
	s.mu.RUnlock()

	if !hasReport && s.autoRun && upstreamBase != "" && upstreamKey != "" {
		s.triggerRunAsync(upstreamBase, upstreamKey, model)
	}
}

// HasResult checks if a channel test result is cached for this upstream.
func (s *Store) HasResult(upstreamBase string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.entries[upstreamBase]
	return ok && entry.Report != nil
}

// GetEntry returns the cached entry (may be nil).
func (s *Store) GetEntry(upstreamBase string) *StoreEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry := s.entries[upstreamBase]
	if entry == nil {
		return nil
	}
	cp := *entry
	return &cp
}

// SetResult stores a channel test result for an upstream.
func (s *Store) SetResult(upstreamBase string, report *Report) {
	s.mu.Lock()
	s.entries[upstreamBase] = &StoreEntry{
		Report:   report,
		TestedAt: time.Now(),
	}
	if report != nil {
		if report.ID == "" {
			report.ID = fmt.Sprintf("%d", time.Now().UnixNano())
		}
		s.history = append(s.history, report)
		if p := s.persist; p != nil && p.Save != nil {
			p.logErr("save_channel_history", p.Save(p.DB, report))
		}
		s.trimHistoryLocked()
	}
	s.mu.Unlock()

	if report != nil && s.onReport != nil {
		s.onReport(upstreamBase, report)
	}
}

// trimHistoryLocked removes oldest records if history exceeds maxHistory. Must hold mu.
func (s *Store) trimHistoryLocked() {
	if s.maxHistory <= 0 || len(s.history) <= s.maxHistory {
		return
	}
	excess := len(s.history) - s.maxHistory
	removed := s.history[:excess]
	s.history = s.history[excess:]
	if p := s.persist; p != nil && p.Delete != nil {
		for _, r := range removed {
			p.logErr("trim_channel_history", p.Delete(p.DB, r.ID))
		}
	}
}

// HistorySummary is a lightweight projection of Report for list views.
type HistorySummary struct {
	ID          string  `json:"id"`
	RunGroup    string  `json:"run_group,omitempty"`
	ChannelName string  `json:"channel_name,omitempty"`
	Target      string  `json:"target"`
	Model       string  `json:"model"`
	Timestamp   string  `json:"timestamp"`
	ElapsedMs   int64   `json:"elapsed_ms"`
	TotalScore  float64 `json:"total_score"`
	Grade       string  `json:"grade"`
	GradeColor  string  `json:"grade_color"`
	VerdictColor string `json:"verdict_color"`
	VerdictLabel string `json:"verdict_label"`
	ChecksTotal int     `json:"checks_total"`
	ChecksPassed int    `json:"checks_passed"`
	RunProfile  string  `json:"run_profile,omitempty"`
}

func summarizeReport(r *Report) HistorySummary {
	s := HistorySummary{
		ID:          r.ID,
		RunGroup:    r.RunGroup,
		ChannelName: r.ChannelName,
		Target:      r.Target,
		Model:       r.Model,
		Timestamp:   r.Timestamp.Format(time.RFC3339Nano),
		ElapsedMs:   r.ElapsedMs,
		RunProfile:  r.RunProfile,
	}
	if r.Score != nil {
		s.TotalScore = r.Score.TotalScore
		s.Grade = r.Score.Grade
		s.GradeColor = r.Score.GradeColor
		s.VerdictColor = r.Score.VerdictColor
		s.VerdictLabel = r.Score.VerdictLabel
		s.ChecksTotal = r.Score.ChecksTotal
		s.ChecksPassed = r.Score.ChecksPassed
	} else {
		for _, c := range r.Checks {
			s.ChecksTotal++
			if c.Pass {
				s.ChecksPassed++
			}
		}
	}
	return s
}

// ListHistorySummary returns lightweight summaries in reverse chronological order.
func (s *Store) ListHistorySummary() []HistorySummary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]HistorySummary, len(s.history))
	for i, r := range s.history {
		out[len(s.history)-1-i] = summarizeReport(r)
	}
	return out
}

// ListHistory returns all history records in reverse chronological order.
func (s *Store) ListHistory() []*Report {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Report, len(s.history))
	for i, r := range s.history {
		out[len(s.history)-1-i] = r
	}
	return out
}

// GetHistory returns a single history record by ID.
func (s *Store) GetHistory(id string) *Report {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, r := range s.history {
		if r.ID == id {
			return r
		}
	}
	return nil
}

// DeleteHistory removes a history record by ID.
func (s *Store) DeleteHistory(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, r := range s.history {
		if r.ID == id {
			s.history = append(s.history[:i], s.history[i+1:]...)
			if p := s.persist; p != nil && p.Delete != nil {
				p.logErr("delete_channel_history", p.Delete(p.DB, id))
			}
			return true
		}
	}
	return false
}

// ListEntries returns all cached entries (for admin report).
func (s *Store) ListEntries() map[string]*StoreEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]*StoreEntry, len(s.entries))
	for k, v := range s.entries {
		cp := *v
		out[k] = &cp
	}
	return out
}

// Delete removes a cached entry.
func (s *Store) Delete(upstreamBase string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, upstreamBase)
}

// triggerRunAsync starts a background channel test if not already running for this upstream.
func (s *Store) triggerRunAsync(upstreamBase, upstreamKey, model string) {
	s.mu.Lock()
	entry, ok := s.entries[upstreamBase]
	if ok && entry.Running {
		s.mu.Unlock()
		return
	}
	if !ok {
		entry = &StoreEntry{}
		s.entries[upstreamBase] = entry
	}
	entry.Running = true
	s.mu.Unlock()

	go func() {
		if model == "" {
			model = "claude-opus-4-6"
		}
		s.logger.Info("auto-channel-test started", "upstream", upstreamBase, "model", model)

		report, err := s.runner.Run(upstreamBase, upstreamKey, model, 2)

		s.mu.Lock()
		entry.Running = false
		if err != nil {
			s.mu.Unlock()
			s.logger.Warn("auto-channel-test failed", "upstream", upstreamBase, "error", err)
			return
		}

		entry.Report = report
		entry.TestedAt = time.Now()
		s.mu.Unlock()

		if s.onReport != nil {
			s.onReport(upstreamBase, report)
		}
		s.logger.Info("auto-channel-test completed",
			"upstream", upstreamBase,
			"summary", report.Summary,
		)
	}()
}

// RunSync runs a synchronous channel test for a single model and stores the result.
func (s *Store) RunSync(upstreamBase, upstreamKey, model, channelName string, concurrency int) (*Report, error) {
	report, err := s.runner.Run(upstreamBase, upstreamKey, model, concurrency)
	if err != nil {
		return nil, err
	}
	if channelName != "" {
		report.ChannelName = channelName
	} else {
		report.ChannelName = AutoChannelName(upstreamBase, model)
	}
	s.SetResult(upstreamBase, report)
	return report, nil
}

// RunMultiSync runs tests for multiple models against one target.
func (s *Store) RunMultiSync(upstreamBase, upstreamKey, channelName string, models []string, concurrency int) ([]*Report, error) {
	reports, err := s.runner.RunMulti(upstreamBase, upstreamKey, models, concurrency)
	if err != nil {
		return nil, err
	}
	for _, r := range reports {
		if channelName != "" {
			r.ChannelName = channelName
		} else {
			r.ChannelName = AutoChannelName(upstreamBase, r.Model)
		}
		s.SetResult(upstreamBase+"#"+r.Model, r)
	}
	return reports, nil
}

// RunSyncCtx runs a synchronous channel test with context for cancellation.
func (s *Store) RunSyncCtx(ctx context.Context, upstreamBase, upstreamKey, model, channelName string, concurrency int) (*Report, error) {
	report, err := s.runner.RunCtx(ctx, upstreamBase, upstreamKey, model, concurrency)
	if err != nil {
		return nil, err
	}
	if channelName != "" {
		report.ChannelName = channelName
	} else {
		report.ChannelName = AutoChannelName(upstreamBase, model)
	}
	s.SetResult(upstreamBase, report)
	return report, nil
}

// RunMultiSyncCtx runs tests for multiple models with context.
func (s *Store) RunMultiSyncCtx(ctx context.Context, upstreamBase, upstreamKey, channelName string, models []string, concurrency int) ([]*Report, error) {
	reports, err := s.runner.RunMultiCtx(ctx, upstreamBase, upstreamKey, models, concurrency)
	if err != nil {
		return nil, err
	}
	runGroup := ""
	if len(reports) > 1 {
		runGroup = fmt.Sprintf("g_%d", time.Now().UnixNano())
	}
	for _, r := range reports {
		r.RunGroup = runGroup
		if channelName != "" {
			r.ChannelName = channelName
		} else {
			r.ChannelName = AutoChannelName(upstreamBase, r.Model)
		}
		s.SetResult(upstreamBase+"#"+r.Model, r)
	}
	return reports, nil
}

// RunMultiStream runs tests for multiple models with streaming events.
// The onEvent callback receives probe-level and model-level events.
// The caller is responsible for emitting the "start" and "done" envelope events.
func (s *Store) RunMultiStream(ctx context.Context, upstreamBase, upstreamKey, channelName string, models []string, concurrency int, onEvent func(StreamEvent)) ([]*Report, error) {
	runGroup := ""
	if len(models) > 1 {
		runGroup = fmt.Sprintf("g_%d", time.Now().UnixNano())
	}

	storeReport := func(r *Report) {
		r.RunGroup = runGroup
		if channelName != "" {
			r.ChannelName = channelName
		} else {
			r.ChannelName = AutoChannelName(upstreamBase, r.Model)
		}
		key := upstreamBase
		if len(models) > 1 {
			key = upstreamBase + "#" + r.Model
		}
		s.SetResult(key, r)
	}

	reports, err := s.runner.RunMultiStream(ctx, upstreamBase, upstreamKey, models, concurrency, func(ev StreamEvent) {
		if ev.Type == "model_done" && ev.Report != nil {
			storeReport(ev.Report)
		}
		onEvent(ev)
	})
	if err != nil {
		return reports, err
	}
	return reports, nil
}

// GetHistoryGroup returns all history records with the given run group ID.
func (s *Store) GetHistoryGroup(groupID string) []*Report {
	if groupID == "" {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Report
	for _, r := range s.history {
		if r.RunGroup == groupID {
			out = append(out, r)
		}
	}
	return out
}

// stripExchanges returns a shallow copy of the report with exchanges removed from probe results.
func stripExchanges(r *Report) *Report {
	lite := *r
	lite.ProbeResults = make([]ProbeResult, len(r.ProbeResults))
	for i, pr := range r.ProbeResults {
		lite.ProbeResults[i] = ProbeResult{
			ProbeID:   pr.ProbeID,
			Label:     pr.Label,
			LatencyMs: pr.LatencyMs,
			Checks:    pr.Checks,
		}
	}
	return &lite
}

// GetHistoryLite returns a report with exchanges stripped from probe results.
func (s *Store) GetHistoryLite(id string) *Report {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, r := range s.history {
		if r.ID == id {
			return stripExchanges(r)
		}
	}
	return nil
}

// GetHistoryGroupLite returns group reports with exchanges stripped.
func (s *Store) GetHistoryGroupLite(groupID string) []*Report {
	if groupID == "" {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Report
	for _, r := range s.history {
		if r.RunGroup == groupID {
			out = append(out, stripExchanges(r))
		}
	}
	return out
}

// GetProbeResult returns a single probe's full result including exchanges.
func (s *Store) GetProbeResult(reportID, probeID string) *ProbeResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, r := range s.history {
		if r.ID == reportID {
			for i := range r.ProbeResults {
				if r.ProbeResults[i].ProbeID == probeID {
					cp := r.ProbeResults[i]
					return &cp
				}
			}
			return nil
		}
	}
	return nil
}

// RetryProbe re-runs a single probe and updates the report in-place.
func (s *Store) RetryProbe(ctx context.Context, reportID, probeID, targetKey string) (*Report, error) {
	s.mu.RLock()
	var target *Report
	for _, r := range s.history {
		if r.ID == reportID {
			target = r
			break
		}
	}
	s.mu.RUnlock()

	if target == nil {
		return nil, fmt.Errorf("report not found: %s", reportID)
	}

	var probe *Probe
	for _, p := range allProbes {
		if p.ID == probeID {
			probe = p
			break
		}
	}
	if probe == nil {
		return nil, fmt.Errorf("probe not found: %s", probeID)
	}

	result := s.runner.runSingleProbe(ctx, probe, target.Target, targetKey, target.Model)

	s.mu.Lock()
	replaced := false
	for i := range target.ProbeResults {
		if target.ProbeResults[i].ProbeID == probeID {
			target.ProbeResults[i] = result
			replaced = true
			break
		}
	}
	if !replaced {
		target.ProbeResults = append(target.ProbeResults, result)
	}

	var checks []CheckResult
	for _, pr := range target.ProbeResults {
		checks = append(checks, pr.Checks...)
	}
	InjectLabels(checks)
	target.Checks = checks

	mode := "full"
	if target.Score != nil {
		mode = target.Score.Mode
	}
	target.Score = CalculateScore(checks, mode)
	target.Summary = BuildSummaryWithScore(checks, target.Score)
	target.Recommended = RecommendFixes(checks)
	s.mu.Unlock()

	if p := s.persist; p != nil && p.Save != nil {
		p.logErr("retry_probe", p.Save(p.DB, target))
	}

	return target, nil
}

// UpdateHistoryName updates the channel name on an existing history record.
func (s *Store) UpdateHistoryName(id, name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.history {
		if r.ID == id {
			r.ChannelName = name
			if p := s.persist; p != nil && p.UpdateName != nil {
				p.logErr("update_channel_name", p.UpdateName(p.DB, id, name))
			}
			return true
		}
	}
	return false
}
