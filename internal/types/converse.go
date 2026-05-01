package types

// ========== Bedrock Converse API Types ==========

// ConverseRequest is the Bedrock Converse API request body.
// POST /model/{modelId}/converse
// POST /model/{modelId}/converse-stream
type ConverseRequest struct {
	Messages                    []ConverseMessage       `json:"messages"`
	System                      []ConverseSystemBlock   `json:"system,omitempty"`
	InferenceConfig             *InferenceConfig        `json:"inferenceConfig,omitempty"`
	ToolConfig                  *ToolConfig             `json:"toolConfig,omitempty"`
	GuardrailConfig             *GuardrailConfig        `json:"guardrailConfig,omitempty"`
	AdditionalModelRequestFields map[string]any          `json:"additionalModelRequestFields,omitempty"`
	AdditionalModelResponseFieldPaths []string           `json:"additionalModelResponseFieldPaths,omitempty"`
	PromptVariables             map[string]PromptVariable `json:"promptVariables,omitempty"`
	RequestMetadata             *RequestMetadata        `json:"requestMetadata,omitempty"`
	PerformanceConfig           *PerformanceConfig      `json:"performanceConfig,omitempty"`
}

// ConverseMessage represents a message in Converse API.
type ConverseMessage struct {
	Role    string                 `json:"role"` // "user" or "assistant"
	Content []ConverseContentBlock `json:"content"`
}

// ConverseContentBlock is the union type for Converse content blocks.
type ConverseContentBlock struct {
	// text
	Text *string `json:"text,omitempty"`

	// image
	Image *ConverseImageBlock `json:"image,omitempty"`

	// document
	Document *ConverseDocumentBlock `json:"document,omitempty"`

	// video
	Video *ConverseVideoBlock `json:"video,omitempty"`

	// toolUse
	ToolUse *ConverseToolUseBlock `json:"toolUse,omitempty"`

	// toolResult
	ToolResult *ConverseToolResultBlock `json:"toolResult,omitempty"`

	// guardContent
	GuardContent *GuardContentBlock `json:"guardContent,omitempty"`

	// reasoningContent
	ReasoningContent *ReasoningContentBlock `json:"reasoningContent,omitempty"`

	// cachePoint
	CachePoint *CachePointBlock `json:"cachePoint,omitempty"`
}

// ConverseImageBlock for image content.
type ConverseImageBlock struct {
	Format string          `json:"format"` // "png", "jpeg", "gif", "webp"
	Source *ConverseSource `json:"source"`
}

// ConverseSource for binary data source.
type ConverseSource struct {
	Bytes string `json:"bytes,omitempty"` // base64-encoded
}

// ConverseDocumentBlock for document content.
type ConverseDocumentBlock struct {
	Format string          `json:"format"` // "pdf", "csv", "doc", "docx", "xls", "xlsx", "html", "txt", "md"
	Name   string          `json:"name"`
	Source *ConverseSource `json:"source"`
}

// ConverseVideoBlock for video content.
type ConverseVideoBlock struct {
	Format string          `json:"format"` // "mkv", "mov", "mp4", "webm", etc.
	Source *ConverseSource `json:"source"`
}

// ConverseToolUseBlock for tool use.
type ConverseToolUseBlock struct {
	ToolUseID string `json:"toolUseId"`
	Name      string `json:"name"`
	Input     any    `json:"input"` // JSON document
}

// ConverseToolResultBlock for tool results.
type ConverseToolResultBlock struct {
	ToolUseID string                 `json:"toolUseId"`
	Content   []ConverseContentBlock `json:"content"`
	Status    string                 `json:"status,omitempty"` // "success", "error"
}

// GuardContentBlock for guardrail content.
type GuardContentBlock struct {
	Text *GuardTextBlock `json:"text,omitempty"`
}

// GuardTextBlock for guardrail text.
type GuardTextBlock struct {
	Text      string   `json:"text"`
	Qualifiers []string `json:"qualifiers,omitempty"` // "grounding_source", "query", "guard_content"
}

// ReasoningContentBlock for chain-of-thought reasoning.
type ReasoningContentBlock struct {
	ReasoningText    *ReasoningTextBlock `json:"reasoningText,omitempty"`
	RedactedContent  *string             `json:"redactedContent,omitempty"` // base64
}

// ReasoningTextBlock contains the reasoning text and signature.
type ReasoningTextBlock struct {
	Text      string `json:"text"`
	Signature string `json:"signature,omitempty"`
}

// CachePointBlock for prompt caching.
type CachePointBlock struct {
	Type string `json:"type"` // "default"
}

// ConverseSystemBlock for system content in Converse API.
type ConverseSystemBlock struct {
	Text       *string          `json:"text,omitempty"`
	Image      *ConverseImageBlock `json:"image,omitempty"`
	Document   *ConverseDocumentBlock `json:"document,omitempty"`
	Video      *ConverseVideoBlock `json:"video,omitempty"`
	GuardContent *GuardContentBlock `json:"guardContent,omitempty"`
	CachePoint *CachePointBlock   `json:"cachePoint,omitempty"`
}

// InferenceConfig for model inference parameters.
type InferenceConfig struct {
	MaxTokens     *int      `json:"maxTokens,omitempty"`
	Temperature   *float64  `json:"temperature,omitempty"` // 0.0 - 1.0
	TopP          *float64  `json:"topP,omitempty"`        // 0.0 - 1.0
	StopSequences []string  `json:"stopSequences,omitempty"` // max 4
}

// ToolConfig for tool configuration.
type ToolConfig struct {
	Tools      []ConverseToolDef `json:"tools"`
	ToolChoice *ConverseToolChoice `json:"toolChoice,omitempty"`
}

// ConverseToolDef defines a tool in Converse API.
type ConverseToolDef struct {
	ToolSpec *ToolSpec `json:"toolSpec,omitempty"`
}

// ToolSpec is the tool specification.
type ToolSpec struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	InputSchema *ToolInputSchema `json:"inputSchema"`
}

// ToolInputSchema wraps the JSON schema for tool input.
type ToolInputSchema struct {
	JSON any `json:"json"` // JSON Schema object
}

// ConverseToolChoice for tool selection.
type ConverseToolChoice struct {
	Auto *struct{} `json:"auto,omitempty"`
	Any  *struct{} `json:"any,omitempty"`
	Tool *ConverseToolChoiceTool `json:"tool,omitempty"`
}

// ConverseToolChoiceTool specifies a specific tool.
type ConverseToolChoiceTool struct {
	Name string `json:"name"`
}

// GuardrailConfig for guardrail integration.
type GuardrailConfig struct {
	GuardrailIdentifier string `json:"guardrailIdentifier"`
	GuardrailVersion    string `json:"guardrailVersion"`
	Trace               string `json:"trace,omitempty"` // "enabled", "disabled"
	StreamProcessingMode string `json:"streamProcessingMode,omitempty"` // "sync", "async"
}

// RequestMetadata for tracking.
type RequestMetadata struct {
	RequestID string `json:"requestId,omitempty"`
}

// PerformanceConfig for performance tuning.
type PerformanceConfig struct {
	Latency string `json:"latency,omitempty"` // "standard", "optimized"
}

// PromptVariable for prompt template variables.
type PromptVariable struct {
	Text string `json:"text"`
}

// ========== Converse API Response Types ==========

// ConverseResponse is the non-streaming response.
type ConverseResponse struct {
	Output                       *ConverseOutput  `json:"output"`
	StopReason                   string           `json:"stopReason"`
	Usage                        *ConverseUsage   `json:"usage"`
	Metrics                      *ConverseMetrics `json:"metrics,omitempty"`
	AdditionalModelResponseFields map[string]any   `json:"additionalModelResponseFields,omitempty"`
	Trace                        *ConverseTrace   `json:"trace,omitempty"`
	PerformanceConfig            *PerformanceConfig `json:"performanceConfig,omitempty"`
}

// ConverseOutput wraps the assistant message.
type ConverseOutput struct {
	Message *ConverseMessage `json:"message,omitempty"`
}

// ConverseUsage reports token usage.
type ConverseUsage struct {
	InputTokens           int `json:"inputTokens"`
	OutputTokens          int `json:"outputTokens"`
	TotalTokens           int `json:"totalTokens"`
	CacheReadInputTokens  int `json:"cacheReadInputTokens,omitempty"`
	CacheWriteInputTokens int `json:"cacheWriteInputTokens,omitempty"`
}

// ConverseMetrics reports latency metrics.
type ConverseMetrics struct {
	LatencyMs int64 `json:"latencyMs"`
}

// ConverseTrace for guardrail/prompt router traces.
type ConverseTrace struct {
	Guardrail    any `json:"guardrail,omitempty"`
	PromptRouter any `json:"promptRouter,omitempty"`
}

// ========== ConverseStream Event Types ==========

// ConverseStreamMessageStart for messageStart event.
type ConverseStreamMessageStart struct {
	Role string `json:"role"` // "assistant"
}

// ConverseStreamContentBlockStart for contentBlockStart event.
type ConverseStreamContentBlockStart struct {
	ContentBlockIndex int                       `json:"contentBlockIndex"`
	Start             *ConverseContentBlockStart `json:"start"`
}

// ConverseContentBlockStart is the start payload (union).
type ConverseContentBlockStart struct {
	ToolUse *ConverseToolUseStart `json:"toolUse,omitempty"`
}

// ConverseToolUseStart for tool use block start.
type ConverseToolUseStart struct {
	ToolUseID string `json:"toolUseId"`
	Name      string `json:"name"`
}

// ConverseStreamContentBlockDelta for contentBlockDelta event.
type ConverseStreamContentBlockDelta struct {
	ContentBlockIndex int                       `json:"contentBlockIndex"`
	Delta             *ConverseContentBlockDelta `json:"delta"`
}

// ConverseContentBlockDelta is the delta payload (union).
type ConverseContentBlockDelta struct {
	Text             *string                     `json:"text,omitempty"`
	ToolUse          *ConverseToolUseDelta       `json:"toolUse,omitempty"`
	ReasoningContent *ReasoningContentBlockDelta `json:"reasoningContent,omitempty"`
}

// ConverseToolUseDelta for tool use input streaming.
type ConverseToolUseDelta struct {
	Input string `json:"input"` // incremental JSON string
}

// ReasoningContentBlockDelta for reasoning streaming.
type ReasoningContentBlockDelta struct {
	Text            *string `json:"text,omitempty"`
	RedactedContent *string `json:"redactedContent,omitempty"` // base64
	Signature       *string `json:"signature,omitempty"`
}

// ConverseStreamContentBlockStop for contentBlockStop event.
type ConverseStreamContentBlockStop struct {
	ContentBlockIndex int `json:"contentBlockIndex"`
}

// ConverseStreamMessageStop for messageStop event.
type ConverseStreamMessageStop struct {
	StopReason                    string `json:"stopReason"`
	AdditionalModelResponseFields any    `json:"additionalModelResponseFields,omitempty"`
}

// ConverseStreamMetadata for metadata event.
type ConverseStreamMetadata struct {
	Usage   *ConverseUsage   `json:"usage"`
	Metrics *ConverseStreamMetrics `json:"metrics"`
	Trace   *ConverseTrace   `json:"trace,omitempty"`
}

// ConverseStreamMetrics for stream metrics.
type ConverseStreamMetrics struct {
	LatencyMs int64 `json:"latencyMs"`
}

// StopReason constants for Converse API.
const (
	StopReasonEndTurn                  = "end_turn"
	StopReasonToolUse                  = "tool_use"
	StopReasonMaxTokens                = "max_tokens"
	StopReasonStopSequence             = "stop_sequence"
	StopReasonGuardrailIntervened      = "guardrail_intervened"
	StopReasonContentFiltered          = "content_filtered"
	StopReasonMalformedModelOutput     = "malformed_model_output"
	StopReasonMalformedToolUse         = "malformed_tool_use"
	StopReasonModelContextWindowExceeded = "model_context_window_exceeded"
)
