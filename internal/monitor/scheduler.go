package monitor

import (
	"context"
	"log/slog"
	"math/rand"
	"sync"
	"time"
)

// SchedulerConfig controls resource limits for the scheduler.
type SchedulerConfig struct {
	MaxConcurrent int           // global max concurrent runs (default 4)
	RunTimeout    time.Duration // per-run timeout (default 5m)
	BackoffBase   time.Duration // backoff base for consecutive errors (default 30s)
	BackoffMax    time.Duration // max backoff duration (default 15m)
}

func (c SchedulerConfig) withDefaults() SchedulerConfig {
	if c.MaxConcurrent <= 0 {
		c.MaxConcurrent = 4
	}
	if c.RunTimeout <= 0 {
		c.RunTimeout = 5 * time.Minute
	}
	if c.BackoffBase <= 0 {
		c.BackoffBase = 30 * time.Second
	}
	if c.BackoffMax <= 0 {
		c.BackoffMax = 15 * time.Minute
	}
	return c
}

// Scheduler runs periodic monitor checks for all enabled targets.
// Channel and intelligence checks are scheduled independently with separate intervals.
// Adaptive frequency: shorter intervals when unhealthy, longer when stable.
// Includes global concurrency limit, per-run timeout, and error backoff.
type Scheduler struct {
	store  *Store
	runner *MonitorRunner
	logger *slog.Logger
	cfg    SchedulerConfig

	mu      sync.Mutex
	cancel  context.CancelFunc
	due     map[string]time.Time // key → next due time
	running map[string]bool      // key → currently executing
	backoff map[string]int       // key → consecutive error count for backoff
	sem     chan struct{}         // global concurrency semaphore
}

// NewScheduler creates a scheduler with default config.
func NewScheduler(store *Store, runner *MonitorRunner, logger *slog.Logger) *Scheduler {
	return NewSchedulerWithConfig(store, runner, logger, SchedulerConfig{})
}

// NewSchedulerWithConfig creates a scheduler with explicit config.
func NewSchedulerWithConfig(store *Store, runner *MonitorRunner, logger *slog.Logger, cfg SchedulerConfig) *Scheduler {
	cfg = cfg.withDefaults()
	return &Scheduler{
		store:   store,
		runner:  runner,
		logger:  logger,
		cfg:     cfg,
		due:     make(map[string]time.Time),
		running: make(map[string]bool),
		backoff: make(map[string]int),
		sem:     make(chan struct{}, cfg.MaxConcurrent),
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
			s.tick(ctx, now)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context, now time.Time) {
	targets := s.store.EnabledTargets()
	for _, t := range targets {
		checkType := t.CheckType
		if checkType == "" {
			checkType = "channel"
		}

		for _, model := range t.Models {
			switch checkType {
			case "channel":
				s.scheduleCheck(ctx, now, t, model, "channel")
			case "intelligence":
				s.scheduleCheck(ctx, now, t, model, "intelligence")
			case "both":
				s.scheduleCheck(ctx, now, t, model, "channel")
				s.scheduleCheck(ctx, now, t, model, "intelligence")
			}
		}
	}
}

func (s *Scheduler) scheduleCheck(ctx context.Context, now time.Time, t *Target, model, checkKind string) {
	key := schedKey(t.ID, model, checkKind)
	baseInterval := t.Interval
	if checkKind == "channel" {
		baseInterval = t.EffectiveChannelInterval()
	} else if checkKind == "intelligence" {
		baseInterval = t.EffectiveIntelligenceInterval()
	}
	if baseInterval < 30*time.Second {
		baseInterval = 30 * time.Second
	}

	due, ok := s.due[key]
	if !ok {
		jitter := time.Duration(rand.Int63n(int64(baseInterval) / 2))
		s.due[key] = now.Add(jitter)
		return
	}
	if now.Before(due) {
		return
	}
	if !s.markRunning(key) {
		return
	}

	// try to acquire global concurrency slot; skip if full
	select {
	case s.sem <- struct{}{}:
	default:
		s.clearRunning(key)
		return
	}

	go s.runCheck(ctx, key, t, model, checkKind)

	interval := s.adaptiveInterval(baseInterval, t.ID, model, checkKind)
	if bo := s.backoffDuration(key); bo > interval {
		interval = bo
	}
	next := now.Add(interval)
	if t.Jitter > 0 {
		next = next.Add(time.Duration(rand.Int63n(int64(t.Jitter))))
	}
	s.due[key] = next
}

// adaptiveInterval adjusts interval based on current health state.
func (s *Scheduler) adaptiveInterval(base time.Duration, targetID, model, checkType string) time.Duration {
	hs := s.store.GetHealthState(targetID, model, checkType)
	if hs == nil {
		return base
	}
	switch hs.Status {
	case StatusCritical:
		adj := base / 4
		if adj < 30*time.Second {
			return 30 * time.Second
		}
		return adj
	case StatusWarning:
		adj := base / 2
		if adj < 30*time.Second {
			return 30 * time.Second
		}
		return adj
	case StatusOK:
		if hs.ConsecOK >= 5 {
			adj := base * 2
			maxInterval := 30 * time.Minute
			if adj > maxInterval {
				return maxInterval
			}
			return adj
		}
		return base
	default:
		return base
	}
}

func schedKey(targetID, model, checkKind string) string {
	return targetID + ":" + model + ":" + checkKind
}

func (s *Scheduler) markRunning(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running[key] {
		return false
	}
	s.running[key] = true
	return true
}

func (s *Scheduler) clearRunning(key string) {
	s.mu.Lock()
	delete(s.running, key)
	s.mu.Unlock()
}

func (s *Scheduler) runCheck(ctx context.Context, key string, target *Target, model, checkKind string) {
	defer s.clearRunning(key)
	defer func() { <-s.sem }()
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("monitor run panic", "target", target.Name, "model", model, "kind", checkKind, "panic", r)
			s.recordError(key)
		}
	}()

	runCtx, cancel := context.WithTimeout(ctx, s.cfg.RunTimeout)
	defer cancel()

	virtualTarget := *target
	virtualTarget.CheckType = checkKind
	run := s.runner.RunTargetCtx(runCtx, &virtualTarget, model)

	if run.Error != "" || run.IntelligenceError != "" {
		s.recordError(key)
	} else {
		s.clearBackoff(key)
	}
}

func (s *Scheduler) recordError(key string) {
	s.mu.Lock()
	s.backoff[key]++
	s.mu.Unlock()
}

func (s *Scheduler) clearBackoff(key string) {
	s.mu.Lock()
	delete(s.backoff, key)
	s.mu.Unlock()
}

func (s *Scheduler) backoffDuration(key string) time.Duration {
	s.mu.Lock()
	count := s.backoff[key]
	s.mu.Unlock()
	if count <= 0 {
		return 0
	}
	d := s.cfg.BackoffBase
	for i := 1; i < count && i < 8; i++ {
		d *= 2
	}
	if d > s.cfg.BackoffMax {
		d = s.cfg.BackoffMax
	}
	return d
}
