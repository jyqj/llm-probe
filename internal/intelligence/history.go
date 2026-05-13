package intelligence

import (
	"database/sql"
	"sync"
)

// RunReportSummary is a lightweight view of RunReport for list display.
type RunReportSummary struct {
	ID             string   `json:"id"`
	DatasetName    string   `json:"dataset_name"`
	Model          string   `json:"model"`
	Target         string   `json:"target"`
	Thinking       bool     `json:"thinking,omitempty"`
	Effort         string   `json:"effort,omitempty"`
	ThinkingMode   string   `json:"thinking_mode,omitempty"`
	TaskTotal      int      `json:"task_total"`
	TaskCompleted  int      `json:"task_completed"`
	TaskErrors     int      `json:"task_errors"`
	ElapsedMs      int64    `json:"elapsed_ms"`
	StartedAt      string   `json:"started_at"`
	ScoreTotal     *float64 `json:"score_total,omitempty"`
	PassRate       *float64 `json:"pass_rate,omitempty"`
	TotalEvaluated int      `json:"total_evaluated,omitempty"`
	TotalPassed    int      `json:"total_passed,omitempty"`
}

// HistoryPersist holds callbacks for SQLite persistence, injected by the caller
// to avoid an import cycle between intelligence and persist.
type HistoryPersist struct {
	DB     *sql.DB
	LogErr func(op string, err error)
	Save   func(db *sql.DB, r *RunReport) error
	Delete func(db *sql.DB, id string) error
	Load   func(db *sql.DB) ([]*RunReport, error)
}

func (p *HistoryPersist) logErr(op string, err error) {
	if err != nil && p != nil && p.LogErr != nil {
		p.LogErr(op, err)
	}
}

// HistoryStore keeps all benchmark run reports in memory, backed by SQLite.
type HistoryStore struct {
	mu      sync.RWMutex
	records []*RunReport
	persist *HistoryPersist
}

// NewHistoryStore creates a new HistoryStore, loading existing records from SQLite.
func NewHistoryStore(p *HistoryPersist) *HistoryStore {
	h := &HistoryStore{persist: p}
	if p != nil && p.Load != nil {
		if rows, err := p.Load(p.DB); err == nil && len(rows) > 0 {
			// rows come back newest-first; reverse to chronological order in memory
			for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
				rows[i], rows[j] = rows[j], rows[i]
			}
			h.records = rows
		}
	}
	return h
}

// Add appends a report to history.
func (h *HistoryStore) Add(report *RunReport) {
	if report == nil {
		return
	}
	if p := h.persist; p != nil && p.Save != nil {
		p.logErr("save_intelligence_history", p.Save(p.DB, report))
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, report)
}

// List returns summaries in reverse chronological order.
func (h *HistoryStore) List() []RunReportSummary {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]RunReportSummary, len(h.records))
	for i, r := range h.records {
		out[len(h.records)-1-i] = RunReportSummary{
			ID:             r.ID,
			DatasetName:    r.DatasetName,
			Model:          r.Model,
			Target:         r.Target,
			Thinking:       r.Thinking,
			Effort:         r.Effort,
			ThinkingMode:   r.ThinkingMode,
			TaskTotal:      r.TaskTotal,
			TaskCompleted:  r.TaskCompleted,
			TaskErrors:     r.TaskErrors,
			ElapsedMs:      r.ElapsedMs,
			StartedAt:      r.StartedAt.Format("2006-01-02 15:04:05"),
			ScoreTotal:     r.ScoreTotal,
			PassRate:       r.PassRate,
			TotalEvaluated: r.TotalEvaluated,
			TotalPassed:    r.TotalPassed,
		}
	}
	return out
}

// Get returns a full report by ID.
func (h *HistoryStore) Get(id string) *RunReport {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, r := range h.records {
		if r.ID == id {
			return r
		}
	}
	return nil
}

// Delete removes a history record by ID.
func (h *HistoryStore) Delete(id string) bool {
	if p := h.persist; p != nil && p.Delete != nil {
		p.logErr("delete_intelligence_history", p.Delete(p.DB, id))
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for i, r := range h.records {
		if r.ID == id {
			h.records = append(h.records[:i], h.records[i+1:]...)
			return true
		}
	}
	return false
}
