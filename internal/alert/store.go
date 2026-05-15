package alert

import (
	"database/sql"
	"sync"
	"time"
)

// StorePersist holds callbacks for SQLite persistence, injected by the caller
// to avoid an import cycle between alert and persist.
type StorePersist struct {
	DB     *sql.DB
	LogErr func(op string, err error)
	Save   func(db *sql.DB, ev *Event) error
	Update func(db *sql.DB, ev *Event) error
	Load   func(db *sql.DB, maxEvents int) ([]*Event, error)
	Trim   func(db *sql.DB, maxEvents int) error
}

func (p *StorePersist) logErr(op string, err error) {
	if err != nil && p != nil && p.LogErr != nil {
		p.LogErr(op, err)
	}
}

// Store is the in-memory alert event store.
type Store struct {
	mu        sync.RWMutex
	persist   *StorePersist
	events    []*Event
	lastFired map[string]time.Time // dedupKey → last fired time
	maxEvents int
}

// NewStore creates an alert store.
func NewStore(p *StorePersist) *Store {
	s := &Store{
		persist:   p,
		events:    make([]*Event, 0),
		lastFired: make(map[string]time.Time),
		maxEvents: 500,
	}
	if p != nil && p.DB != nil && p.Load != nil {
		loaded, err := p.Load(p.DB, 500)
		if err == nil && len(loaded) > 0 {
			s.events = loaded
			// rebuild lastFired from loaded events
			for _, ev := range loaded {
				key := dedupKeyFor(ev)
				if existing, ok := s.lastFired[key]; !ok || ev.FiredAt.After(existing) {
					s.lastFired[key] = ev.FiredAt
				}
			}
		}
	}
	return s
}

// RecordEvent stores an alert event and updates the cooldown tracker.
func (s *Store) RecordEvent(ev *Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, ev)
	if len(s.events) > s.maxEvents {
		s.events = s.events[len(s.events)-s.maxEvents:]
	}
	dedupKey := dedupKeyFor(ev)
	s.lastFired[dedupKey] = ev.FiredAt
	if p := s.persist; p != nil && p.DB != nil {
		if p.Save != nil {
			p.logErr("save_alert_event", p.Save(p.DB, ev))
		}
		if p.Trim != nil {
			p.logErr("trim_alert_events", p.Trim(p.DB, s.maxEvents))
		}
	}
}

// IsInCooldown checks if a rule+target+model combo is within cooldown.
func (s *Store) IsInCooldown(dedupKey string, cooldown time.Duration) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	last, ok := s.lastFired[dedupKey]
	if !ok {
		return false
	}
	return time.Since(last) < cooldown
}

// ResolveEvent marks the most recent firing event for a dedup key as resolved.
func (s *Store) ResolveEvent(dedupKey string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for i := len(s.events) - 1; i >= 0; i-- {
		ev := s.events[i]
		key := dedupKeyFor(ev)
		if key == dedupKey && ev.Status == EventFiring {
			ev.Status = EventResolved
			ev.ResolvedAt = &now
			if p := s.persist; p != nil && p.DB != nil && p.Update != nil {
				p.logErr("resolve_alert_event", p.Update(p.DB, ev))
			}
			break
		}
	}
}

func dedupKeyFor(ev *Event) string {
	if ev == nil {
		return ""
	}
	return ev.RuleName + ":" + ev.TargetID + ":" + ev.Model + ":" + ev.CheckType
}

// ListEvents returns recent events in reverse chronological order.
func (s *Store) ListEvents(limit int) []*Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 {
		limit = 50
	}
	out := make([]*Event, 0, limit)
	for i := len(s.events) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, s.events[i])
	}
	return out
}

// ActiveEvents returns all currently firing events.
func (s *Store) ActiveEvents() []*Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Event
	for _, ev := range s.events {
		if ev.Status == EventFiring {
			out = append(out, ev)
		}
	}
	return out
}
