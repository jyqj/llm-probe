package channeltest

import (
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

// Store caches per-upstream channel test results only.
type Store struct {
	mu       sync.RWMutex
	entries  map[string]*StoreEntry // key = upstream base URL
	runner   *Runner
	logger   *slog.Logger
	autoRun  bool
	onReport ReportSink
}

// NewStore creates a channel test store.
func NewStore(runner *Runner, logger *slog.Logger, autoRun bool, onReport ReportSink) *Store {
	return &Store{
		entries:  make(map[string]*StoreEntry),
		runner:   runner,
		logger:   logger,
		autoRun:  autoRun,
		onReport: onReport,
	}
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
	s.mu.Unlock()

	if report != nil && s.onReport != nil {
		s.onReport(upstreamBase, report)
	}
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

		report, err := s.runner.Run(upstreamBase, upstreamKey, model, true)

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

// RunSync runs a synchronous channel test and stores the result.
func (s *Store) RunSync(upstreamBase, upstreamKey, model string, quick bool) (*Report, error) {
	report, err := s.runner.Run(upstreamBase, upstreamKey, model, quick)
	if err != nil {
		return nil, err
	}
	s.SetResult(upstreamBase, report)
	return report, nil
}
