package eval

// Aggregate computes a ReportScore from individual eval results.
func Aggregate(results []EvalResult, categories map[int]string) ReportScore {
	rs := ReportScore{}
	catMap := make(map[string]*CategoryScore)
	totalScore := 0.0

	for i, r := range results {
		rs.TotalEvaluated++
		totalScore += r.Score
		if r.Pass {
			rs.TotalPassed++
		}

		cat := ""
		if categories != nil {
			cat = categories[i]
		}
		if cat == "" {
			cat = "uncategorized"
		}
		cs, ok := catMap[cat]
		if !ok {
			cs = &CategoryScore{Category: cat}
			catMap[cat] = cs
		}
		cs.Total++
		cs.Score += r.Score
		if r.Pass {
			cs.Passed++
		}
	}

	if rs.TotalEvaluated > 0 {
		rs.ScoreTotal = totalScore / float64(rs.TotalEvaluated) * 100
		rs.PassRate = float64(rs.TotalPassed) / float64(rs.TotalEvaluated) * 100
	}

	for _, cs := range catMap {
		if cs.Total > 0 {
			cs.Score = cs.Score / float64(cs.Total) * 100
			cs.PassRate = float64(cs.Passed) / float64(cs.Total) * 100
		}
		rs.Categories = append(rs.Categories, *cs)
	}

	return rs
}

// EvaluateTaskResult evaluates a single task answer using the appropriate evaluator.
func EvaluateTaskResult(answer, expected, evalType string) EvalResult {
	if expected == "" {
		return EvalResult{EvalType: "manual"}
	}
	if evalType == "" {
		evalType = "contains"
	}
	ev := Get(evalType)
	if ev == nil {
		return EvalResult{EvalType: evalType, JudgeReason: "unknown evaluator type"}
	}
	r := ev.Evaluate(answer, expected)
	return EvalResult{
		Pass:     r.Pass,
		Score:    r.Score,
		EvalType: r.EvalType,
	}
}
