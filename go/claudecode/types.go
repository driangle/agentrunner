package claudecode

import "encoding/json"

// StreamMessage is the top-level envelope for all Claude stream-json lines.
// Each line from `claude -p --output-format stream-json` deserializes into this type.
type StreamMessage struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype,omitempty"`

	// Content holds content blocks lifted from the assistant message wrapper.
	// Populated by Parse for assistant-type messages.
	Content []ContentBlock `json:"-"`

	// AssistantMsg holds the nested "message" object in assistant-type lines.
	AssistantMsg *AssistantMessage `json:"message,omitempty"`

	// Result fields — present when Type == "result".
	Result        string       `json:"result,omitempty"`
	IsError       bool         `json:"is_error,omitempty"`
	TotalCostUSD  float64      `json:"total_cost_usd,omitempty"`
	DurationMs    float64      `json:"duration_ms,omitempty"`
	DurationAPIMs float64      `json:"duration_api_ms,omitempty"`
	NumTurns      int          `json:"num_turns,omitempty"`
	SessionID     string       `json:"session_id,omitempty"`
	Model         string       `json:"model,omitempty"`
	Usage         *ResultUsage `json:"usage,omitempty"`

	// System/init fields.
	Tools []json.RawMessage `json:"tools,omitempty"`

	// RateLimitInfo holds details from rate_limit_event messages.
	RateLimitInfo *RateLimitInfo `json:"rate_limit_info,omitempty"`

	// StreamEvent fields — present when Type == "stream_event".
	EventRaw        json.RawMessage  `json:"event,omitempty"`
	ParentToolUseID string           `json:"parent_tool_use_id,omitempty"`
	Event           *StreamEventInner `json:"-"` // parsed from EventRaw by Parse
}

// AssistantMessage is the nested "message" object inside assistant-type stream lines.
type AssistantMessage struct {
	Model      string         `json:"model,omitempty"`
	ID         string         `json:"id,omitempty"`
	Content    []ContentBlock `json:"content,omitempty"`
	StopReason string         `json:"stop_reason,omitempty"`
	Usage      *StreamUsage   `json:"usage,omitempty"`
}

// ContentBlock represents one block inside an assistant message.
type ContentBlock struct {
	// Type is the block kind: "text", "thinking", "tool_use", or "tool_result".
	Type string `json:"type"`

	// Text holds text content for "text" blocks.
	Text string `json:"text,omitempty"`

	// Thinking holds thinking content for "thinking" blocks.
	Thinking string `json:"thinking,omitempty"`

	// Name is the tool name for "tool_use" blocks.
	Name string `json:"name,omitempty"`

	// Input holds the tool call arguments for "tool_use" blocks.
	Input json.RawMessage `json:"input,omitempty"`

	// Content holds the tool execution output for "tool_result" blocks.
	Content json.RawMessage `json:"content,omitempty"`
}

// StreamEventInner is the parsed inner event from a stream_event line.
// These correspond to Anthropic API streaming event types.
type StreamEventInner struct {
	// Type is the event kind: message_start, content_block_start,
	// content_block_delta, content_block_stop, message_delta, message_stop.
	Type string `json:"type"`

	// Message holds metadata from message_start events.
	Message *MessageStartData `json:"message,omitempty"`

	// Index identifies the content block in block events.
	Index int `json:"index,omitempty"`

	// ContentBlock describes a block in content_block_start events.
	ContentBlock *ContentBlockInfo `json:"content_block,omitempty"`

	// Delta carries incremental data in delta events.
	Delta *Delta `json:"delta,omitempty"`

	// Usage carries token counts in message_delta events.
	Usage *StreamUsage `json:"usage,omitempty"`
}

// MessageStartData carries message metadata from a message_start event.
type MessageStartData struct {
	Model string       `json:"model"`
	ID    string       `json:"id"`
	Usage *StreamUsage `json:"usage,omitempty"`
}

// ContentBlockInfo describes a content block in content_block_start events.
type ContentBlockInfo struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
	ID   string `json:"id,omitempty"`
}

// Delta carries incremental data in content_block_delta and message_delta events.
type Delta struct {
	Type         string `json:"type,omitempty"`
	Text         string `json:"text,omitempty"`
	Thinking     string `json:"thinking,omitempty"`
	PartialJSON  string `json:"partial_json,omitempty"`
	StopReason   string `json:"stop_reason,omitempty"`
	StopSequence string `json:"stop_sequence,omitempty"`
}

// RateLimitInfo carries details from rate_limit_event messages.
type RateLimitInfo struct {
	Status         string  `json:"status"`
	RateLimitType  string  `json:"rateLimitType,omitempty"`
	Utilization    float64 `json:"utilization,omitempty"`
	ResetsAt       int64   `json:"resetsAt,omitempty"`
	IsUsingOverage bool    `json:"isUsingOverage,omitempty"`
}

// ResultUsage carries token counts from the final result message.
// It includes cache token fields that StreamUsage lacks.
type ResultUsage struct {
	InputTokens              int `json:"input_tokens,omitempty"`
	OutputTokens             int `json:"output_tokens,omitempty"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// StreamUsage carries token counts from streaming events.
type StreamUsage struct {
	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`
}
