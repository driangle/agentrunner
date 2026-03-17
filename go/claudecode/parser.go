package claudecode

import "encoding/json"

// Parse parses a single JSON line from Claude Code's stream-json output
// into a typed StreamMessage. Unknown fields are silently ignored for
// forward compatibility.
//
// For assistant-type lines, content blocks are lifted from the nested
// "message" wrapper into StreamMessage.Content for convenient access.
// For stream_event lines, the inner event is parsed into StreamMessage.Event.
func Parse(line string) (StreamMessage, error) {
	var msg StreamMessage
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
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
