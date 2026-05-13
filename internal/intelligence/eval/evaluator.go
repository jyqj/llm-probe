package eval

// Evaluator scores a model's answer against the expected output.
type Evaluator interface {
	Name() string
	Evaluate(answer, expected string) Result
}

// Result is the outcome of a single evaluation.
type Result struct {
	Pass        bool     `json:"pass"`
	Score       float64  `json:"score"`
	EvalType    string   `json:"eval_type"`
	Matched     []string `json:"matched,omitempty"`
	Reason      string   `json:"reason,omitempty"`
}

// EvalResult extends a task run result with evaluation data.
type EvalResult struct {
	Pass        bool    `json:"pass"`
	Score       float64 `json:"score"`
	EvalType    string  `json:"eval_type"`
	JudgeReason string  `json:"judge_reason,omitempty"`
}

// CategoryScore aggregates evaluation results by category.
type CategoryScore struct {
	Category string  `json:"category"`
	Total    int     `json:"total"`
	Passed   int     `json:"passed"`
	Score    float64 `json:"score"`
	PassRate float64 `json:"pass_rate"`
}

// ReportScore aggregates evaluation results for a full run.
type ReportScore struct {
	ScoreTotal     float64         `json:"score_total"`
	PassRate       float64         `json:"pass_rate"`
	TotalEvaluated int             `json:"total_evaluated"`
	TotalPassed    int             `json:"total_passed"`
	Categories     []CategoryScore `json:"category_scores,omitempty"`
}

// Get returns an evaluator by type name.
func Get(evalType string) Evaluator {
	switch evalType {
	case "exact":
		return ExactEvaluator{}
	case "contains":
		return ContainsEvaluator{}
	case "regex":
		return RegexEvaluator{}
	default:
		return nil
	}
}
