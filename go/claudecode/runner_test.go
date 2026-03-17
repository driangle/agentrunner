package claudecode

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	agentrunner "github.com/driangle/agentrunner-go"
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

// helperBuilder returns a CommandBuilder that re-executes the test binary
// as a helper process with the given mode.
func helperBuilder(mode string) CommandBuilder {
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
	r := NewRunner(WithCommandBuilder(helperBuilder("happy")))
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
	if result.DurationMs != 1234 {
		t.Errorf("duration_ms = %d, want 1234", result.DurationMs)
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
	r := NewRunner(WithCommandBuilder(helperBuilder("error_result")))
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
	r := NewRunner(WithCommandBuilder(helperBuilder("no_result")))
	_, err := r.Run(context.Background(), "hello")
	if err != agentrunner.ErrNoResult {
		t.Errorf("err = %v, want ErrNoResult", err)
	}
}

func TestRunNonZeroExit(t *testing.T) {
	r := NewRunner(WithCommandBuilder(helperBuilder("nonzero_exit")))
	_, err := r.Run(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), agentrunner.ErrNonZeroExit.Error()) {
		t.Errorf("err = %v, want to contain ErrNonZeroExit", err)
	}
	if !strings.Contains(err.Error(), "fatal error from claude") {
		t.Errorf("err = %v, want to contain stderr output", err)
	}
}

func TestRunTimeout(t *testing.T) {
	r := NewRunner(WithCommandBuilder(helperBuilder("slow")))
	_, err := r.Run(context.Background(), "hello", agentrunner.WithTimeout(100*time.Millisecond))
	if err != agentrunner.ErrTimeout {
		t.Errorf("err = %v, want ErrTimeout", err)
	}
}

func TestRunCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	r := NewRunner(WithCommandBuilder(helperBuilder("slow")))

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
	r := NewRunner(WithCommandBuilder(helperBuilder("init_session_only")))
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
	if !strings.Contains(err.Error(), agentrunner.ErrNotFound.Error()) {
		t.Errorf("err = %v, want to contain ErrNotFound", err)
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
	WithAllowedTools("Read", "Write")(&agentrunner.Options{})
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
	WithContinue(true)(opts)
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
	WithIncludePartialMessages(true)(opts)
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

// --- RunStream tests ---

func TestRunStreamHappyPath(t *testing.T) {
	r := NewRunner(WithCommandBuilder(helperBuilder("stream_multi")))
	msgCh, errCh := r.RunStream(context.Background(), "say hello")

	var messages []agentrunner.Message
	for msg := range msgCh {
		messages = append(messages, msg)
	}

	err := <-errCh
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(messages) != 5 {
		t.Fatalf("got %d messages, want 5", len(messages))
	}

	// First message should be system.
	if messages[0].Type != agentrunner.MessageTypeSystem {
		t.Errorf("messages[0].Type = %q, want %q", messages[0].Type, agentrunner.MessageTypeSystem)
	}

	// Last message should be result.
	last := messages[len(messages)-1]
	if last.Type != agentrunner.MessageTypeResult {
		t.Errorf("last message type = %q, want %q", last.Type, agentrunner.MessageTypeResult)
	}

	// Middle messages should be assistant (stream_event maps to assistant).
	for _, msg := range messages[1 : len(messages)-1] {
		if msg.Type != agentrunner.MessageTypeAssistant {
			t.Errorf("intermediate message type = %q, want %q", msg.Type, agentrunner.MessageTypeAssistant)
		}
	}
}

func TestRunStreamMessageOrdering(t *testing.T) {
	r := NewRunner(WithCommandBuilder(helperBuilder("stream_multi")))
	msgCh, errCh := r.RunStream(context.Background(), "test ordering")

	var types []agentrunner.MessageType
	for msg := range msgCh {
		types = append(types, msg.Type)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify result is always last.
	if len(types) == 0 {
		t.Fatal("no messages received")
	}
	if types[len(types)-1] != agentrunner.MessageTypeResult {
		t.Errorf("last type = %q, want %q", types[len(types)-1], agentrunner.MessageTypeResult)
	}

	// Verify system is first.
	if types[0] != agentrunner.MessageTypeSystem {
		t.Errorf("first type = %q, want %q", types[0], agentrunner.MessageTypeSystem)
	}
}

func TestRunStreamChannelClosure(t *testing.T) {
	r := NewRunner(WithCommandBuilder(helperBuilder("stream_no_result")))
	msgCh, errCh := r.RunStream(context.Background(), "partial")

	// Drain messages.
	var count int
	for range msgCh {
		count++
	}

	// Channel should be closed even without result message.
	if count != 2 {
		t.Errorf("got %d messages, want 2", count)
	}

	// No error expected — process exits cleanly, just no result message.
	if err := <-errCh; err != nil {
		t.Logf("got error (acceptable): %v", err)
	}
}

func TestRunStreamTimeout(t *testing.T) {
	r := NewRunner(WithCommandBuilder(helperBuilder("slow")))
	msgCh, errCh := r.RunStream(context.Background(), "hello",
		agentrunner.WithTimeout(100*time.Millisecond))

	// Drain messages.
	for range msgCh {
	}

	err := <-errCh
	if err != agentrunner.ErrTimeout {
		t.Errorf("err = %v, want ErrTimeout", err)
	}
}

func TestRunStreamCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	r := NewRunner(WithCommandBuilder(helperBuilder("slow")))
	msgCh, errCh := r.RunStream(ctx, "hello")

	// Give the process a moment to start, then cancel.
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Drain messages.
	for range msgCh {
	}

	err := <-errCh
	if err != agentrunner.ErrCancelled {
		t.Errorf("err = %v, want ErrCancelled", err)
	}
}

func TestRunStreamNonZeroExit(t *testing.T) {
	r := NewRunner(WithCommandBuilder(helperBuilder("nonzero_exit")))
	msgCh, errCh := r.RunStream(context.Background(), "hello")

	// Drain messages.
	for range msgCh {
	}

	err := <-errCh
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), agentrunner.ErrNonZeroExit.Error()) {
		t.Errorf("err = %v, want to contain ErrNonZeroExit", err)
	}
}

func TestRunStreamNotFound(t *testing.T) {
	r := NewRunner(WithBinary("nonexistent-binary-xyz"))
	msgCh, errCh := r.RunStream(context.Background(), "hello")

	// Drain messages.
	for range msgCh {
	}

	err := <-errCh
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), agentrunner.ErrNotFound.Error()) {
		t.Errorf("err = %v, want to contain ErrNotFound", err)
	}
}

func TestRunStreamOnMessageCallback(t *testing.T) {
	r := NewRunner(WithCommandBuilder(helperBuilder("stream_multi")))

	var callbackMessages []agentrunner.Message
	msgCh, errCh := r.RunStream(context.Background(), "test callback",
		WithOnMessage(func(msg agentrunner.Message) {
			callbackMessages = append(callbackMessages, msg)
		}),
	)

	var channelMessages []agentrunner.Message
	for msg := range msgCh {
		channelMessages = append(channelMessages, msg)
	}

	if err := <-errCh; err != nil {
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

func TestRunStreamRawJSON(t *testing.T) {
	r := NewRunner(WithCommandBuilder(helperBuilder("stream_multi")))
	msgCh, errCh := r.RunStream(context.Background(), "test raw")

	for msg := range msgCh {
		if len(msg.Raw) == 0 {
			t.Errorf("message type %q has empty Raw field", msg.Type)
		}
	}

	if err := <-errCh; err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Session (Start) tests ---

func TestStartHappyPath(t *testing.T) {
	r := NewRunner(WithCommandBuilder(helperBuilder("happy")))
	session := r.Start(context.Background(), "say hello")

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
	r := NewRunner(WithCommandBuilder(helperBuilder("slow")))
	session := r.Start(context.Background(), "long task")

	// Give the process a moment to start, then abort.
	time.Sleep(50 * time.Millisecond)
	session.Abort()

	// Drain messages.
	for range session.Messages {
	}

	_, err := session.Result()
	if err != agentrunner.ErrCancelled {
		t.Errorf("err = %v, want ErrCancelled", err)
	}
}

func TestStartMessagesAndResult(t *testing.T) {
	r := NewRunner(WithCommandBuilder(helperBuilder("stream_multi")))
	session := r.Start(context.Background(), "test session")

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
	r := NewRunner(WithCommandBuilder(helperBuilder("slow")))
	session := r.Start(context.Background(), "hello",
		agentrunner.WithTimeout(100*time.Millisecond))

	for range session.Messages {
	}

	_, err := session.Result()
	if err != agentrunner.ErrTimeout {
		t.Errorf("err = %v, want ErrTimeout", err)
	}
}

func TestStartSendNotSupported(t *testing.T) {
	r := NewRunner(WithCommandBuilder(helperBuilder("happy")))
	session := r.Start(context.Background(), "hello")

	err := session.Send("test input")
	if err != agentrunner.ErrNotSupported {
		t.Errorf("err = %v, want ErrNotSupported", err)
	}

	// Drain to avoid goroutine leak.
	for range session.Messages {
	}
	session.Result()
}

func TestStartNotFound(t *testing.T) {
	r := NewRunner(WithBinary("nonexistent-binary-xyz"))
	session := r.Start(context.Background(), "hello")

	// Drain messages (should be none).
	for range session.Messages {
	}

	_, err := session.Result()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), agentrunner.ErrNotFound.Error()) {
		t.Errorf("err = %v, want to contain ErrNotFound", err)
	}
}
