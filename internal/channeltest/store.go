package channeltest

import (
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
	mu       sync.RWMutex
	entries  map[string]*StoreEntry // key = upstream base URL
	history  []*Report
	runner   *Runner
	logger   *slog.Logger
	autoRun  bool
	onReport ReportSink
	persist  *StorePersist
}

// NewStore creates a channel test store.
func NewStore(runner *Runner, logger *slog.Logger, autoRun bool, onReport ReportSink, p *StorePersist) *Store {
	s := &Store{
		entries:  make(map[string]*StoreEntry),
		runner:   runner,
		logger:   logger,
		autoRun:  autoRun,
		onReport: onReport,
		persist:  p,
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
	}
	s.mu.Unlock()

	if report != nil && s.onReport != nil {
		s.onReport(upstreamBase, report)
	}
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
