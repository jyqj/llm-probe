package probe

import (
	"log/slog"
	"sync"
	"time"
)

// ReportSink receives completed probe reports for external consumption.
type ReportSink func(upstreamBase string, report *ProbeReport)

// StoreEntry holds cached probe result for one upstream.
type StoreEntry struct {
	Report   *ProbeReport
	ProbedAt time.Time
	Probing  bool // true while async probe is running
}

// Store caches per-upstream probe results only.
type Store struct {
	mu        sync.RWMutex
	entries   map[string]*StoreEntry // key = upstream base URL
	prober    *Prober
	logger    *slog.Logger
	autoProbe bool
	onReport  ReportSink
}

// NewStore creates a probe store.
func NewStore(prober *Prober, logger *slog.Logger, autoProbe bool, onReport ReportSink) *Store {
	return &Store{
		entries:   make(map[string]*StoreEntry),
		prober:    prober,
		logger:    logger,
		autoProbe: autoProbe,
		onReport:  onReport,
	}
}

// EnsureProbe triggers async probe for an upstream when no cached report exists.
func (s *Store) EnsureProbe(upstreamBase, upstreamKey, model string) {
	s.mu.RLock()
	entry, ok := s.entries[upstreamBase]
	hasReport := ok && entry.Report != nil
	s.mu.RUnlock()

	if !hasReport && s.autoProbe && upstreamBase != "" && upstreamKey != "" {
		s.triggerAsync(upstreamBase, upstreamKey, model)
	}
}

// HasResult checks if a probe result is cached for this upstream.
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

// SetResult stores a probe result for an upstream.
func (s *Store) SetResult(upstreamBase string, report *ProbeReport) {
	s.mu.Lock()
	s.entries[upstreamBase] = &StoreEntry{
		Report:   report,
		ProbedAt: time.Now(),
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

// triggerAsync starts a background probe if not already running for this upstream.
func (s *Store) triggerAsync(upstreamBase, upstreamKey, model string) {
	s.mu.Lock()
	entry, ok := s.entries[upstreamBase]
	if ok && entry.Probing {
		s.mu.Unlock()
		return
	}
	if !ok {
		entry = &StoreEntry{}
		s.entries[upstreamBase] = entry
	}
	entry.Probing = true
	s.mu.Unlock()

	go func() {
		if model == "" {
			model = "claude-opus-4-6"
		}
		s.logger.Info("auto-probe started", "upstream", upstreamBase, "model", model)

		report, err := s.prober.RunSuite(upstreamBase, upstreamKey, model, true)

		s.mu.Lock()
		entry.Probing = false
		if err != nil {
			s.mu.Unlock()
			s.logger.Warn("auto-probe failed", "upstream", upstreamBase, "error", err)
			return
		}

		entry.Report = report
		entry.ProbedAt = time.Now()
		s.mu.Unlock()

		if s.onReport != nil {
			s.onReport(upstreamBase, report)
		}
		s.logger.Info("auto-probe completed",
			"upstream", upstreamBase,
			"summary", report.Summary,
		)
	}()
}

// ProbeSync runs a synchronous probe and stores the result.
func (s *Store) ProbeSync(upstreamBase, upstreamKey, model string, quick bool) (*ProbeReport, error) {
	report, err := s.prober.RunSuite(upstreamBase, upstreamKey, model, quick)
	if err != nil {
		return nil, err
	}
	s.SetResult(upstreamBase, report)
	return report, nil
}
