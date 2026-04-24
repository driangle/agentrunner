package gemini

import "encoding/json"

// StreamMessage is the top-level envelope for all Gemini stream-json JSONL lines.
// Each line from `gemini --output-format stream-json` deserializes into this type.
// Fields are populated based on the event type.
type StreamMessage struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp,omitempty"`

	// init fields
	SessionID string `json:"session_id,omitempty"`
	Model     string `json:"model,omitempty"`

	// message fields
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
	Delta   bool   `json:"delta,omitempty"`

	// tool_use fields
	ToolNameField string          `json:"tool_name,omitempty"`
	ToolID        string          `json:"tool_id,omitempty"`
	Parameters    json.RawMessage `json:"parameters,omitempty"`

	// tool_result fields
	Status string `json:"status,omitempty"`
	Output string `json:"output,omitempty"`

	// error fields
	Severity     string `json:"severity,omitempty"`
	MessageField string `json:"message,omitempty"`

	// result fields
	Error *EventError  `json:"error,omitempty"`
	Stats *StreamStats `json:"stats,omitempty"`
}

// Text returns the content from assistant message events.
func (m *StreamMessage) Text() string {
	if m.Type == "message" && m.Role == "assistant" {
		return m.Content
	}
	return ""
}

// Thinking returns empty string — Gemini CLI does not expose thinking content.
func (m *StreamMessage) Thinking() string {
	return ""
}

// ToolName returns the tool name from tool_use events.
func (m *StreamMessage) ToolName() string {
	if m.Type == "tool_use" {
		return m.ToolNameField
	}
	return ""
}

// ToolInput returns the tool parameters as raw JSON from tool_use events.
func (m *StreamMessage) ToolInput() json.RawMessage {
	if m.Type == "tool_use" && len(m.Parameters) > 0 {
		return m.Parameters
	}
	return nil
}

// ToolOutput returns the tool output as raw JSON from tool_result events.
func (m *StreamMessage) ToolOutput() json.RawMessage {
	if m.Type == "tool_result" && m.Output != "" {
		b, _ := json.Marshal(m.Output)
		return b
	}
	return nil
}

// IsErrorResult reports whether this message represents an error.
func (m *StreamMessage) IsErrorResult() bool {
	if m.Type == "error" {
		return true
	}
	if m.Type == "result" && m.Status == "error" {
		return true
	}
	return false
}

// ErrorMessage returns the error description.
func (m *StreamMessage) ErrorMessage() string {
	if m.Type == "error" {
		return m.MessageField
	}
	if m.Type == "result" && m.Error != nil {
		return m.Error.Message
	}
	return ""
}

// EventError holds error details from result or error events.
type EventError struct {
	Type    string `json:"type,omitempty"`
	Message string `json:"message"`
}

// StreamStats holds token counts and timing from a result event.
type StreamStats struct {
	TotalTokens  int `json:"total_tokens,omitempty"`
	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`
	Cached       int `json:"cached,omitempty"`
	Input        int `json:"input,omitempty"`
	DurationMs   int `json:"duration_ms,omitempty"`
	ToolCalls    int `json:"tool_calls,omitempty"`

	// Models holds per-model token breakdowns.
	Models map[string]ModelStreamStats `json:"models,omitempty"`
}

// ModelStreamStats holds per-model token counts.
type ModelStreamStats struct {
	TotalTokens  int `json:"total_tokens,omitempty"`
	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`
	Cached       int `json:"cached,omitempty"`
	Input        int `json:"input,omitempty"`
}
