package eval

import "strings"

// ExactEvaluator checks if the answer exactly matches the expected output.
type ExactEvaluator struct{}

func (ExactEvaluator) Name() string { return "exact" }

func (ExactEvaluator) Evaluate(answer, expected string) Result {
	a := strings.TrimSpace(answer)
	e := strings.TrimSpace(expected)
	pass := strings.EqualFold(a, e)
	score := 0.0
	if pass {
		score = 1.0
	}
	return Result{
		Pass:     pass,
		Score:    score,
		EvalType: "exact",
	}
}
