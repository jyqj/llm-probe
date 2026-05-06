package benchmark

import (
	"context"
	"net/http"
	"time"
)

// Task is one benchmark item — generic across all datasets.
type Task struct {
	TaskID          string            `json:"task_id"`
	Prompt          string            `json:"prompt"`
	ReferenceAnswer string            `json:"reference_answer,omitempty"`
	Language        string            `json:"language,omitempty"`
	Category        string            `json:"category,omitempty"`
	Rubric          []RubricItem      `json:"rubric,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"` // extensible key-value pairs
}

// TaskSummary is the safe/list view of a task.
type TaskSummary struct {
	TaskID   string            `json:"task_id"`
	Prompt   string            `json:"prompt"`
	Language string            `json:"language,omitempty"`
	Category string            `json:"category,omitempty"`
	Rubric   []RubricItem      `json:"rubric,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// RubricItem is a generic evaluation criterion.
type RubricItem struct {
	ID          string           `json:"id"`
	Title       string           `json:"title"`
	Annotations RubricAnnotation `json:"annotations,omitempty"`
}

type RubricAnnotation struct {
	Type       string `json:"type,omitempty"`
	Importance string `json:"importance,omitempty"`
}

// Summary returns a TaskSummary, optionally including rubric.
func (t Task) Summary(includeRubric bool) TaskSummary {
	s := TaskSummary{
		TaskID:   t.TaskID,
		Prompt:   t.Prompt,
		Language: t.Language,
		Category: t.Category,
		Metadata: t.Metadata,
	}
	if includeRubric {
		s.Rubric = t.Rubric
	}
	return s
}

// Filter controls which tasks to select.
type Filter struct {
	Language string
	Category string
	TaskIDs  []string
	Limit    int
}

// Stats describes a dataset's composition.
type Stats struct {
	Name       string         `json:"name"`
	Version    string         `json:"version"`
	Source     string         `json:"source,omitempty"`
	TotalTasks int            `json:"total_tasks"`
	Languages  map[string]int `json:"languages"`
	Categories map[string]int `json:"categories"`
}

// Dataset is the interface all benchmark datasets must implement.
type Dataset interface {
	// Identity
	Name() string
	Version() string
	Source() string
	Stats() Stats

	// Query
	Tasks() []Task
	Filter(f Filter) []Task
	Find(taskID string) (Task, bool)
}

// RunRequest is the generic request to run a benchmark.
type RunRequest struct {
	TargetBase  string   `json:"target_base"`
	TargetKey   string   `json:"target_key"`
	Model       string   `json:"model"`
	TaskIDs     []string `json:"task_ids,omitempty"`
	Language    string   `json:"language,omitempty"`
	Category    string   `json:"category,omitempty"`
	Limit       int      `json:"limit,omitempty"`
	MaxTokens   int      `json:"max_tokens,omitempty"`
	Concurrency int      `json:"concurrency,omitempty"`
}

// RunReport is the complete result of a benchmark run.
type RunReport struct {
	DatasetName    string          `json:"dataset_name"`
	DatasetVersion string          `json:"dataset_version"`
	Source         string          `json:"source,omitempty"`
	Target         string          `json:"target"`
	Model          string          `json:"model"`
	StartedAt      time.Time       `json:"started_at"`
	ElapsedMs      int64           `json:"elapsed_ms"`
	TaskTotal      int             `json:"task_total"`
	TaskCompleted  int             `json:"task_completed"`
	TaskErrors     int             `json:"task_errors"`
	Results        []TaskRunResult `json:"results"`
	EvaluationNote string          `json:"evaluation_note"`
}

// TaskRunResult is the outcome of running one task.
type TaskRunResult struct {
	Index     int          `json:"index"`
	Task      TaskSummary  `json:"task"`
	Answer    string       `json:"answer,omitempty"`
	Error     string       `json:"error,omitempty"`
	ElapsedMs int64        `json:"elapsed_ms"`
	Rubric    []RubricItem `json:"rubric,omitempty"`
}

// StreamEvent is sent for each completed task during streaming.
type StreamEvent struct {
	Type      string         `json:"type"` // "progress" | "complete" | "error"
	Index     int            `json:"index"`
	Total     int            `json:"total"`
	Completed int            `json:"completed"`
	Errors    int            `json:"errors"`
	Result    *TaskRunResult `json:"result,omitempty"`
	Report    *RunReport     `json:"report,omitempty"`
	ErrorMsg  string         `json:"error_msg,omitempty"`
}

// Runner executes benchmark tasks against a target API.
type Runner struct {
	Client *http.Client
}

// NewRunner creates a generic benchmark runner.
func NewRunner(client *http.Client) *Runner {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Minute}
	}
	return &Runner{Client: client}
}

// Run executes tasks concurrently and returns a complete report.
func (r *Runner) Run(ctx context.Context, ds Dataset, req RunRequest) (*RunReport, error) {
	return r.run(ctx, ds, req, nil)
}

// RunStream executes tasks concurrently, calling onEvent for each completion.
func (r *Runner) RunStream(ctx context.Context, ds Dataset, req RunRequest, onEvent func(StreamEvent)) (*RunReport, error) {
	return r.run(ctx, ds, req, onEvent)
}
