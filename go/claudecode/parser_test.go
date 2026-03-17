package claudecode

import (
	"bufio"
	"os"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		check   func(t *testing.T, msg StreamMessage)
		wantErr bool
	}{
		{
			name:  "system/init",
			input: `{"type":"system","subtype":"init","session_id":"abc-123","model":"claude-sonnet-4-6","tools":["Bash","Read"]}`,
			check: func(t *testing.T, msg StreamMessage) {
				if msg.Type != "system" {
					t.Errorf("type = %q, want %q", msg.Type, "system")
				}
				if msg.Subtype != "init" {
					t.Errorf("subtype = %q, want %q", msg.Subtype, "init")
				}
				if msg.SessionID != "abc-123" {
					t.Errorf("session_id = %q, want %q", msg.SessionID, "abc-123")
				}
				if msg.Model != "claude-sonnet-4-6" {
					t.Errorf("model = %q, want %q", msg.Model, "claude-sonnet-4-6")
				}
				if len(msg.Tools) != 2 {
					t.Errorf("tools len = %d, want 2", len(msg.Tools))
				}
			},
		},
		{
			name:  "assistant/text",
			input: `{"type":"assistant","message":{"model":"claude-sonnet-4-6","id":"msg_01","content":[{"type":"text","text":"Hello world"}],"stop_reason":"end_turn","usage":{"input_tokens":10,"output_tokens":5}}}`,
			check: func(t *testing.T, msg StreamMessage) {
				if msg.Type != "assistant" {
					t.Errorf("type = %q, want %q", msg.Type, "assistant")
				}
				if len(msg.Content) != 1 {
					t.Fatalf("content len = %d, want 1", len(msg.Content))
				}
				if msg.Content[0].Type != "text" {
					t.Errorf("content[0].type = %q, want %q", msg.Content[0].Type, "text")
				}
				if msg.Content[0].Text != "Hello world" {
					t.Errorf("content[0].text = %q, want %q", msg.Content[0].Text, "Hello world")
				}
				if msg.AssistantMsg == nil {
					t.Fatal("assistant_msg is nil")
				}
				if msg.AssistantMsg.StopReason != "end_turn" {
					t.Errorf("stop_reason = %q, want %q", msg.AssistantMsg.StopReason, "end_turn")
				}
				if msg.AssistantMsg.Usage == nil || msg.AssistantMsg.Usage.InputTokens != 10 {
					t.Errorf("usage.input_tokens = %v, want 10", msg.AssistantMsg.Usage)
				}
			},
		},
		{
			name:  "assistant/thinking",
			input: `{"type":"assistant","message":{"model":"claude-sonnet-4-6","id":"msg_02","content":[{"type":"thinking","thinking":"Let me think about this..."}]}}`,
			check: func(t *testing.T, msg StreamMessage) {
				if len(msg.Content) != 1 {
					t.Fatalf("content len = %d, want 1", len(msg.Content))
				}
				if msg.Content[0].Type != "thinking" {
					t.Errorf("content[0].type = %q, want %q", msg.Content[0].Type, "thinking")
				}
				if msg.Content[0].Thinking != "Let me think about this..." {
					t.Errorf("content[0].thinking = %q, want %q", msg.Content[0].Thinking, "Let me think about this...")
				}
			},
		},
		{
			name:  "assistant/tool_use",
			input: `{"type":"assistant","message":{"model":"claude-sonnet-4-6","id":"msg_03","content":[{"type":"tool_use","name":"Read","input":{"file_path":"/tmp/test.go"}}]}}`,
			check: func(t *testing.T, msg StreamMessage) {
				if len(msg.Content) != 1 {
					t.Fatalf("content len = %d, want 1", len(msg.Content))
				}
				if msg.Content[0].Type != "tool_use" {
					t.Errorf("content[0].type = %q, want %q", msg.Content[0].Type, "tool_use")
				}
				if msg.Content[0].Name != "Read" {
					t.Errorf("content[0].name = %q, want %q", msg.Content[0].Name, "Read")
				}
				if len(msg.Content[0].Input) == 0 {
					t.Error("content[0].input is empty")
				}
			},
		},
		{
			name:  "result/success",
			input: `{"type":"result","subtype":"success","is_error":false,"result":"Task completed","total_cost_usd":0.05,"duration_ms":1234,"duration_api_ms":1100,"num_turns":3,"session_id":"sess-1"}`,
			check: func(t *testing.T, msg StreamMessage) {
				if msg.Type != "result" {
					t.Errorf("type = %q, want %q", msg.Type, "result")
				}
				if msg.Subtype != "success" {
					t.Errorf("subtype = %q, want %q", msg.Subtype, "success")
				}
				if msg.IsError {
					t.Error("is_error = true, want false")
				}
				if msg.Result != "Task completed" {
					t.Errorf("result = %q, want %q", msg.Result, "Task completed")
				}
				if msg.TotalCostUSD != 0.05 {
					t.Errorf("total_cost_usd = %f, want 0.05", msg.TotalCostUSD)
				}
				if msg.DurationMs != 1234 {
					t.Errorf("duration_ms = %f, want 1234", msg.DurationMs)
				}
				if msg.NumTurns != 3 {
					t.Errorf("num_turns = %d, want 3", msg.NumTurns)
				}
				if msg.SessionID != "sess-1" {
					t.Errorf("session_id = %q, want %q", msg.SessionID, "sess-1")
				}
			},
		},
		{
			name:  "result/error",
			input: `{"type":"result","subtype":"error","is_error":true,"result":"Something failed","session_id":"sess-2"}`,
			check: func(t *testing.T, msg StreamMessage) {
				if msg.Subtype != "error" {
					t.Errorf("subtype = %q, want %q", msg.Subtype, "error")
				}
				if !msg.IsError {
					t.Error("is_error = false, want true")
				}
				if msg.Result != "Something failed" {
					t.Errorf("result = %q, want %q", msg.Result, "Something failed")
				}
			},
		},
		{
			name:  "stream_event/message_start",
			input: `{"type":"stream_event","event":{"type":"message_start","message":{"model":"claude-sonnet-4-6","id":"msg_04","usage":{"input_tokens":100,"output_tokens":0}}},"session_id":"sess-3"}`,
			check: func(t *testing.T, msg StreamMessage) {
				if msg.Type != "stream_event" {
					t.Errorf("type = %q, want %q", msg.Type, "stream_event")
				}
				if msg.Event == nil {
					t.Fatal("event is nil")
				}
				if msg.Event.Type != "message_start" {
					t.Errorf("event.type = %q, want %q", msg.Event.Type, "message_start")
				}
				if msg.Event.Message == nil {
					t.Fatal("event.message is nil")
				}
				if msg.Event.Message.Model != "claude-sonnet-4-6" {
					t.Errorf("event.message.model = %q, want %q", msg.Event.Message.Model, "claude-sonnet-4-6")
				}
			},
		},
		{
			name:  "stream_event/content_block_start",
			input: `{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_01","name":"Bash"}},"session_id":"sess-3"}`,
			check: func(t *testing.T, msg StreamMessage) {
				if msg.Event == nil {
					t.Fatal("event is nil")
				}
				if msg.Event.Type != "content_block_start" {
					t.Errorf("event.type = %q, want %q", msg.Event.Type, "content_block_start")
				}
				if msg.Event.ContentBlock == nil {
					t.Fatal("event.content_block is nil")
				}
				if msg.Event.ContentBlock.Name != "Bash" {
					t.Errorf("event.content_block.name = %q, want %q", msg.Event.ContentBlock.Name, "Bash")
				}
			},
		},
		{
			name:  "stream_event/content_block_delta/text",
			input: `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}},"session_id":"sess-3"}`,
			check: func(t *testing.T, msg StreamMessage) {
				if msg.Event == nil {
					t.Fatal("event is nil")
				}
				if msg.Event.Delta == nil {
					t.Fatal("event.delta is nil")
				}
				if msg.Event.Delta.Type != "text_delta" {
					t.Errorf("delta.type = %q, want %q", msg.Event.Delta.Type, "text_delta")
				}
				if msg.Event.Delta.Text != "Hello" {
					t.Errorf("delta.text = %q, want %q", msg.Event.Delta.Text, "Hello")
				}
			},
		},
		{
			name:  "stream_event/content_block_delta/thinking",
			input: `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"hmm"}},"session_id":"sess-3"}`,
			check: func(t *testing.T, msg StreamMessage) {
				if msg.Event.Delta.Thinking != "hmm" {
					t.Errorf("delta.thinking = %q, want %q", msg.Event.Delta.Thinking, "hmm")
				}
			},
		},
		{
			name:  "stream_event/content_block_delta/input_json",
			input: `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"key\":"}},"session_id":"sess-3"}`,
			check: func(t *testing.T, msg StreamMessage) {
				if msg.Event.Delta.PartialJSON != `{"key":` {
					t.Errorf("delta.partial_json = %q, want %q", msg.Event.Delta.PartialJSON, `{"key":`)
				}
			},
		},
		{
			name:  "stream_event/message_delta",
			input: `{"type":"stream_event","event":{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":42}},"session_id":"sess-3"}`,
			check: func(t *testing.T, msg StreamMessage) {
				if msg.Event.Delta.StopReason != "end_turn" {
					t.Errorf("delta.stop_reason = %q, want %q", msg.Event.Delta.StopReason, "end_turn")
				}
				if msg.Event.Usage == nil || msg.Event.Usage.OutputTokens != 42 {
					t.Errorf("usage.output_tokens = %v, want 42", msg.Event.Usage)
				}
			},
		},
		{
			name:  "rate_limit_event",
			input: `{"type":"rate_limit_event","rate_limit_info":{"status":"allowed_warning","rateLimitType":"seven_day","utilization":0.56,"resetsAt":1772805600,"isUsingOverage":false}}`,
			check: func(t *testing.T, msg StreamMessage) {
				if msg.Type != "rate_limit_event" {
					t.Errorf("type = %q, want %q", msg.Type, "rate_limit_event")
				}
				if msg.RateLimitInfo == nil {
					t.Fatal("rate_limit_info is nil")
				}
				if msg.RateLimitInfo.Status != "allowed_warning" {
					t.Errorf("status = %q, want %q", msg.RateLimitInfo.Status, "allowed_warning")
				}
				if msg.RateLimitInfo.Utilization != 0.56 {
					t.Errorf("utilization = %f, want 0.56", msg.RateLimitInfo.Utilization)
				}
				if msg.RateLimitInfo.IsUsingOverage {
					t.Error("is_using_overage = true, want false")
				}
			},
		},
		{
			name:  "unknown type is forward compatible",
			input: `{"type":"new_future_type","some_field":"value"}`,
			check: func(t *testing.T, msg StreamMessage) {
				if msg.Type != "new_future_type" {
					t.Errorf("type = %q, want %q", msg.Type, "new_future_type")
				}
			},
		},
		{
			name:    "invalid JSON returns error",
			input:   `{not valid json}`,
			wantErr: true,
		},
		{
			name:    "empty string returns error",
			input:   ``,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.check != nil {
				tt.check(t, msg)
			}
		})
	}
}

func TestParseSessionFixture(t *testing.T) {
	f, err := os.Open("testdata/claude-code-session.jsonl")
	if err != nil {
		t.Skipf("fixture not available: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024) // 10MB max line

	typeCounts := make(map[string]int)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if line == "" {
			continue
		}

		msg, err := Parse(line)
		if err != nil {
			t.Errorf("line %d: Parse error: %v", lineNum, err)
			continue
		}

		typeCounts[msg.Type]++

		// Verify assistant messages have content lifted
		if msg.Type == "assistant" && msg.AssistantMsg != nil && len(msg.AssistantMsg.Content) > 0 {
			if len(msg.Content) == 0 {
				t.Errorf("line %d: assistant message has content in wrapper but not lifted", lineNum)
			}
		}

		// Verify stream events have inner event parsed
		if msg.Type == "stream_event" && len(msg.EventRaw) > 0 {
			if msg.Event == nil {
				t.Errorf("line %d: stream_event has event_raw but Event is nil", lineNum)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner error: %v", err)
	}

	// Verify we saw the expected message types
	expectedTypes := []string{"system", "assistant", "result", "stream_event", "rate_limit_event"}
	for _, typ := range expectedTypes {
		if typeCounts[typ] == 0 {
			t.Errorf("expected at least one %q message, got none", typ)
		}
	}

	t.Logf("parsed %d lines, type distribution: %v", lineNum, typeCounts)
}
