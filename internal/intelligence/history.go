package intelligence

import "sync"

// RunReportSummary is a lightweight view of RunReport for list display.
type RunReportSummary struct {
	ID             string `json:"id"`
	DatasetName    string `json:"dataset_name"`
	Model          string `json:"model"`
	Target         string `json:"target"`
	Thinking       bool   `json:"thinking,omitempty"`
	TaskTotal      int    `json:"task_total"`
	TaskCompleted  int    `json:"task_completed"`
	TaskErrors     int    `json:"task_errors"`
	ElapsedMs      int64  `json:"elapsed_ms"`
	StartedAt      string `json:"started_at"`
}

// HistoryStore keeps all benchmark run reports in memory.
type HistoryStore struct {
	mu      sync.RWMutex
	records []*RunReport
}

// NewHistoryStore creates a new HistoryStore.
func NewHistoryStore() *HistoryStore {
	return &HistoryStore{}
}

// Add appends a report to history.
func (h *HistoryStore) Add(report *RunReport) {
	if report == nil {
		return
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
			ID:            r.ID,
			DatasetName:   r.DatasetName,
			Model:         r.Model,
			Target:        r.Target,
			Thinking:      r.Thinking,
			TaskTotal:     r.TaskTotal,
			TaskCompleted: r.TaskCompleted,
			TaskErrors:    r.TaskErrors,
			ElapsedMs:     r.ElapsedMs,
			StartedAt:     r.StartedAt.Format("2006-01-02 15:04:05"),
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
