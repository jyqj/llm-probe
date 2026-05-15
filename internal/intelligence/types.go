package intelligence

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"time"
)

// Task is one intelligence-test item — generic across all datasets.
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
	Language       string
	Category       string
	TaskIDs        []string
	Limit          int
	ImportantFirst bool
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

// Dataset is the interface all intelligence datasets must implement.
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

// RunRequest is the generic request to run a intelligence.
type RunRequest struct {
	TargetBase     string   `json:"target_base"`
	TargetKey      string   `json:"target_key"`
	Model          string   `json:"model"`
	TaskIDs        []string `json:"task_ids,omitempty"`
	Language       string   `json:"language,omitempty"`
	Category       string   `json:"category,omitempty"`
	Limit          int      `json:"limit,omitempty"`
	ImportantFirst bool     `json:"important_first,omitempty"`
	Thinking       bool     `json:"thinking,omitempty"`
	Effort         string   `json:"effort,omitempty"`
	ThinkingMode   string   `json:"thinking_mode,omitempty"`
	MaxTokens      int      `json:"max_tokens,omitempty"`
	Concurrency    int      `json:"concurrency,omitempty"`
	BaselineID     string   `json:"baseline_id,omitempty"`
}

// BaselineComparison holds the result of comparing a run against a reference baseline.
type BaselineComparison struct {
	BaselineID       string           `json:"baseline_id"`
	BaselineName     string           `json:"baseline_name,omitempty"`
	BaselineScore    float64          `json:"baseline_score"`
	CurrentScore     float64          `json:"current_score"`
	RelativeScore    float64          `json:"relative_score"`
	OverlappingTasks int              `json:"overlapping_tasks"`
	ConsistentTasks  int              `json:"consistent_tasks"`
	TaskComparisons  []TaskComparison `json:"task_comparisons,omitempty"`
}

// TaskComparison records one task's baseline vs current score.
type TaskComparison struct {
	TaskID        string  `json:"task_id"`
	BaselineScore float64 `json:"baseline_score"`
	CurrentScore  float64 `json:"current_score"`
	Deviation     float64 `json:"deviation"`
	Consistent    bool    `json:"consistent"`
}

// RunReport is the complete result of a intelligence-test run.
type RunReport struct {
	ID             string          `json:"id"`
	DatasetName    string          `json:"dataset_name"`
	DatasetVersion string          `json:"dataset_version"`
	Source         string          `json:"source,omitempty"`
	Target         string          `json:"target"`
	Model          string          `json:"model"`
	Thinking       bool            `json:"thinking,omitempty"`
	Effort         string          `json:"effort,omitempty"`
	ThinkingMode   string          `json:"thinking_mode,omitempty"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	StartedAt      time.Time       `json:"started_at"`
	ElapsedMs      int64           `json:"elapsed_ms"`
	TaskTotal      int             `json:"task_total"`
	TaskCompleted  int             `json:"task_completed"`
	TaskErrors     int             `json:"task_errors"`
	Results        []TaskRunResult `json:"results"`
	EvaluationNote string          `json:"evaluation_note"`
	ScoreTotal     *float64        `json:"score_total,omitempty"`
	PassRate       *float64        `json:"pass_rate,omitempty"`
	TotalEvaluated int             `json:"total_evaluated,omitempty"`
	TotalPassed    int             `json:"total_passed,omitempty"`

	BaselineComparison *BaselineComparison `json:"baseline_comparison,omitempty"`
}

// CompareToBaseline computes a baseline comparison between this report and a reference.
func (r *RunReport) CompareToBaseline(baselineID, baselineName string, baseline *RunReport) {
	if baseline == nil || r == nil {
		return
	}

	baseScores := make(map[string]float64)
	for _, tr := range baseline.Results {
		if tr.Score != nil {
			baseScores[tr.Task.TaskID] = *tr.Score
		}
	}

	comp := &BaselineComparison{
		BaselineID:   baselineID,
		BaselineName: baselineName,
	}
	if baseline.ScoreTotal != nil {
		comp.BaselineScore = *baseline.ScoreTotal
	}
	if r.ScoreTotal != nil {
		comp.CurrentScore = *r.ScoreTotal
	}

	for _, tr := range r.Results {
		bs, ok := baseScores[tr.Task.TaskID]
		if !ok {
			continue
		}
		cs := 0.0
		if tr.Score != nil {
			cs = *tr.Score
		} else if tr.Error != "" {
			cs = 0
		} else {
			continue
		}
		comp.OverlappingTasks++
		dev := cs - bs
		consistent := dev >= -0.1 && dev <= 0.1
		comp.TaskComparisons = append(comp.TaskComparisons, TaskComparison{
			TaskID:        tr.Task.TaskID,
			BaselineScore: bs,
			CurrentScore:  cs,
			Deviation:     dev,
			Consistent:    consistent,
		})
	}

	if comp.BaselineScore > 0 {
		comp.RelativeScore = math.Round(comp.CurrentScore/comp.BaselineScore*10000) / 100
	} else if comp.CurrentScore > 0 {
		comp.RelativeScore = 100
	}

	consistentCount := 0
	for _, tc := range comp.TaskComparisons {
		if tc.Consistent {
			consistentCount++
		}
	}
	comp.ConsistentTasks = consistentCount

	r.BaselineComparison = comp
	r.EvaluationNote = fmt.Sprintf(
		"基线对比模式：以官方满血 (%s) 在相同 effort 下的表现为参考 (%.1f 分)，"+
			"当前渠道相对得分 %.1f%% (%d/%d 题一致)。"+
			"判定标准为相对性能而非绝对正确率。",
		baselineName, comp.BaselineScore,
		comp.RelativeScore, consistentCount, comp.OverlappingTasks,
	)
}

// TaskRunResult is the outcome of running one task.
type TaskRunResult struct {
	Index       int          `json:"index"`
	Task        TaskSummary  `json:"task"`
	Answer      string       `json:"answer,omitempty"`
	Error       string       `json:"error,omitempty"`
	ElapsedMs   int64        `json:"elapsed_ms"`
	Rubric      []RubricItem `json:"rubric,omitempty"`
	Pass        *bool        `json:"pass,omitempty"`
	Score       *float64     `json:"score,omitempty"`
	EvalType    string       `json:"eval_type,omitempty"`
	JudgeReason string       `json:"judge_reason,omitempty"`
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

// Runner executes intelligence-test tasks against a target API.
type Runner struct {
	Client *http.Client
}

// NewRunner creates a generic intelligence-test runner.
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
