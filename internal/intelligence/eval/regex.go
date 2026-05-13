package eval

import (
	"regexp"
	"strings"
)

// RegexEvaluator checks if the answer matches a regex pattern.
type RegexEvaluator struct{}

func (RegexEvaluator) Name() string { return "regex" }

func (RegexEvaluator) Evaluate(answer, expected string) Result {
	a := strings.TrimSpace(answer)
	e := strings.TrimSpace(expected)

	if e == "" {
		return Result{Pass: true, Score: 1.0, EvalType: "regex", Reason: "empty pattern"}
	}

	re, err := regexp.Compile(e)
	if err != nil {
		return Result{Pass: false, Score: 0, EvalType: "regex", Reason: "invalid regex: " + err.Error()}
	}

	matches := re.FindAllString(a, -1)
	pass := len(matches) > 0
	score := 0.0
	if pass {
		score = 1.0
	}
	return Result{
		Pass:     pass,
		Score:    score,
		EvalType: "regex",
		Matched:  matches,
	}
}
