package claudecode

import (
	"encoding/json"

	agentrunner "github.com/driangle/agent-runner/go"
)

// Parse parses a single JSON line from Claude Code's stream-json output
// into a typed StreamMessage. Unknown fields are silently ignored for
// forward compatibility.
//
// For assistant-type lines, content blocks are lifted from the nested
// "message" wrapper into StreamMessage.Content for convenient access.
// For stream_event lines, the inner event is parsed into StreamMessage.Event.
func Parse(line []byte) (StreamMessage, error) {
	var msg StreamMessage
	if err := json.Unmarshal(line, &msg); err != nil {
		return StreamMessage{}, err
	}

	switch msg.Type {
	case "assistant":
		if msg.AssistantMsg != nil {
			msg.Content = msg.AssistantMsg.Content
		}

	case "stream_event":
		if len(msg.EventRaw) > 0 {
			var inner StreamEventInner
			if err := json.Unmarshal(msg.EventRaw, &inner); err != nil {
				return msg, nil // envelope OK, inner unparseable — skip gracefully
			}
			msg.Event = &inner
		}
	}

	return msg, nil
}

// ParseMessage extracts the Claude-specific StreamMessage from a common Message.
// If the message was produced by the Claude runner, it returns the pre-parsed value
// from Parsed. Otherwise it parses Raw.
func ParseMessage(msg agentrunner.Message) (*StreamMessage, error) {
	if sm, ok := msg.Parsed.(*StreamMessage); ok {
		return sm, nil
	}
	parsed, err := Parse(msg.Raw)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
