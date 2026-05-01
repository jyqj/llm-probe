// Package types defines the data structures for Anthropic Messages API and Bedrock Converse API.
package types

// ========== Anthropic Messages API Types ==========

// MessagesRequest is the Anthropic Messages API request body.
// POST /v1/messages
type MessagesRequest struct {
	Model         string          `json:"model"`
	Messages      []Message       `json:"messages"`
	MaxTokens     int             `json:"max_tokens"`
	System        any             `json:"system,omitempty"`        // string or []SystemBlock
	Metadata      *Metadata       `json:"metadata,omitempty"`
	StopSequences []string        `json:"stop_sequences,omitempty"`
	Stream        bool            `json:"stream,omitempty"`
	Temperature   *float64        `json:"temperature,omitempty"`
	TopP          *float64        `json:"top_p,omitempty"`
	TopK          *int            `json:"top_k,omitempty"`
	Tools         []Tool          `json:"tools,omitempty"`
	ToolChoice    any             `json:"tool_choice,omitempty"` // string or ToolChoice object
	OutputConfig  *OutputConfig   `json:"output_config,omitempty"`

	// Extended thinking (Claude 3.7+ / Claude 4+)
	Thinking *ThinkingConfig `json:"thinking,omitempty"`

	// Anthropic-specific headers passed as fields by some providers
	AnthropicVersion string `json:"anthropic_version,omitempty"`
	AnthropicBeta    any    `json:"anthropic_beta,omitempty"` // string or []string
}

// Message represents a conversation message.
type Message struct {
	Role    string `json:"role"` // "user" or "assistant"
	Content any    `json:"content"`  // string or []ContentBlock
}

// ContentBlock is a union type for message content.
type ContentBlock struct {
	Type string `json:"type"` // "text", "image", "tool_use", "tool_result", "document", "thinking", "redacted_thinking"

	// text
	Text string `json:"text,omitempty"`

	// image
	Source *ImageSource `json:"source,omitempty"`

	// tool_use
	ID     string       `json:"id,omitempty"`
	Name   string       `json:"name,omitempty"`
	Input  any          `json:"input,omitempty"`  // JSON object
	Caller *CallerInfo  `json:"caller,omitempty"` // e.g. {"type":"direct"}

	// tool_result
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   any    `json:"content,omitempty"`   // string or []ContentBlock
	IsError   bool   `json:"is_error,omitempty"`

	// document
	Document *DocumentSource `json:"document,omitempty"`

	// thinking (extended thinking)
	Thinking  string `json:"thinking,omitempty"`
	Signature string `json:"signature,omitempty"`

	// redacted_thinking
	Data string `json:"data,omitempty"`

	// citations
	Citations []Citation `json:"citations,omitempty"`

	// cache_control
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

// CallerInfo indicates how a tool_use was invoked.
type CallerInfo struct {
	Type string `json:"type"` // "direct"
}

// ImageSource describes an image input.
type ImageSource struct {
	Type      string `json:"type"` // "base64" or "url"
	MediaType string `json:"media_type,omitempty"` // "image/jpeg", "image/png", "image/gif", "image/webp"
	Data      string `json:"data,omitempty"`
	URL       string `json:"url,omitempty"`
}

// DocumentSource describes a document input.
type DocumentSource struct {
	Type      string `json:"type"` // "base64"
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
	Name      string `json:"name,omitempty"`
}

// Citation for citation content.
type Citation struct {
	Type          string         `json:"type"`
	CitedText     string         `json:"cited_text,omitempty"`
	DocumentIndex int            `json:"document_index,omitempty"`
	DocumentTitle string         `json:"document_title,omitempty"`
	StartIndex    int            `json:"start_index,omitempty"`
	EndIndex      int            `json:"end_index,omitempty"`
	URL           string         `json:"url,omitempty"`
}

// Tool defines a tool/function.
type Tool struct {
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	InputSchema any        `json:"input_schema"` // JSON Schema
	Type        string     `json:"type,omitempty"` // "custom", "computer_20241022", etc.
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

// ToolChoiceObj represents the tool_choice field.
type ToolChoiceObj struct {
	Type string `json:"type"` // "auto", "any", "tool", "none"
	Name string `json:"name,omitempty"` // when type is "tool"
}

// OutputConfig for structured output (JSON schema).
type OutputConfig struct {
	Format *OutputFormat `json:"format,omitempty"`
}

// OutputFormat specifies the output format.
type OutputFormat struct {
	Type   string `json:"type"`             // "json_schema", "text"
	Schema any    `json:"schema,omitempty"` // JSON Schema object
	Name   string `json:"name,omitempty"`
}

// ThinkingConfig for extended thinking.
type ThinkingConfig struct {
	Type         string `json:"type"`           // "enabled", "disabled"
	BudgetTokens int    `json:"budget_tokens,omitempty"`
}

// Metadata for request metadata.
type Metadata struct {
	UserID string `json:"user_id,omitempty"`
}

// CacheControl for prompt caching.
type CacheControl struct {
	Type string `json:"type"` // "ephemeral"
}

// SystemBlock represents a system content block.
type SystemBlock struct {
	Type         string        `json:"type"` // "text"
	Text         string        `json:"text"`
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

// ========== Anthropic Messages API Response Types ==========

// MessagesResponse is the non-streaming response.
type MessagesResponse struct {
	ID                string             `json:"id"`
	Type              string             `json:"type"` // "message"
	Role              string             `json:"role"` // "assistant"
	Content           []ContentBlock     `json:"content"`
	Model             string             `json:"model"`
	StopReason        string             `json:"stop_reason"`          // "end_turn", "max_tokens", "stop_sequence", "tool_use"
	StopSequence      *string            `json:"stop_sequence"`
	StopDetails       *StopDetails       `json:"stop_details"`         // null or stop details
	Usage             *MessagesUsage     `json:"usage"`
	ContextManagement *ContextManagement `json:"context_management,omitempty"`
}

// StopDetails provides additional info about why generation stopped.
type StopDetails struct {
	Type string `json:"type,omitempty"`
}

// ContextManagement for context window management info.
type ContextManagement struct {
	AppliedEdits []any `json:"applied_edits"`
}

// MessagesUsage reports token usage (matches latest official API).
type MessagesUsage struct {
	InputTokens              int           `json:"input_tokens"`
	OutputTokens             int           `json:"output_tokens"`
	CacheCreationInputTokens int           `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int           `json:"cache_read_input_tokens"`
	CacheCreation            *CacheCreationDetail `json:"cache_creation,omitempty"`
	ServiceTier              string        `json:"service_tier,omitempty"`
	InferenceGeo             string        `json:"inference_geo,omitempty"`
	Speed                    string        `json:"speed,omitempty"`
}

// CacheCreationDetail for tiered caching info.
type CacheCreationDetail struct {
	Ephemeral5mInputTokens int `json:"ephemeral_5m_input_tokens"`
	Ephemeral1hInputTokens int `json:"ephemeral_1h_input_tokens"`
}

// ========== Streaming Event Types (SSE) ==========

// StreamEvent is a server-sent event.
type StreamEvent struct {
	Type string `json:"type"`
	// The actual payload varies by type:
	// message_start   → { message: MessagesResponse }
	// content_block_start → { index, content_block }
	// content_block_delta → { index, delta }
	// content_block_stop  → { index }
	// message_delta   → { delta: {stop_reason, stop_sequence}, usage: {output_tokens} }
	// message_stop    → {}
	// ping            → {}
}

// MessageStartEvent for "message_start" SSE.
type MessageStartEvent struct {
	Type    string           `json:"type"` // "message_start"
	Message *MessagesResponse `json:"message"`
}

// ContentBlockStartEvent for "content_block_start" SSE.
type ContentBlockStartEvent struct {
	Type         string        `json:"type"` // "content_block_start"
	Index        int           `json:"index"`
	ContentBlock *ContentBlock `json:"content_block"`
}

// ContentBlockDeltaEvent for "content_block_delta" SSE.
type ContentBlockDeltaEvent struct {
	Type  string        `json:"type"` // "content_block_delta"
	Index int           `json:"index"`
	Delta *ContentDelta `json:"delta"`
}

// ContentDelta is the delta payload within a content_block_delta event.
type ContentDelta struct {
	Type string `json:"type"` // "text_delta", "input_json_delta", "thinking_delta", "signature_delta"

	// text_delta
	Text string `json:"text,omitempty"`

	// input_json_delta (tool_use streaming)
	PartialJSON string `json:"partial_json,omitempty"`

	// thinking_delta
	Thinking string `json:"thinking,omitempty"`

	// signature_delta
	Signature string `json:"signature,omitempty"`
}

// ContentBlockStopEvent for "content_block_stop" SSE.
type ContentBlockStopEvent struct {
	Type  string `json:"type"` // "content_block_stop"
	Index int    `json:"index"`
}

// MessageDeltaEvent for "message_delta" SSE.
type MessageDeltaEvent struct {
	Type              string             `json:"type"` // "message_delta"
	Delta             *MessageDeltaBody  `json:"delta"`
	Usage             *MessageDeltaUsage `json:"usage,omitempty"`
	ContextManagement *ContextManagement `json:"context_management,omitempty"`
}

// MessageDeltaBody is the delta in message_delta.
type MessageDeltaBody struct {
	StopReason   string       `json:"stop_reason"`
	StopSequence *string      `json:"stop_sequence"`
	StopDetails  *StopDetails `json:"stop_details"`
}

// MessageDeltaUsage is the usage in message_delta (streaming returns more fields).
type MessageDeltaUsage struct {
	InputTokens              int `json:"input_tokens,omitempty"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
	OutputTokens             int `json:"output_tokens"`
}


// MessageStopEvent for "message_stop" SSE.
type MessageStopEvent struct {
	Type string `json:"type"` // "message_stop"
}

// PingEvent for "ping" SSE.
type PingEvent struct {
	Type string `json:"type"` // "ping"
}
