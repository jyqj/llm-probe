package monitor

import (
	"context"
	"log/slog"
	"math/rand"
	"sync"
	"time"
)

// Scheduler runs periodic monitor checks for all enabled targets.
type Scheduler struct {
	store  *Store
	runner *MonitorRunner
	logger *slog.Logger

	mu     sync.Mutex
	cancel context.CancelFunc
	due    map[string]time.Time // targetID:model → next due time
}

// NewScheduler creates a scheduler.
func NewScheduler(store *Store, runner *MonitorRunner, logger *slog.Logger) *Scheduler {
	return &Scheduler{
		store:  store,
		runner: runner,
		logger: logger,
		due:    make(map[string]time.Time),
	}
}

// Start begins the scheduler loop. Call Stop() to shut it down.
func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.mu.Unlock()

	s.logger.Info("monitor scheduler started")
	go s.loop(ctx)
}

// Stop shuts down the scheduler.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.mu.Unlock()
	s.logger.Info("monitor scheduler stopped")
}

func (s *Scheduler) loop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.tick(now)
		}
	}
}

func (s *Scheduler) tick(now time.Time) {
	targets := s.store.EnabledTargets()
	for _, t := range targets {
		for _, model := range t.Models {
			key := stateKey(t.ID, model)
			due, ok := s.due[key]
			if !ok {
				jitter := time.Duration(rand.Int63n(int64(t.Interval) / 2))
				s.due[key] = now.Add(jitter)
				continue
			}
			if now.Before(due) {
				continue
			}
			go s.runOne(t, model)
			next := now.Add(t.Interval)
			if t.Jitter > 0 {
				next = next.Add(time.Duration(rand.Int63n(int64(t.Jitter))))
			}
			s.due[key] = next
		}
	}
}

func (s *Scheduler) runOne(target *Target, model string) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("monitor run panic", "target", target.Name, "model", model, "panic", r)
		}
	}()
	s.runner.RunTarget(target, model)
}
