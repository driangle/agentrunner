package claudecode

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/driangle/agent-runner/agentrunner"
)

// helperProcess is invoked when the test binary is re-executed with
// GO_HELPER_PROCESS=1. It writes canned output to stdout based on
// GO_HELPER_MODE and exits with GO_HELPER_EXIT code.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_HELPER_PROCESS") != "1" {
		return
	}
	mode := os.Getenv("GO_HELPER_MODE")
	switch mode {
	case "happy":
		fmt.Println(`{"type":"system","subtype":"init","session_id":"sess-1","model":"claude-sonnet-4-6"}`)
		fmt.Println(`{"type":"assistant","message":{"model":"claude-sonnet-4-6","id":"msg_01","content":[{"type":"text","text":"Hello world"}]}}`)
		fmt.Println(`{"type":"result","subtype":"success","result":"Hello world","is_error":false,"total_cost_usd":0.05,"duration_ms":1234,"duration_api_ms":1100,"num_turns":2,"session_id":"sess-1","usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":10,"cache_read_input_tokens":20}}`)
	case "error_result":
		fmt.Println(`{"type":"result","subtype":"error","result":"Something failed","is_error":true,"session_id":"sess-err","usage":{"input_tokens":10,"output_tokens":5}}`)
	case "no_result":
		fmt.Println(`{"type":"system","subtype":"init","session_id":"sess-x"}`)
	case "nonzero_exit":
		fmt.Fprintln(os.Stderr, "fatal error from claude")
		os.Exit(1)
	case "stream_multi":
		fmt.Println(`{"type":"system","subtype":"init","session_id":"sess-s1","model":"claude-sonnet-4-6"}`)
		fmt.Println(`{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}},"session_id":"sess-s1"}`)
		fmt.Println(`{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" world"}},"session_id":"sess-s1"}`)
		fmt.Println(`{"type":"assistant","message":{"model":"claude-sonnet-4-6","id":"msg_01","content":[{"type":"text","text":"Hello world"}]}}`)
		fmt.Println(`{"type":"result","subtype":"success","result":"Hello world","is_error":false,"total_cost_usd":0.05,"duration_ms":500,"session_id":"sess-s1","usage":{"input_tokens":100,"output_tokens":50}}`)
	case "stream_no_result":
		fmt.Println(`{"type":"system","subtype":"init","session_id":"sess-nr"}`)
		fmt.Println(`{"type":"assistant","message":{"model":"claude-sonnet-4-6","id":"msg_01","content":[{"type":"text","text":"partial"}]}}`)
	case "init_session_only":
		fmt.Println(`{"type":"system","subtype":"init","session_id":"sess-from-init","model":"claude-sonnet-4-6"}`)
		fmt.Println(`{"type":"result","subtype":"success","result":"done","is_error":false,"total_cost_usd":0.01,"duration_ms":100,"usage":{"input_tokens":10,"output_tokens":5}}`)
	case "slow":
		time.Sleep(5 * time.Second)
		fmt.Println(`{"type":"result","subtype":"success","result":"late","session_id":"sess-slow"}`)
	default:
		fmt.Fprintln(os.Stderr, "unknown mode: "+mode)
		os.Exit(2)
	}
	os.Exit(0)
}

// helperBuilder returns a commandBuilder that re-executes the test binary
// as a helper process with the given mode.
func helperBuilder(mode string) commandBuilder {
	return func(ctx context.Context, name string, args ...string) *exec.Cmd {
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestHelperProcess$")
		cmd.Env = append(os.Environ(),
			"GO_HELPER_PROCESS=1",
			"GO_HELPER_MODE="+mode,
		)
		return cmd
	}
}

// --- Run tests (delegate to Start) ---

func TestRunHappyPath(t *testing.T) {
	r := NewRunner(withCommandBuilder(helperBuilder("happy")))
	result, err := r.Run(context.Background(), "say hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "Hello world" {
		t.Errorf("text = %q, want %q", result.Text, "Hello world")
	}
	if result.SessionID != "sess-1" {
		t.Errorf("session_id = %q, want %q", result.SessionID, "sess-1")
	}
	if result.CostUSD != 0.05 {
		t.Errorf("cost = %f, want 0.05", result.CostUSD)
	}
	if result.Duration != 1234*time.Millisecond {
		t.Errorf("duration = %v, want %v", result.Duration, 1234*time.Millisecond)
	}
	if result.IsError {
		t.Error("is_error = true, want false")
	}
	if result.Usage.InputTokens != 100 {
		t.Errorf("input_tokens = %d, want 100", result.Usage.InputTokens)
	}
	if result.Usage.OutputTokens != 50 {
		t.Errorf("output_tokens = %d, want 50", result.Usage.OutputTokens)
	}
	if result.Usage.CacheCreationInputTokens != 10 {
		t.Errorf("cache_creation = %d, want 10", result.Usage.CacheCreationInputTokens)
	}
	if result.Usage.CacheReadInputTokens != 20 {
		t.Errorf("cache_read = %d, want 20", result.Usage.CacheReadInputTokens)
	}
}

func TestRunErrorResult(t *testing.T) {
	r := NewRunner(withCommandBuilder(helperBuilder("error_result")))
	result, err := r.Run(context.Background(), "fail please")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("is_error = false, want true")
	}
	if result.Text != "Something failed" {
		t.Errorf("text = %q, want %q", result.Text, "Something failed")
	}
	if result.SessionID != "sess-err" {
		t.Errorf("session_id = %q, want %q", result.SessionID, "sess-err")
	}
}

func TestRunNoResult(t *testing.T) {
	r := NewRunner(withCommandBuilder(helperBuilder("no_result")))
	_, err := r.Run(context.Background(), "hello")
	if err != agentrunner.ErrNoResult {
		t.Errorf("err = %v, want ErrNoResult", err)
	}
}

func TestRunNonZeroExit(t *testing.T) {
	r := NewRunner(withCommandBuilder(helperBuilder("nonzero_exit")))
	_, err := r.Run(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, agentrunner.ErrNonZeroExit) {
		t.Errorf("err = %v, want ErrNonZeroExit", err)
	}
	if !strings.Contains(err.Error(), "fatal error from claude") {
		t.Errorf("err = %v, want to contain stderr output", err)
	}
}

func TestRunTimeout(t *testing.T) {
	r := NewRunner(withCommandBuilder(helperBuilder("slow")))
	_, err := r.Run(context.Background(), "hello", agentrunner.WithTimeout(100*time.Millisecond))
	if err != agentrunner.ErrTimeout {
		t.Errorf("err = %v, want ErrTimeout", err)
	}
}

func TestRunCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	r := NewRunner(withCommandBuilder(helperBuilder("slow")))

	done := make(chan error, 1)
	go func() {
		_, err := r.Run(ctx, "hello")
		done <- err
	}()

	// Give the process a moment to start, then cancel.
	time.Sleep(50 * time.Millisecond)
	cancel()

	err := <-done
	if err != agentrunner.ErrCancelled {
		t.Errorf("err = %v, want ErrCancelled", err)
	}
}

func TestRunSessionIDFromInit(t *testing.T) {
	r := NewRunner(withCommandBuilder(helperBuilder("init_session_only")))
	result, err := r.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The result message has no session_id, so it should fall back to the
	// session_id from the system/init message.
	if result.SessionID != "sess-from-init" {
		t.Errorf("session_id = %q, want %q", result.SessionID, "sess-from-init")
	}
}

func TestRunNotFound(t *testing.T) {
	r := NewRunner(WithBinary("nonexistent-binary-xyz"))
	_, err := r.Run(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, agentrunner.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

// --- Argument building tests ---

func TestBuildArgsMinimal(t *testing.T) {
	opts := &agentrunner.Options{}
	args := buildArgs("hello world", opts)

	expected := []string{"--print", "--output-format", "stream-json", "--verbose", "--", "hello world"}
	if len(args) != len(expected) {
		t.Fatalf("args = %v, want %v", args, expected)
	}
	for i, a := range args {
		if a != expected[i] {
			t.Errorf("args[%d] = %q, want %q", i, a, expected[i])
		}
	}
}

func TestBuildArgsAllCommonOptions(t *testing.T) {
	opts := &agentrunner.Options{
		Model:              "claude-sonnet-4-6",
		SystemPrompt:       "You are helpful",
		AppendSystemPrompt: "Be concise",
		MaxTurns:           5,
		SkipPermissions:    true,
	}
	args := buildArgs("test prompt", opts)

	mustContain := []string{
		"--verbose",
		"--model", "claude-sonnet-4-6",
		"--system-prompt", "You are helpful",
		"--append-system-prompt", "Be concise",
		"--max-turns", "5",
		"--dangerously-skip-permissions",
	}
	joined := strings.Join(args, " ")
	for _, s := range mustContain {
		if !strings.Contains(joined, s) {
			t.Errorf("args missing %q: %v", s, args)
		}
	}

	// Prompt must be last, after "--"
	if args[len(args)-1] != "test prompt" || args[len(args)-2] != "--" {
		t.Errorf("prompt not at end: %v", args)
	}
}

func TestBuildArgsClaudeOptions(t *testing.T) {
	opts := &agentrunner.Options{}
	// Apply Claude-specific options.
	WithAllowedTools("Read", "Write")(opts)
	WithDisallowedTools("Bash")(opts)
	WithMCPConfig("/tmp/mcp.json")(opts)
	WithJSONSchema(`{"type":"object"}`)(opts)
	WithMaxBudgetUSD(1.5)(opts)
	WithResume("sess-123")(opts)

	args := buildArgs("test", opts)
	joined := strings.Join(args, " ")

	mustContain := []string{
		"--allowedTools Read",
		"--allowedTools Write",
		"--disallowedTools Bash",
		"--mcp-config /tmp/mcp.json",
		`--json-schema {"type":"object"}`,
		"--max-budget-usd 1.5",
		"--resume sess-123",
	}
	for _, s := range mustContain {
		if !strings.Contains(joined, s) {
			t.Errorf("args missing %q: %v", s, args)
		}
	}
}

func TestBuildArgsSessionID(t *testing.T) {
	opts := &agentrunner.Options{}
	WithSessionID("my-session-42")(opts)
	args := buildArgs("test", opts)

	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--session-id my-session-42") {
		t.Errorf("args missing --session-id: %v", args)
	}
}

func TestBuildArgsContinue(t *testing.T) {
	opts := &agentrunner.Options{}
	WithContinue()(opts)
	args := buildArgs("test", opts)

	found := false
	for _, a := range args {
		if a == "--continue" {
			found = true
		}
	}
	if !found {
		t.Errorf("args missing --continue: %v", args)
	}
}

func TestBuildArgsIncludePartialMessages(t *testing.T) {
	opts := &agentrunner.Options{}
	WithIncludePartialMessages()(opts)
	args := buildArgs("test", opts)

	found := false
	for _, a := range args {
		if a == "--include-partial-messages" {
			found = true
		}
	}
	if !found {
		t.Errorf("args missing --include-partial-messages: %v", args)
	}
}

// --- Start/Session tests ---

func TestStartHappyPath(t *testing.T) {
	r := NewRunner(withCommandBuilder(helperBuilder("happy")))
	session, err := r.Start(context.Background(), "say hello")
	if err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}

	var messages []agentrunner.Message
	for msg := range session.Messages {
		messages = append(messages, msg)
	}

	result, err := session.Result()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Text != "Hello world" {
		t.Errorf("text = %q, want %q", result.Text, "Hello world")
	}
	if result.SessionID != "sess-1" {
		t.Errorf("session_id = %q, want %q", result.SessionID, "sess-1")
	}

	// Should have received system + assistant + result = 3 messages
	if len(messages) != 3 {
		t.Errorf("got %d messages, want 3", len(messages))
	}
}

func TestStartAbortMidStream(t *testing.T) {
	r := NewRunner(withCommandBuilder(helperBuilder("slow")))
	session, err := r.Start(context.Background(), "long task")
	if err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}

	// Give the process a moment to start, then abort.
	time.Sleep(50 * time.Millisecond)
	session.Abort()

	// Drain messages.
	for range session.Messages {
	}

	_, err = session.Result()
	if err != agentrunner.ErrCancelled {
		t.Errorf("err = %v, want ErrCancelled", err)
	}
}

func TestStartMessagesAndResult(t *testing.T) {
	r := NewRunner(withCommandBuilder(helperBuilder("stream_multi")))
	session, err := r.Start(context.Background(), "test session")
	if err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}

	var types []agentrunner.MessageType
	for msg := range session.Messages {
		types = append(types, msg.Type)
	}

	// Verify ordering.
	if len(types) == 0 {
		t.Fatal("no messages received")
	}
	if types[0] != agentrunner.MessageTypeSystem {
		t.Errorf("first type = %q, want system", types[0])
	}
	if types[len(types)-1] != agentrunner.MessageTypeResult {
		t.Errorf("last type = %q, want result", types[len(types)-1])
	}

	result, err := session.Result()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "Hello world" {
		t.Errorf("text = %q, want %q", result.Text, "Hello world")
	}
}

func TestStartTimeout(t *testing.T) {
	r := NewRunner(withCommandBuilder(helperBuilder("slow")))
	session, err := r.Start(context.Background(), "hello",
		agentrunner.WithTimeout(100*time.Millisecond))
	if err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}

	for range session.Messages {
	}

	_, err = session.Result()
	if err != agentrunner.ErrTimeout {
		t.Errorf("err = %v, want ErrTimeout", err)
	}
}

func TestStartSendNotSupported(t *testing.T) {
	r := NewRunner(withCommandBuilder(helperBuilder("happy")))
	session, err := r.Start(context.Background(), "hello")
	if err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}

	sendErr := session.Send("test input")
	if sendErr != agentrunner.ErrNotSupported {
		t.Errorf("err = %v, want ErrNotSupported", sendErr)
	}

	// Drain to avoid goroutine leak.
	for range session.Messages {
	}
	session.Result()
}

func TestStartNotFound(t *testing.T) {
	r := NewRunner(WithBinary("nonexistent-binary-xyz"))
	_, err := r.Start(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, agentrunner.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestStartOnMessageCallback(t *testing.T) {
	r := NewRunner(withCommandBuilder(helperBuilder("stream_multi")))

	var callbackMessages []agentrunner.Message
	session, err := r.Start(context.Background(), "test callback",
		WithOnMessage(func(msg agentrunner.Message) {
			callbackMessages = append(callbackMessages, msg)
		}),
	)
	if err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}

	var channelMessages []agentrunner.Message
	for msg := range session.Messages {
		channelMessages = append(channelMessages, msg)
	}

	if _, err := session.Result(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Callback should have received the same messages as the channel.
	if len(callbackMessages) != len(channelMessages) {
		t.Fatalf("callback got %d messages, channel got %d", len(callbackMessages), len(channelMessages))
	}

	for i := range callbackMessages {
		if callbackMessages[i].Type != channelMessages[i].Type {
			t.Errorf("message[%d]: callback type %q != channel type %q",
				i, callbackMessages[i].Type, channelMessages[i].Type)
		}
	}
}

func TestStartRawJSON(t *testing.T) {
	r := NewRunner(withCommandBuilder(helperBuilder("stream_multi")))
	session, err := r.Start(context.Background(), "test raw")
	if err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}

	for msg := range session.Messages {
		if len(msg.Raw) == 0 {
			t.Errorf("message type %q has empty Raw field", msg.Type)
		}
	}

	if _, err := session.Result(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStartChannelClosure(t *testing.T) {
	r := NewRunner(withCommandBuilder(helperBuilder("stream_no_result")))
	session, err := r.Start(context.Background(), "partial")
	if err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}

	// Drain messages.
	var count int
	for range session.Messages {
		count++
	}

	// Channel should be closed even without result message.
	if count != 2 {
		t.Errorf("got %d messages, want 2", count)
	}

	// ErrNoResult expected since process exits cleanly but no result message.
	if _, err := session.Result(); err != nil {
		t.Logf("got error (acceptable): %v", err)
	}
}

// --- Message accessor tests ---

func TestMessageAccessors(t *testing.T) {
	// Text accessor via Parsed
	textMsg := agentrunner.Message{
		Type: agentrunner.MessageTypeAssistant,
		Parsed: &StreamMessage{
			Type:    "assistant",
			Content: []ContentBlock{{Type: "text", Text: "hello"}},
		},
	}
	if got := textMsg.Text(); got != "hello" {
		t.Errorf("Text() = %q, want %q", got, "hello")
	}

	// ToolName accessor
	toolMsg := agentrunner.Message{
		Type: agentrunner.MessageTypeAssistant,
		Parsed: &StreamMessage{
			Type:    "assistant",
			Content: []ContentBlock{{Type: "tool_use", Name: "Read"}},
		},
	}
	if got := toolMsg.ToolName(); got != "Read" {
		t.Errorf("ToolName() = %q, want %q", got, "Read")
	}

	// IsError accessor
	errMsg := agentrunner.Message{
		Type: agentrunner.MessageTypeResult,
		Parsed: &StreamMessage{
			Type:    "result",
			IsError: true,
			Result:  "something broke",
		},
	}
	if !errMsg.IsError() {
		t.Error("IsError() = false, want true")
	}
	if got := errMsg.ErrorMessage(); got != "something broke" {
		t.Errorf("ErrorMessage() = %q, want %q", got, "something broke")
	}

	// ParseMessage
	sm, ok := ParseMessage(textMsg)
	if !ok {
		t.Fatal("ParseMessage returned false")
	}
	if sm.Text() != "hello" {
		t.Errorf("ParseMessage().Text() = %q, want %q", sm.Text(), "hello")
	}

	// ParseMessage with wrong type
	wrongMsg := agentrunner.Message{Type: agentrunner.MessageTypeAssistant, Parsed: "not a StreamMessage"}
	_, ok = ParseMessage(wrongMsg)
	if ok {
		t.Error("ParseMessage returned true for wrong type")
	}
}
