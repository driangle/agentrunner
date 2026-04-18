package codex

import "encoding/json"

// StreamMessage is the top-level envelope for all Codex JSONL lines.
// Each line from `codex exec --json` deserializes into this type.
type StreamMessage struct {
	Type     string `json:"type"`
	ThreadID string `json:"thread_id,omitempty"`

	// Item holds the nested item for item.started and item.completed events.
	Item *Item `json:"item,omitempty"`

	// Usage holds token counts from turn.completed events.
	Usage *TurnUsage `json:"usage,omitempty"`

	// Message holds the error message for error-type events.
	Message string `json:"message,omitempty"`

	// Error holds structured error info from turn.failed events.
	Error *TurnError `json:"error,omitempty"`
}

// Text returns the text content from this message, if available.
// Returns the text field from agent_message items.
func (m *StreamMessage) Text() string {
	if m.Item != nil && m.Item.Type == "agent_message" {
		return m.Item.Text
	}
	return ""
}

// Thinking returns empty string — Codex does not expose thinking content.
func (m *StreamMessage) Thinking() string {
	return ""
}

// ToolName returns the command being executed for command_execution items.
func (m *StreamMessage) ToolName() string {
	if m.Item != nil && m.Item.Type == "command_execution" {
		return m.Item.Command
	}
	return ""
}

// ToolInput returns the command as raw JSON for command_execution items.
func (m *StreamMessage) ToolInput() json.RawMessage {
	if m.Item != nil && m.Item.Type == "command_execution" && m.Item.Command != "" {
		b, _ := json.Marshal(m.Item.Command)
		return b
	}
	return nil
}

// ToolOutput returns the aggregated output as raw JSON for completed command_execution items.
func (m *StreamMessage) ToolOutput() json.RawMessage {
	if m.Item != nil && m.Item.Type == "command_execution" && m.Item.Status == "completed" {
		b, _ := json.Marshal(m.Item.AggregatedOutput)
		return b
	}
	return nil
}

// IsErrorResult reports whether this message represents an error.
func (m *StreamMessage) IsErrorResult() bool {
	return m.Type == "error" || m.Type == "turn.failed"
}

// ErrorMessage returns the error description.
func (m *StreamMessage) ErrorMessage() string {
	if m.Type == "error" {
		return m.Message
	}
	if m.Type == "turn.failed" && m.Error != nil {
		return m.Error.Message
	}
	return ""
}

// Item represents a Codex event item (agent_message or command_execution).
type Item struct {
	ID               string `json:"id"`
	Type             string `json:"type"`
	Text             string `json:"text,omitempty"`
	Command          string `json:"command,omitempty"`
	AggregatedOutput string `json:"aggregated_output,omitempty"`
	ExitCode         *int   `json:"exit_code,omitempty"`
	Status           string `json:"status,omitempty"`
}

// TurnUsage holds token counts from a turn.completed event.
type TurnUsage struct {
	InputTokens        int `json:"input_tokens,omitempty"`
	CachedInputTokens  int `json:"cached_input_tokens,omitempty"`
	OutputTokens       int `json:"output_tokens,omitempty"`
}

// TurnError holds error details from a turn.failed event.
type TurnError struct {
	Message string `json:"message"`
}
