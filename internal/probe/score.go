package probe

// Category represents a scoring category for probe checks.
type Category string

const (
	CatCritical      Category = "critical"
	CatStructural    Category = "structural"
	CatHeaders       Category = "headers"
	CatThinking      Category = "thinking"
	CatSSE           Category = "sse"
	CatCache         Category = "cache"
	CatBehavioral    Category = "behavioral"
	CatContamination Category = "contamination"
)

// CategoryMeta holds static metadata for each category.
type CategoryMeta struct {
	Key    Category
	Label  string
	Weight float64
}

var categoryOrder = []CategoryMeta{
	{CatCritical, "反伪造核心", 30},
	{CatStructural, "响应体结构", 20},
	{CatHeaders, "HTTP 头部", 12},
	{CatThinking, "思考与签名", 12},
	{CatSSE, "流格式", 8},
	{CatCache, "缓存行为", 8},
	{CatBehavioral, "LLM 行为验证", 7},
	{CatContamination, "上游泄漏", 3},
}

// checkCategoryMap maps check names to their category.
var checkCategoryMap = map[string]Category{
	// Critical
	"id_format":                CatCritical,
	"backend_type":             CatCritical,
	"inference_geo":            CatCritical,
	"stop_details":             CatCritical,
	"stop_details_structure":   CatCritical,
	"small_output_tokens":      CatCritical,
	"small_stop_reason":        CatCritical,

	// Structural
	"usage_structure":         CatStructural,
	"field_order":             CatStructural,
	"model_name":              CatStructural,
	"stop_reason":             CatStructural,
	"tool_stop_reason":        CatStructural,
	"delta_usage_slim":        CatStructural,
	"message_start_usage":     CatStructural,
	"nonstream_fields":        CatStructural,
	"nonstream_type":          CatStructural,
	"nonstream_role":          CatStructural,
	"tool_use_id":             CatStructural,
	"web_search_result":       CatStructural,
	"structured_json_valid":   CatStructural,
	"structured_schema_match": CatStructural,
	"structured_stop_reason":  CatStructural,

	// Headers
	"headers":             CatHeaders,
	"request_id":          CatHeaders,
	"x_new_api_version":   CatHeaders,
	"cf_headers":          CatHeaders,
	"server_timing":       CatHeaders,

	// Thinking
	"signature":                CatThinking,
	"thinking_present":         CatThinking,
	"thinking_order":           CatThinking,
	"thinking_display_omitted": CatThinking,
	"no_thinking_leak":         CatThinking,

	// SSE
	"sse_done":        CatSSE,
	"sse_event_order": CatSSE,
	"sse_tailing":     CatSSE,

	// Cache
	"cache_small_probe":    CatCache,
	"cache_fake":           CatCache,
	"small_ephemeral_zero": CatCache,
	"small_cache_zero":     CatCache,

	// Behavioral
	"tag_replay":         CatBehavioral,
	"identity_response":  CatBehavioral,
	"poison_answer":      CatBehavioral,
	"logic_answer":       CatBehavioral,
	"image_ocr":          CatBehavioral,
	"pdf_extract":        CatBehavioral,

	// Contamination
	"container":      CatContamination,
	"bedrock_state":  CatContamination,
}

// CategoryScore represents the score for a single category.
type CategoryScore struct {
	Key        Category      `json:"key"`
	Label      string        `json:"label"`
	Passed     int           `json:"passed"`
	Total      int           `json:"total"`
	Weight     float64       `json:"weight"`
	Score      float64       `json:"score"`
	Percentage float64       `json:"percentage"`
	Checks     []CheckResult `json:"checks"`
}

// ScoreReport is the full scoring result.
type ScoreReport struct {
	TotalScore      float64         `json:"total_score"`
	Grade           string          `json:"grade"`
	GradeColor      string          `json:"grade_color"`
	Verdict         string          `json:"verdict"`
	VerdictLabel    string          `json:"verdict_label"`
	VerdictColor    string          `json:"verdict_color"`
	Categories      []CategoryScore `json:"categories"`
	CriticalPenalty float64         `json:"critical_penalty"`
	Mode            string          `json:"mode"`
	ChecksTotal     int             `json:"checks_total"`
	ChecksPassed    int             `json:"checks_passed"`
}

// CalculateScore computes a weighted score from check results.
func CalculateScore(checks []CheckResult, mode string) *ScoreReport {
	// Step 1: Deduplicate checks per category — same name → worst result
	type dedupKey struct {
		cat  Category
		name string
	}
	deduped := map[dedupKey]CheckResult{}
	for _, c := range checks {
		cat, ok := checkCategoryMap[c.Name]
		if !ok {
			continue
		}
		dk := dedupKey{cat, c.Name}
		if existing, exists := deduped[dk]; !exists {
			deduped[dk] = c
		} else if existing.Pass && !c.Pass {
			// Take the failing one (worst result)
			deduped[dk] = c
		}
	}

	// Step 2: Group into categories
	catChecks := map[Category][]CheckResult{}
	for dk, c := range deduped {
		catChecks[dk.cat] = append(catChecks[dk.cat], c)
	}

	// Step 3: Calculate per-category scores
	var categories []CategoryScore
	var totalWeightedScore float64
	var totalActiveWeight float64
	criticalFails := 0
	backendTypeFailed := false

	for _, meta := range categoryOrder {
		cks, exists := catChecks[meta.Key]
		if !exists || len(cks) == 0 {
			continue
		}

		passed := 0
		for _, c := range cks {
			if c.Pass {
				passed++
			} else if meta.Key == CatCritical {
				criticalFails++
				if c.Name == "backend_type" {
					backendTypeFailed = true
				}
			}
		}

		total := len(cks)
		pct := 0.0
		if total > 0 {
			pct = float64(passed) / float64(total) * 100
		}
		catScore := pct / 100 * meta.Weight

		categories = append(categories, CategoryScore{
			Key:        meta.Key,
			Label:      meta.Label,
			Passed:     passed,
			Total:      total,
			Weight:     meta.Weight,
			Score:      catScore,
			Percentage: pct,
			Checks:     cks,
		})

		totalWeightedScore += catScore
		totalActiveWeight += meta.Weight
	}

	// Step 4: Normalize score to 0-100
	score := 0.0
	if totalActiveWeight > 0 {
		score = totalWeightedScore / totalActiveWeight * 100
	}

	// Step 5: Critical penalty
	var penalty float64
	if criticalFails >= 3 {
		penalty = 10
		score -= penalty
	}
	if score < 0 {
		score = 0
	}

	// Step 6: Grade
	grade, gradeColor := mapGrade(score)

	// Step 7: Verdict
	verdict, verdictLabel, verdictColor := mapVerdict(score, criticalFails, backendTypeFailed)

	// Totals
	totalChecks := 0
	totalPassed := 0
	for _, cs := range categories {
		totalChecks += cs.Total
		totalPassed += cs.Passed
	}

	return &ScoreReport{
		TotalScore:      round2(score),
		Grade:           grade,
		GradeColor:      gradeColor,
		Verdict:         verdict,
		VerdictLabel:    verdictLabel,
		VerdictColor:    verdictColor,
		Categories:      categories,
		CriticalPenalty: penalty,
		Mode:            mode,
		ChecksTotal:     totalChecks,
		ChecksPassed:    totalPassed,
	}
}

func mapGrade(score float64) (string, string) {
	switch {
	case score >= 95:
		return "A+", "green"
	case score >= 90:
		return "A", "green"
	case score >= 80:
		return "B", "blue"
	case score >= 70:
		return "C", "yellow"
	case score >= 50:
		return "D", "orange"
	default:
		return "F", "red"
	}
}

func mapVerdict(score float64, criticalFails int, backendTypeFailed bool) (string, string, string) {
	if backendTypeFailed {
		return "non_official", "非官方后端", "red"
	}
	criticalAllPass := criticalFails == 0
	switch {
	case score >= 95 && criticalAllPass:
		return "official", "官方 API", "green"
	case score >= 80 && criticalAllPass:
		return "good", "伪装良好", "blue"
	case score >= 80:
		return "suspected", "疑似伪造", "orange"
	case score >= 50:
		return "poor", "伪装较差", "yellow"
	default:
		return "fake", "明确假冒", "red"
	}
}

func round2(f float64) float64 {
	return float64(int(f*100+0.5)) / 100
}
