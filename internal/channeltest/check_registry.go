package channeltest


// Category represents a scoring category for channel checks.
type Category string

const (
	CatFingerprint Category = "fingerprint"
	CatStructural  Category = "structural"
	CatSignature   Category = "signature"
	CatBehavioral  Category = "behavioral"
	CatMultimodal  Category = "multimodal"
)

// CategoryMeta holds static metadata for each category.
type CategoryMeta struct {
	Key    Category
	Label  string
	Weight float64
}

var categoryOrder = []CategoryMeta{
	{CatFingerprint, "LLM 指纹验证", 25},
	{CatStructural, "结构完整性", 25},
	{CatSignature, "签名校验", 20},
	{CatBehavioral, "行为验证", 20},
	{CatMultimodal, "多模态能力", 10},
}

// CheckMeta is the single source of truth for a channel check.
// It centralizes scoring category and the default remediation switch.
type CheckMeta struct {
	Name       string
	Category   Category
	DefaultFix Fix
}

var checkRegistry = map[string]CheckMeta{
	// ── Fingerprint: ID/类型/地理/上游泄漏 ──
	"id_format":              {Name: "id_format", Category: CatFingerprint, DefaultFix: "id_rewrite"},
	"backend_type":           {Name: "backend_type", Category: CatFingerprint, DefaultFix: "id_rewrite"},
	"inference_geo":          {Name: "inference_geo", Category: CatFingerprint, DefaultFix: "force_geo"},
	"stop_details":           {Name: "stop_details", Category: CatFingerprint, DefaultFix: "body_rewrite"},
	"stop_details_structure": {Name: "stop_details_structure", Category: CatFingerprint, DefaultFix: "body_rewrite"},
	"small_output_tokens":    {Name: "small_output_tokens", Category: CatFingerprint, DefaultFix: "body_rewrite"},
	"small_stop_reason":      {Name: "small_stop_reason", Category: CatFingerprint, DefaultFix: "body_rewrite"},
	"container":              {Name: "container", Category: CatFingerprint, DefaultFix: "strip_container"},
	"bedrock_state":          {Name: "bedrock_state", Category: CatFingerprint, DefaultFix: "strip_bedrock"},
	"request_id":             {Name: "request_id", Category: CatFingerprint, DefaultFix: "headers_fake"},
	"x_new_api_version":      {Name: "x_new_api_version", Category: CatFingerprint, DefaultFix: "headers_fake"},
	"cf_ray_format":          {Name: "cf_ray_format", Category: CatFingerprint, DefaultFix: "headers_fake"},
	"cookie_domain":          {Name: "cookie_domain", Category: CatFingerprint, DefaultFix: "headers_fake"},
	"hidden_prompt":          {Name: "hidden_prompt", Category: CatFingerprint},
	"token_budget":           {Name: "token_budget", Category: CatFingerprint},
	"service_tier":           {Name: "service_tier", Category: CatFingerprint},
	"server_header":          {Name: "server_header", Category: CatFingerprint, DefaultFix: "headers_fake"},
	"signature_type_leak":    {Name: "signature_type_leak", Category: CatFingerprint},

	// ── Structural: 响应体/头部/SSE/缓存 ──
	"usage_structure":           {Name: "usage_structure", Category: CatStructural, DefaultFix: "body_rewrite"},
	"field_order":               {Name: "field_order", Category: CatStructural, DefaultFix: "body_rewrite"},
	"model_name":                {Name: "model_name", Category: CatStructural, DefaultFix: "body_rewrite"},
	"stop_reason":               {Name: "stop_reason", Category: CatStructural, DefaultFix: "body_rewrite"},
	"tool_stop_reason":          {Name: "tool_stop_reason", Category: CatStructural, DefaultFix: "body_rewrite"},
	"delta_usage_slim":          {Name: "delta_usage_slim", Category: CatStructural, DefaultFix: "body_rewrite"},
	"message_start_usage":       {Name: "message_start_usage", Category: CatStructural, DefaultFix: "body_rewrite"},
	"message_start_output_zero": {Name: "message_start_output_zero", Category: CatStructural, DefaultFix: "body_rewrite"},
	"nonstream_fields":          {Name: "nonstream_fields", Category: CatStructural, DefaultFix: "body_rewrite"},
	"nonstream_type":            {Name: "nonstream_type", Category: CatStructural, DefaultFix: "body_rewrite"},
	"nonstream_role":            {Name: "nonstream_role", Category: CatStructural, DefaultFix: "body_rewrite"},
	"tool_use_id":               {Name: "tool_use_id", Category: CatStructural, DefaultFix: "id_rewrite"},
	"web_search_result":         {Name: "web_search_result", Category: CatStructural, DefaultFix: "signature_rewrite"},
	"structured_json_valid":     {Name: "structured_json_valid", Category: CatStructural, DefaultFix: "body_rewrite"},
	"structured_schema_match":   {Name: "structured_schema_match", Category: CatStructural, DefaultFix: "body_rewrite"},
	"structured_stop_reason":    {Name: "structured_stop_reason", Category: CatStructural, DefaultFix: "body_rewrite"},
	"headers":                   {Name: "headers", Category: CatStructural, DefaultFix: "headers_fake"},
	"cf_headers":                {Name: "cf_headers", Category: CatStructural, DefaultFix: "headers_fake"},
	"server_timing":             {Name: "server_timing", Category: CatStructural, DefaultFix: "headers_fake"},
	"sse_done":                  {Name: "sse_done", Category: CatStructural, DefaultFix: "strip_done"},
	"sse_event_order":           {Name: "sse_event_order", Category: CatStructural, DefaultFix: "body_rewrite"},
	"sse_tailing":               {Name: "sse_tailing", Category: CatStructural},
	"sse_ping_position":         {Name: "sse_ping_position", Category: CatStructural, DefaultFix: "body_rewrite"},
	"cache_small_probe":         {Name: "cache_small_probe", Category: CatStructural, DefaultFix: "small_probe_zero"},
	"cache_fake":                {Name: "cache_fake", Category: CatStructural, DefaultFix: "cache_fake"},
	"small_ephemeral_zero":      {Name: "small_ephemeral_zero", Category: CatStructural, DefaultFix: "small_probe_zero"},
	"small_cache_zero":          {Name: "small_cache_zero", Category: CatStructural, DefaultFix: "small_probe_zero"},
	"stop_sequence_null":        {Name: "stop_sequence_null", Category: CatStructural, DefaultFix: "body_rewrite"},
	"usage_fields_complete":     {Name: "usage_fields_complete", Category: CatStructural, DefaultFix: "body_rewrite"},
	"cache_creation_complete":   {Name: "cache_creation_complete", Category: CatStructural, DefaultFix: "body_rewrite"},
	"server_tool_type":          {Name: "server_tool_type", Category: CatStructural, DefaultFix: "body_rewrite"},
	"citations_present":         {Name: "citations_present", Category: CatStructural, DefaultFix: "body_rewrite"},
	"body_key_order":            {Name: "body_key_order", Category: CatStructural, DefaultFix: "body_rewrite"},
	"server_tool_usage":         {Name: "server_tool_usage", Category: CatStructural, DefaultFix: "body_rewrite"},

	// ── Signature: 签名/思考块 ──
	"signature":                {Name: "signature", Category: CatSignature, DefaultFix: "signature_rewrite"},
	"signature_length":         {Name: "signature_length", Category: CatSignature, DefaultFix: "signature_rewrite"},
	"thinking_present":         {Name: "thinking_present", Category: CatSignature, DefaultFix: "thinking_inject"},
	"thinking_order":           {Name: "thinking_order", Category: CatSignature, DefaultFix: "thinking_inject"},
	"thinking_display_omitted": {Name: "thinking_display_omitted", Category: CatSignature, DefaultFix: "thinking_inject"},
	"no_thinking_leak":         {Name: "no_thinking_leak", Category: CatSignature, DefaultFix: "body_rewrite"},

	// ── Behavioral: LLM 行为/身份/推理 ──
	"tag_replay":             {Name: "tag_replay", Category: CatBehavioral},
	"identity_response":      {Name: "identity_response", Category: CatBehavioral},
	"identity_no_leak":       {Name: "identity_no_leak", Category: CatBehavioral},
	"identity_platform":      {Name: "identity_platform", Category: CatBehavioral},
	"poison_answer":          {Name: "poison_answer", Category: CatBehavioral},
	"logic_answer":           {Name: "logic_answer", Category: CatBehavioral},
	"tool_forced_compliance": {Name: "tool_forced_compliance", Category: CatBehavioral},

	// ── Multimodal: 多模态能力 ──
	"image_ocr":   {Name: "image_ocr", Category: CatMultimodal},
	"pdf_extract": {Name: "pdf_extract", Category: CatMultimodal},
}

// checkCategoryMap maps check names to their category.
// Kept as package-level map for existing tests and callers; derived from checkRegistry.
var checkCategoryMap = buildCheckCategoryMap()

func buildCheckCategoryMap() map[string]Category {
	out := make(map[string]Category, len(checkRegistry))
	for name, meta := range checkRegistry {
		out[name] = meta.Category
	}
	return out
}

func defaultFixForCheck(name string) Fix {
	if meta, ok := checkRegistry[name]; ok {
		return meta.DefaultFix
	}
	return ""
}

func failCheck(name, detail string) CheckResult {
	return CheckResult{Name: name, Pass: false, Detail: detail, Fix: defaultFixForCheck(name)}
}

func failChecks(names []string, detail string) []CheckResult {
	out := make([]CheckResult, 0, len(names))
	for _, name := range names {
		out = append(out, failCheck(name, detail))
	}
	return out
}

func batchFail(names []string) []CheckResult {
	return failChecks(names, "could not parse SSE response")
}
