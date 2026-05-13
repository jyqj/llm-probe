package eval

import "strings"

// ContainsEvaluator checks if the answer contains the expected substring.
type ContainsEvaluator struct{}

func (ContainsEvaluator) Name() string { return "contains" }

func (ContainsEvaluator) Evaluate(answer, expected string) Result {
	a := strings.ToLower(strings.TrimSpace(answer))
	e := strings.ToLower(strings.TrimSpace(expected))

	if e == "" {
		return Result{Pass: true, Score: 1.0, EvalType: "contains", Reason: "empty expected"}
	}

	parts := strings.Split(e, "|")
	var matched []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" && strings.Contains(a, p) {
			matched = append(matched, p)
		}
	}

	pass := len(matched) > 0
	score := float64(len(matched)) / float64(len(parts))
	return Result{
		Pass:     pass,
		Score:    score,
		EvalType: "contains",
		Matched:  matched,
	}
}
