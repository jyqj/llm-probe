package monitor

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// MonitorPersist holds callbacks for SQLite persistence.
type MonitorPersist struct {
	DB                  *sql.DB
	LogErr              func(op string, err error)
	SaveTarget          func(db *sql.DB, t *Target) error
	DeleteTarget        func(db *sql.DB, id string) error
	LoadAllTargets      func(db *sql.DB) ([]*Target, error)
	SaveHealthState     func(db *sql.DB, hs *HealthState) error
	DeleteHealthStates  func(db *sql.DB, targetID string) error
	DeleteHealthState   func(db *sql.DB, targetID, model, checkType string) error
	LoadAllHealthStates func(db *sql.DB) ([]*HealthState, error)
	SaveRun             func(db *sql.DB, run *MonitorRun) error
	LoadAllRuns         func(db *sql.DB, maxRuns int) ([]*MonitorRun, error)
	TrimRuns            func(db *sql.DB, maxRuns int) error
}

func (p *MonitorPersist) logErr(op string, err error) {
	if err != nil && p != nil && p.LogErr != nil {
		p.LogErr(op, err)
	}
}

// Store is the in-memory store for monitor targets, health states, and runs.
type Store struct {
	mu      sync.RWMutex
	persist *MonitorPersist
	targets map[string]*Target
	states  map[string]*HealthState // key = targetID:model:checkType
	runs    []*MonitorRun
	maxRuns int
}

// NewStore creates a monitor store, optionally loading from SQLite.
func NewStore(p *MonitorPersist) *Store {
	s := &Store{
		persist: p,
		targets: make(map[string]*Target),
		states:  make(map[string]*HealthState),
		maxRuns: 1000,
	}
	if p != nil && p.DB != nil {
		if p.LoadAllTargets != nil {
			targets, err := p.LoadAllTargets(p.DB)
			if err != nil {
				p.logErr("load_targets", err)
			} else {
				for _, t := range targets {
					s.targets[t.ID] = t
				}
			}
		}
		if p.LoadAllHealthStates != nil {
			states, err := p.LoadAllHealthStates(p.DB)
			if err != nil {
				p.logErr("load_health_states", err)
			} else {
				for _, st := range states {
					s.states[stateKey(st.TargetID, st.Model, st.CheckType)] = st
				}
			}
		}
		if p.LoadAllRuns != nil {
			runs, err := p.LoadAllRuns(p.DB, s.maxRuns)
			if err != nil {
				p.logErr("load_runs", err)
			} else {
				s.runs = runs
			}
		}
	}

	// reconcile: ensure every target+model+checkType has a health state
	for _, t := range s.targets {
		for _, m := range t.Models {
			for _, ct := range checkTypesFor(t.CheckType) {
				key := stateKey(t.ID, m, ct)
				if s.states[key] == nil {
					st := &HealthState{TargetID: t.ID, Model: m, CheckType: ct, Status: StatusUnknown}
					s.states[key] = st
					if p != nil && p.DB != nil && p.SaveHealthState != nil {
						p.logErr("reconcile_health_state", p.SaveHealthState(p.DB, st))
					}
				}
			}
		}
	}

	return s
}

func stateKey(targetID, model, checkType string) string {
	if checkType == "" {
		checkType = "channel"
	}
	return targetID + ":" + model + ":" + checkType
}

func checkTypesFor(ct string) []string {
	switch ct {
	case "intelligence":
		return []string{"intelligence"}
	case "both":
		return []string{"channel", "intelligence"}
	default:
		return []string{"channel"}
	}
}

// ── Targets ──

func (s *Store) CreateTarget(req TargetCreateRequest) (*Target, error) {
	if req.BaseURL == "" {
		return nil, fmt.Errorf("base_url is required")
	}
	if req.APIKey == "" {
		return nil, fmt.Errorf("api_key is required")
	}
	if len(req.Models) == 0 {
		req.Models = []string{"claude-sonnet-4-6"}
	}

	interval := 5 * time.Minute
	if req.Interval != "" {
		d, err := time.ParseDuration(req.Interval)
		if err != nil {
			return nil, fmt.Errorf("invalid interval: %w", err)
		}
		if d < 30*time.Second {
			return nil, fmt.Errorf("interval must be >= 30s")
		}
		interval = d
	}

	var jitter time.Duration
	if req.Jitter != "" {
		d, err := time.ParseDuration(req.Jitter)
		if err != nil {
			return nil, fmt.Errorf("invalid jitter: %w", err)
		}
		jitter = d
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	checkType := req.CheckType
	if checkType == "" {
		checkType = "channel"
	}
	if !validCheckType(checkType) {
		return nil, fmt.Errorf("invalid check_type: %s", checkType)
	}

	intLimit := 3
	if req.IntelligenceLimit != nil {
		intLimit = *req.IntelligenceLimit
	}
	intMaxLimit := 0
	if req.IntelligenceMaxLimit != nil {
		intMaxLimit = *req.IntelligenceMaxLimit
	}
	intThreshold := 1.0
	if req.IntelligenceThreshold != nil {
		intThreshold = *req.IntelligenceThreshold
	}

	var channelInterval time.Duration
	if req.ChannelInterval != "" {
		d, err := time.ParseDuration(req.ChannelInterval)
		if err != nil {
			return nil, fmt.Errorf("invalid channel_interval: %w", err)
		}
		channelInterval = d
	}
	var intelligenceInterval time.Duration
	if req.IntelligenceInterval != "" {
		d, err := time.ParseDuration(req.IntelligenceInterval)
		if err != nil {
			return nil, fmt.Errorf("invalid intelligence_interval: %w", err)
		}
		intelligenceInterval = d
	}

	now := time.Now()
	t := &Target{
		ID:                    newID(),
		Name:                  req.Name,
		BaseURL:               req.BaseURL,
		APIKey:                req.APIKey,
		Models:                req.Models,
		Interval:              interval,
		Jitter:                jitter,
		Enabled:               enabled,
		CreatedAt:             now,
		UpdatedAt:             now,
		CheckType:             checkType,
		Profile:               req.Profile,
		ChannelInterval:       channelInterval,
		IntelligenceInterval:  intelligenceInterval,
		IntelligenceDataset:   req.IntelligenceDataset,
		IntelligenceLimit:     intLimit,
		IntelligenceMaxLimit:  intMaxLimit,
		IntelligenceThreshold: intThreshold,
		BaselineID:            req.BaselineID,
		Effort:                req.Effort,
		ThinkingMode:          req.ThinkingMode,
	}
	if req.MaxTokens != nil {
		t.MaxTokens = *req.MaxTokens
	}
	if t.Name == "" {
		t.Name = t.BaseURL
	}

	if p := s.persist; p != nil && p.DB != nil && p.SaveTarget != nil {
		if err := p.SaveTarget(p.DB, t); err != nil {
			return nil, fmt.Errorf("persist target: %w", err)
		}
	}

	s.mu.Lock()
	s.targets[t.ID] = t
	for _, m := range t.Models {
		for _, ct := range checkTypesFor(t.CheckType) {
			key := stateKey(t.ID, m, ct)
			if s.states[key] == nil {
				st := &HealthState{
					TargetID:  t.ID,
					Model:     m,
					CheckType: ct,
					Status:    StatusUnknown,
				}
				s.states[key] = st
				if p := s.persist; p != nil && p.DB != nil && p.SaveHealthState != nil {
					p.logErr("save_initial_health_state", p.SaveHealthState(p.DB, st))
				}
			}
		}
	}
	s.mu.Unlock()

	return t, nil
}

func (s *Store) GetTarget(id string) *Target {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.targets[id]
}

func (s *Store) ListTargets() []*Target {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Target, 0, len(s.targets))
	for _, t := range s.targets {
		out = append(out, t)
	}
	return out
}

func (s *Store) UpdateTarget(id string, req TargetUpdateRequest) (*Target, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	orig := s.targets[id]
	if orig == nil {
		return nil, fmt.Errorf("target not found")
	}

	// work on a copy so memory is unchanged if DB write fails
	cp := *orig
	t := &cp

	if req.Name != nil {
		t.Name = *req.Name
	}
	if req.BaseURL != nil {
		t.BaseURL = *req.BaseURL
	}
	if req.APIKey != nil {
		t.APIKey = *req.APIKey
	}
	if req.Models != nil {
		t.Models = req.Models
	}
	if req.Interval != nil {
		d, err := time.ParseDuration(*req.Interval)
		if err != nil {
			return nil, fmt.Errorf("invalid interval: %w", err)
		}
		t.Interval = d
	}
	if req.Jitter != nil {
		d, err := time.ParseDuration(*req.Jitter)
		if err != nil {
			return nil, fmt.Errorf("invalid jitter: %w", err)
		}
		t.Jitter = d
	}
	if req.Enabled != nil {
		t.Enabled = *req.Enabled
	}
	if req.CheckType != nil {
		if !validCheckType(*req.CheckType) {
			return nil, fmt.Errorf("invalid check_type: %s", *req.CheckType)
		}
		t.CheckType = *req.CheckType
	}
	if req.Profile != nil {
		t.Profile = *req.Profile
	}
	if req.ChannelInterval != nil {
		d, err := time.ParseDuration(*req.ChannelInterval)
		if err != nil {
			return nil, fmt.Errorf("invalid channel_interval: %w", err)
		}
		t.ChannelInterval = d
	}
	if req.IntelligenceInterval != nil {
		d, err := time.ParseDuration(*req.IntelligenceInterval)
		if err != nil {
			return nil, fmt.Errorf("invalid intelligence_interval: %w", err)
		}
		t.IntelligenceInterval = d
	}
	if req.IntelligenceDataset != nil {
		t.IntelligenceDataset = *req.IntelligenceDataset
	}
	if req.IntelligenceLimit != nil {
		t.IntelligenceLimit = *req.IntelligenceLimit
	}
	if req.IntelligenceMaxLimit != nil {
		t.IntelligenceMaxLimit = *req.IntelligenceMaxLimit
	}
	if req.IntelligenceThreshold != nil {
		t.IntelligenceThreshold = *req.IntelligenceThreshold
	}
	if req.BaselineID != nil {
		t.BaselineID = *req.BaselineID
	}
	if req.Effort != nil {
		t.Effort = *req.Effort
	}
	if req.ThinkingMode != nil {
		t.ThinkingMode = *req.ThinkingMode
	}
	if req.MaxTokens != nil {
		t.MaxTokens = *req.MaxTokens
	}
	t.UpdatedAt = time.Now()

	// persist first — fail means memory stays unchanged
	if p := s.persist; p != nil && p.DB != nil && p.SaveTarget != nil {
		if err := p.SaveTarget(p.DB, t); err != nil {
			return nil, fmt.Errorf("persist target update: %w", err)
		}
	}

	// commit to memory
	s.targets[id] = t

	// reconcile model+checkType states if models or checkType changed
	if req.Models != nil || req.CheckType != nil {
		oldKeys := make(map[string]bool)
		for _, m := range orig.Models {
			for _, ct := range checkTypesFor(orig.CheckType) {
				oldKeys[stateKey(t.ID, m, ct)] = true
			}
		}
		newKeys := make(map[string]bool)
		for _, m := range t.Models {
			for _, ct := range checkTypesFor(t.CheckType) {
				newKeys[stateKey(t.ID, m, ct)] = true
			}
		}
		for k := range oldKeys {
			if !newKeys[k] {
				delete(s.states, k)
			}
		}
		if p := s.persist; p != nil && p.DB != nil && p.DeleteHealthStates != nil {
			p.logErr("reconcile_delete_states", p.DeleteHealthStates(p.DB, t.ID))
		}
		for _, m := range t.Models {
			for _, ct := range checkTypesFor(t.CheckType) {
				key := stateKey(t.ID, m, ct)
				if s.states[key] == nil {
					st := &HealthState{TargetID: t.ID, Model: m, CheckType: ct, Status: StatusUnknown}
					s.states[key] = st
					if p := s.persist; p != nil && p.DB != nil && p.SaveHealthState != nil {
						p.logErr("save_new_state", p.SaveHealthState(p.DB, st))
					}
				}
			}
		}
	}

	return t, nil
}

func (s *Store) DeleteTarget(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.targets[id] == nil {
		return false
	}
	t := s.targets[id]
	if p := s.persist; p != nil && p.DB != nil {
		if p.DeleteTarget != nil {
			p.logErr("delete_target", p.DeleteTarget(p.DB, id))
		}
		if p.DeleteHealthStates != nil {
			p.logErr("delete_health_states", p.DeleteHealthStates(p.DB, id))
		}
	}
	delete(s.targets, id)
	for _, m := range t.Models {
		for _, ct := range checkTypesFor(t.CheckType) {
			delete(s.states, stateKey(id, m, ct))
		}
	}
	return true
}

// ── States ──

func (s *Store) GetState(targetID, model, checkType string) *HealthState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st := s.states[stateKey(targetID, model, checkType)]
	if st == nil {
		return nil
	}
	cp := *st
	return &cp
}

// GetHealthState returns the health state for adaptive scheduling.
func (s *Store) GetHealthState(targetID, model, checkType string) *HealthState {
	return s.GetState(targetID, model, checkType)
}

func (s *Store) ListStates() []*HealthState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*HealthState, 0, len(s.states))
	for _, st := range s.states {
		cp := *st
		out = append(out, &cp)
	}
	return out
}

// ── Runs ──

func (s *Store) RecordRun(run *MonitorRun) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if run.ID == "" {
		run.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	key := stateKey(run.TargetID, run.Model, run.CheckType)
	st := s.states[key]
	if st == nil {
		st = &HealthState{
			TargetID:  run.TargetID,
			Model:     run.Model,
			CheckType: run.CheckType,
			Status:    StatusUnknown,
		}
		s.states[key] = st
	}
	st.Transition(run)

	s.runs = append(s.runs, run)
	if len(s.runs) > s.maxRuns {
		s.runs = s.runs[len(s.runs)-s.maxRuns:]
	}

	if p := s.persist; p != nil && p.DB != nil {
		if p.SaveRun != nil {
			p.logErr("save_run", p.SaveRun(p.DB, run))
		}
		if p.SaveHealthState != nil {
			p.logErr("save_health_state", p.SaveHealthState(p.DB, st))
		}
		if p.TrimRuns != nil {
			p.logErr("trim_runs", p.TrimRuns(p.DB, s.maxRuns))
		}
	}
}

func (s *Store) ListRuns(targetID string, limit int) []*MonitorRun {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}

	var out []*MonitorRun
	for i := len(s.runs) - 1; i >= 0 && len(out) < limit; i-- {
		r := s.runs[i]
		if targetID == "" || r.TargetID == targetID {
			out = append(out, r)
		}
	}
	return out
}

func (s *Store) GetRun(id string) *MonitorRun {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, r := range s.runs {
		if r.ID == id {
			return r
		}
	}
	return nil
}

// EnabledTargets returns all enabled targets (for the scheduler).
func (s *Store) EnabledTargets() []*Target {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Target
	for _, t := range s.targets {
		if t.Enabled {
			cp := *t
			out = append(out, &cp)
		}
	}
	return out
}
