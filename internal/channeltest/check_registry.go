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
// It centralizes scoring category, display label, and the default remediation switch.
type CheckMeta struct {
	Name       string
	Label      string // human-readable Chinese description
	Category   Category
	DefaultFix Fix
}

var checkRegistry = map[string]CheckMeta{
	// ── Fingerprint: ID/类型/地理/上游泄漏 ──
	"id_format":              {Name: "id_format", Label: "消息 ID 格式验证", Category: CatFingerprint, DefaultFix: "id_rewrite"},
	"backend_type":           {Name: "backend_type", Label: "后端类型检测", Category: CatFingerprint, DefaultFix: "id_rewrite"},
	"inference_geo":          {Name: "inference_geo", Label: "推理地理位置", Category: CatFingerprint, DefaultFix: "force_geo"},
	"stop_details":           {Name: "stop_details", Label: "stop_details 字段存在性", Category: CatFingerprint, DefaultFix: "body_rewrite"},
	"stop_details_structure": {Name: "stop_details_structure", Label: "stop_details 结构一致性", Category: CatFingerprint, DefaultFix: "body_rewrite"},
	"small_output_tokens":    {Name: "small_output_tokens", Label: "最小 token 计数验证", Category: CatFingerprint, DefaultFix: "body_rewrite"},
	"small_stop_reason":      {Name: "small_stop_reason", Label: "小请求 stop_reason 验证", Category: CatFingerprint, DefaultFix: "body_rewrite"},
	"container":              {Name: "container", Label: "容器字段泄漏检测", Category: CatFingerprint, DefaultFix: "strip_container"},
	"bedrock_state":          {Name: "bedrock_state", Label: "Bedrock 状态泄漏", Category: CatFingerprint, DefaultFix: "strip_bedrock"},
	"request_id":             {Name: "request_id", Label: "Request-Id 格式验证", Category: CatFingerprint, DefaultFix: "headers_fake"},
	"x_new_api_version":      {Name: "x_new_api_version", Label: "X-New-Api-Version 泄漏", Category: CatFingerprint, DefaultFix: "headers_fake"},
	"cf_ray_format":          {Name: "cf_ray_format", Label: "Cf-Ray 格式验证", Category: CatFingerprint, DefaultFix: "headers_fake"},
	"cookie_domain":          {Name: "cookie_domain", Label: "Cookie 域名验证", Category: CatFingerprint, DefaultFix: "headers_fake"},
	"hidden_prompt":          {Name: "hidden_prompt", Label: "隐藏 system prompt 检测", Category: CatFingerprint},
	"token_budget":           {Name: "token_budget", Label: "Token 预算一致性", Category: CatFingerprint},
	"service_tier":           {Name: "service_tier", Label: "service_tier 字段存在性", Category: CatFingerprint},
	"server_header":          {Name: "server_header", Label: "Server 响应头验证", Category: CatFingerprint, DefaultFix: "headers_fake"},
	"signature_type_leak":    {Name: "signature_type_leak", Label: "signature_type 字段泄漏", Category: CatFingerprint},

	// ── Structural: 响应体/头部/SSE/缓存 ──
	"usage_structure":           {Name: "usage_structure", Label: "usage 嵌套结构验证", Category: CatStructural, DefaultFix: "body_rewrite"},
	"field_order":               {Name: "field_order", Label: "JSON 字段顺序验证", Category: CatStructural, DefaultFix: "body_rewrite"},
	"model_name":                {Name: "model_name", Label: "模型自报匹配", Category: CatStructural, DefaultFix: "body_rewrite"},
	"stop_reason":               {Name: "stop_reason", Label: "stop_reason 合法性", Category: CatStructural, DefaultFix: "body_rewrite"},
	"tool_stop_reason":          {Name: "tool_stop_reason", Label: "工具调用 stop_reason", Category: CatStructural, DefaultFix: "body_rewrite"},
	"delta_usage_slim":          {Name: "delta_usage_slim", Label: "message_delta usage 精简格式", Category: CatStructural, DefaultFix: "body_rewrite"},
	"message_start_usage":       {Name: "message_start_usage", Label: "message_start usage 验证", Category: CatStructural, DefaultFix: "body_rewrite"},
	"message_start_output_zero": {Name: "message_start_output_zero", Label: "message_start output_tokens=0", Category: CatStructural, DefaultFix: "body_rewrite"},
	"nonstream_fields":          {Name: "nonstream_fields", Label: "非流式必需字段完整性", Category: CatStructural, DefaultFix: "body_rewrite"},
	"nonstream_type":            {Name: "nonstream_type", Label: "非流式 type=message", Category: CatStructural, DefaultFix: "body_rewrite"},
	"nonstream_role":            {Name: "nonstream_role", Label: "非流式 role=assistant", Category: CatStructural, DefaultFix: "body_rewrite"},
	"tool_use_id":               {Name: "tool_use_id", Label: "工具调用 ID 格式", Category: CatStructural, DefaultFix: "id_rewrite"},
	"web_search_result":         {Name: "web_search_result", Label: "web_search 结果结构", Category: CatStructural, DefaultFix: "signature_rewrite"},
	"structured_json_valid":     {Name: "structured_json_valid", Label: "JSON schema 结构化输出", Category: CatStructural, DefaultFix: "body_rewrite"},
	"structured_schema_match":   {Name: "structured_schema_match", Label: "结构化输出 schema 匹配", Category: CatStructural, DefaultFix: "body_rewrite"},
	"structured_stop_reason":    {Name: "structured_stop_reason", Label: "结构化输出 stop_reason", Category: CatStructural, DefaultFix: "body_rewrite"},
	"headers":                   {Name: "headers", Label: "Anthropic 速率限制头验证", Category: CatStructural, DefaultFix: "headers_fake"},
	"cf_headers":                {Name: "cf_headers", Label: "Cloudflare 头验证", Category: CatStructural, DefaultFix: "headers_fake"},
	"server_timing":             {Name: "server_timing", Label: "Server-Timing 头验证", Category: CatStructural, DefaultFix: "headers_fake"},
	"sse_done":                  {Name: "sse_done", Label: "[DONE] 哨兵检测", Category: CatStructural, DefaultFix: "strip_done"},
	"sse_event_order":           {Name: "sse_event_order", Label: "SSE 事件顺序验证", Category: CatStructural, DefaultFix: "body_rewrite"},
	"sse_tailing":               {Name: "sse_tailing", Label: "SSE 尾部换行格式", Category: CatStructural},
	"sse_ping_position":         {Name: "sse_ping_position", Label: "SSE ping 位置验证", Category: CatStructural, DefaultFix: "body_rewrite"},
	"cache_small_probe":         {Name: "cache_small_probe", Label: "小请求 cache 归零验证", Category: CatStructural, DefaultFix: "small_probe_zero"},
	"cache_fake":                {Name: "cache_fake", Label: "cache 伪造检测", Category: CatStructural, DefaultFix: "cache_fake"},
	"small_ephemeral_zero":      {Name: "small_ephemeral_zero", Label: "小请求 ephemeral 归零", Category: CatStructural, DefaultFix: "small_probe_zero"},
	"small_cache_zero":          {Name: "small_cache_zero", Label: "小请求 cache 字段归零", Category: CatStructural, DefaultFix: "small_probe_zero"},
	"stop_sequence_null":        {Name: "stop_sequence_null", Label: "stop_sequence 字段为 null", Category: CatStructural, DefaultFix: "body_rewrite"},
	"usage_fields_complete":     {Name: "usage_fields_complete", Label: "usage 字段完整性", Category: CatStructural, DefaultFix: "body_rewrite"},
	"cache_creation_complete":   {Name: "cache_creation_complete", Label: "cache_creation 完整性", Category: CatStructural, DefaultFix: "body_rewrite"},
	"server_tool_type":          {Name: "server_tool_type", Label: "server_tool_use 类型验证", Category: CatStructural, DefaultFix: "body_rewrite"},
	"citations_present":         {Name: "citations_present", Label: "web_search 引用验证", Category: CatStructural, DefaultFix: "body_rewrite"},
	"body_key_order":            {Name: "body_key_order", Label: "响应体字段排序", Category: CatStructural, DefaultFix: "body_rewrite"},
	"server_tool_usage":         {Name: "server_tool_usage", Label: "server_tool_use 统计验证", Category: CatStructural, DefaultFix: "body_rewrite"},

	// ── Signature: 签名/思考块 ──
	"signature":                {Name: "signature", Label: "thinking signature 校验", Category: CatSignature, DefaultFix: "signature_rewrite"},
	"signature_length":         {Name: "signature_length", Label: "signature 长度验证", Category: CatSignature, DefaultFix: "signature_rewrite"},
	"thinking_present":         {Name: "thinking_present", Label: "thinking 块存在性", Category: CatSignature, DefaultFix: "thinking_inject"},
	"thinking_order":           {Name: "thinking_order", Label: "thinking 块排序验证", Category: CatSignature, DefaultFix: "thinking_inject"},
	"thinking_display_omitted": {Name: "thinking_display_omitted", Label: "thinking display=omitted 模式", Category: CatSignature, DefaultFix: "thinking_inject"},
	"no_thinking_leak":         {Name: "no_thinking_leak", Label: "未请求 thinking 无泄漏", Category: CatSignature, DefaultFix: "body_rewrite"},

	// ── Behavioral: LLM 行为/身份/推理 ──
	"tag_replay":             {Name: "tag_replay", Label: "随机 tag 回显验证", Category: CatBehavioral},
	"identity_response":      {Name: "identity_response", Label: "身份自述验证", Category: CatBehavioral},
	"identity_no_leak":       {Name: "identity_no_leak", Label: "内部代号无泄漏", Category: CatBehavioral},
	"identity_platform":      {Name: "identity_platform", Label: "平台自述验证", Category: CatBehavioral},
	"poison_answer":          {Name: "poison_answer", Label: "毒药推理题验证", Category: CatBehavioral},
	"logic_answer":           {Name: "logic_answer", Label: "逻辑推理题验证", Category: CatBehavioral},
	"tool_forced_compliance": {Name: "tool_forced_compliance", Label: "强制工具调用合规性", Category: CatBehavioral},
	"magic_refusal":          {Name: "magic_refusal", Label: "拒答字符串 refusal", Category: CatBehavioral},

	// ── Multimodal: 多模态能力 ──
	"image_ocr":   {Name: "image_ocr", Label: "图片 OCR 识别", Category: CatMultimodal},
	"pdf_extract": {Name: "pdf_extract", Label: "PDF 文本提取", Category: CatMultimodal},

	// ── Effort-level thinking ──
	"effort_high_thinking":   {Name: "effort_high_thinking", Label: "effortHigh 必须 thinking 块", Category: CatSignature},
	"effort_high_signature":  {Name: "effort_high_signature", Label: "effortHigh signature 有效", Category: CatSignature, DefaultFix: "signature_rewrite"},
	"effort_medium_no_think": {Name: "effort_medium_no_think", Label: "effortMedium 应抑制 thinking", Category: CatSignature},
	"effort_low_no_think":    {Name: "effort_low_no_think", Label: "effortLow 应跳过 thinking", Category: CatSignature},
	"effort_max_thinking":    {Name: "effort_max_thinking", Label: "effortMax 必须 thinking 块", Category: CatSignature},
	"effort_xhigh_thinking":  {Name: "effort_xhigh_thinking", Label: "effortXHigh 必须 thinking 块 (仅 Opus 4.7)", Category: CatSignature},

	// ── Signature validation ──
	"signature_empty_rejected": {Name: "signature_empty_rejected", Label: "空 signature 拒绝验证", Category: CatSignature},

	// ── Bash tool ──
	"bash_stop_reason":   {Name: "bash_stop_reason", Label: "bash tool stop_reason 验证", Category: CatStructural, DefaultFix: "body_rewrite"},
	"bash_tool_name":     {Name: "bash_tool_name", Label: "bash tool name 验证", Category: CatStructural, DefaultFix: "body_rewrite"},
	"bash_tool_rejected": {Name: "bash_tool_rejected", Label: "非法 bash tool 拒绝验证", Category: CatStructural},

	// ── Minimal token billing ──
	"minimal_input_tokens":  {Name: "minimal_input_tokens", Label: "最小 token 计费核对 (input)", Category: CatFingerprint},
	"minimal_output_tokens": {Name: "minimal_output_tokens", Label: "最小 token 计费核对 (output)", Category: CatFingerprint},
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

func labelForCheck(name string) string {
	if meta, ok := checkRegistry[name]; ok {
		return meta.Label
	}
	return ""
}

func failCheck(name, detail string) CheckResult {
	return CheckResult{Name: name, Label: labelForCheck(name), Pass: false, Detail: detail, Fix: defaultFixForCheck(name)}
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
