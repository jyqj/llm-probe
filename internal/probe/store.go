package probe

import (
	"log/slog"
	"sync"
	"time"

	"bedrock-gateway/internal/config"
)

// StoreEntry holds cached probe result for one upstream.
type StoreEntry struct {
	Report    *ProbeReport
	Config    config.DisguiseConfig
	ProbedAt  time.Time
	Probing   bool // true while async probe is running
}

// Store caches per-upstream probe results and DisguiseConfig.
type Store struct {
	mu       sync.RWMutex
	entries  map[string]*StoreEntry // key = upstream base URL
	prober   *Prober
	fallback config.DisguiseConfig  // global fallback config
	logger   *slog.Logger
	autoProbe bool
}

// NewStore creates a probe store.
func NewStore(prober *Prober, fallback config.DisguiseConfig, logger *slog.Logger, autoProbe bool) *Store {
	return &Store{
		entries:   make(map[string]*StoreEntry),
		prober:    prober,
		fallback:  fallback,
		logger:    logger,
		autoProbe: autoProbe,
	}
}

// GetConfig returns the DisguiseConfig for a given upstream base URL.
// If no probe result is cached, returns the global fallback.
// If autoProbe is enabled and upstream hasn't been probed, triggers async probe.
func (s *Store) GetConfig(upstreamBase, upstreamKey, model string) config.DisguiseConfig {
	s.mu.RLock()
	entry, ok := s.entries[upstreamBase]
	s.mu.RUnlock()

	if ok && entry.Report != nil {
		return entry.Config
	}

	// No cached result — maybe trigger async probe
	if s.autoProbe && upstreamBase != "" && upstreamKey != "" {
		s.triggerAsync(upstreamBase, upstreamKey, model)
	}

	return s.fallback
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
	return s.entries[upstreamBase]
}

// SetResult stores a probe result for an upstream.
func (s *Store) SetResult(upstreamBase string, report *ProbeReport, cfg config.DisguiseConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[upstreamBase] = &StoreEntry{
		Report:   report,
		Config:   cfg,
		ProbedAt: time.Now(),
	}
}

// SetFallback updates the global fallback DisguiseConfig.
func (s *Store) SetFallback(cfg config.DisguiseConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fallback = cfg
}

// ListEntries returns all cached entries (for admin report).
func (s *Store) ListEntries() map[string]*StoreEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]*StoreEntry, len(s.entries))
	for k, v := range s.entries {
		out[k] = v
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
		defer s.mu.Unlock()
		entry.Probing = false

		if err != nil {
			s.logger.Warn("auto-probe failed", "upstream", upstreamBase, "error", err)
			return
		}

		entry.Report = report
		entry.Config = report.Recommended
		entry.ProbedAt = time.Now()
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
	s.SetResult(upstreamBase, report, report.Recommended)
	return report, nil
}
