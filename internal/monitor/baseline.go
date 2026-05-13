package monitor

import (
	"database/sql"
	"sync"
	"time"

	"detector-service/internal/channeltest"
	"detector-service/internal/intelligence"
)

// BaselinePersist holds callbacks for SQLite persistence.
type BaselinePersist struct {
	DB      *sql.DB
	LogErr  func(op string, err error)
	Save    func(db *sql.DB, b *Baseline) error
	Delete  func(db *sql.DB, id string) error
	LoadAll func(db *sql.DB) ([]*Baseline, error)
}

func (p *BaselinePersist) logErr(op string, err error) {
	if err != nil && p != nil && p.LogErr != nil {
		p.LogErr(op, err)
	}
}

// Baseline is a recorded "ground truth" run for comparison.
type Baseline struct {
	ID                 string                  `json:"id"`
	Name               string                  `json:"name"`
	Model              string                  `json:"model"`
	ThinkingEffort     string                  `json:"thinking_effort"`
	Effort             string                  `json:"effort,omitempty"`
	ThinkingMode       string                  `json:"thinking_mode,omitempty"`
	MaxTokens          int                     `json:"max_tokens,omitempty"`
	ChannelReport      *channeltest.Report     `json:"channel_report,omitempty"`
	IntelligenceReport *intelligence.RunReport  `json:"intelligence_report,omitempty"`
	CreatedAt          time.Time               `json:"created_at"`
}

// BaselineCreateRequest is the API input for recording a baseline.
type BaselineCreateRequest struct {
	Name           string `json:"name"`
	BaseURL        string `json:"base_url"`
	APIKey         string `json:"api_key"`
	Model          string `json:"model"`
	ThinkingEffort string `json:"thinking_effort,omitempty"`
	Dataset        string `json:"dataset,omitempty"`
	Effort         string `json:"effort,omitempty"`
	ThinkingMode   string `json:"thinking_mode,omitempty"`
	MaxTokens      int    `json:"max_tokens,omitempty"`
}

// BaselineStore holds baselines in memory, backed by SQLite.
type BaselineStore struct {
	mu        sync.RWMutex
	persist   *BaselinePersist
	baselines map[string]*Baseline
}

// NewBaselineStore creates a baseline store, optionally loading from SQLite.
func NewBaselineStore(p *BaselinePersist) *BaselineStore {
	s := &BaselineStore{
		persist:   p,
		baselines: make(map[string]*Baseline),
	}
	if p != nil && p.DB != nil && p.LoadAll != nil {
		if baselines, err := p.LoadAll(p.DB); err == nil {
			for _, b := range baselines {
				s.baselines[b.ID] = b
			}
		}
	}
	return s
}

func (s *BaselineStore) Add(b *Baseline) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if b.ID == "" {
		b.ID = newID()
	}
	if p := s.persist; p != nil && p.DB != nil && p.Save != nil {
		p.logErr("save_baseline", p.Save(p.DB, b))
	}
	s.baselines[b.ID] = b
}

func (s *BaselineStore) Get(id string) *Baseline {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.baselines[id]
}

func (s *BaselineStore) List() []*Baseline {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Baseline, 0, len(s.baselines))
	for _, b := range s.baselines {
		out = append(out, b)
	}
	return out
}

func (s *BaselineStore) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.baselines[id]; !ok {
		return false
	}
	if p := s.persist; p != nil && p.DB != nil && p.Delete != nil {
		p.logErr("delete_baseline", p.Delete(p.DB, id))
	}
	delete(s.baselines, id)
	return true
}
