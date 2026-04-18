package codex

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/driangle/agentrunner/go"
)

// TestHelperProcess is invoked when the test binary is re-executed with
// GO_HELPER_PROCESS=1. It writes canned output to stdout based on
// GO_HELPER_MODE and exits with GO_HELPER_EXIT code.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_HELPER_PROCESS") != "1" {
		return
	}
	mode := os.Getenv("GO_HELPER_MODE")
	switch mode {
	case "happy":
		fmt.Println(`{"type":"thread.started","thread_id":"tid-1"}`)
		fmt.Println(`{"type":"turn.started"}`)
		fmt.Println(`{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"Hello world"}}`)
		fmt.Println(`{"type":"turn.completed","usage":{"input_tokens":100,"cached_input_tokens":20,"output_tokens":50}}`)
	case "tool_use":
		fmt.Println(`{"type":"thread.started","thread_id":"tid-2"}`)
		fmt.Println(`{"type":"turn.started"}`)
		fmt.Println(`{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"Let me check."}}`)
		fmt.Println(`{"type":"item.started","item":{"id":"item_1","type":"command_execution","command":"ls -la","aggregated_output":"","exit_code":null,"status":"in_progress"}}`)
		fmt.Println(`{"type":"item.completed","item":{"id":"item_1","type":"command_execution","command":"ls -la","aggregated_output":"total 0\nfile.txt\n","exit_code":0,"status":"completed"}}`)
		fmt.Println(`{"type":"item.completed","item":{"id":"item_2","type":"agent_message","text":"Found file.txt"}}`)
		fmt.Println(`{"type":"turn.completed","usage":{"input_tokens":200,"cached_input_tokens":40,"output_tokens":80}}`)
	case "error":
		fmt.Println(`{"type":"thread.started","thread_id":"tid-err"}`)
		fmt.Println(`{"type":"turn.started"}`)
		fmt.Println(`{"type":"error","message":"something went wrong"}`)
		fmt.Println(`{"type":"turn.failed","error":{"message":"something went wrong"}}`)
	case "no_result":
		fmt.Println(`{"type":"thread.started","thread_id":"tid-nr"}`)
		fmt.Println(`{"type":"turn.started"}`)
	case "nonzero_exit":
		fmt.Fprintln(os.Stderr, "fatal error from codex")
		os.Exit(1)
	case "slow":
		time.Sleep(5 * time.Second)
		fmt.Println(`{"type":"turn.completed","usage":{"input_tokens":10,"output_tokens":5}}`)
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

// --- Run tests ---

func TestRunHappyPath(t *testing.T) {
	r := NewRunner(withCommandBuilder(helperBuilder("happy")))
	result, err := r.Run(context.Background(), "say hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "Hello world" {
		t.Errorf("text = %q, want %q", result.Text, "Hello world")
	}
	if result.SessionID != "tid-1" {
		t.Errorf("session_id = %q, want %q", result.SessionID, "tid-1")
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
	if result.Usage.CacheReadInputTokens != 20 {
		t.Errorf("cache_read = %d, want 20", result.Usage.CacheReadInputTokens)
	}
}

func TestRunWithToolUse(t *testing.T) {
	r := NewRunner(withCommandBuilder(helperBuilder("tool_use")))
	result, err := r.Run(context.Background(), "list files")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "Found file.txt" {
		t.Errorf("text = %q, want %q", result.Text, "Found file.txt")
	}
	if result.SessionID != "tid-2" {
		t.Errorf("session_id = %q, want %q", result.SessionID, "tid-2")
	}
	if result.Usage.InputTokens != 200 {
		t.Errorf("input_tokens = %d, want 200", result.Usage.InputTokens)
	}
}

func TestRunErrorResult(t *testing.T) {
	r := NewRunner(withCommandBuilder(helperBuilder("error")))
	result, err := r.Run(context.Background(), "fail please")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("is_error = false, want true")
	}
	if result.Text != "something went wrong" {
		t.Errorf("text = %q, want %q", result.Text, "something went wrong")
	}
	if result.SessionID != "tid-err" {
		t.Errorf("session_id = %q, want %q", result.SessionID, "tid-err")
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
	if !strings.Contains(err.Error(), "fatal error from codex") {
		t.Errorf("err = %v, want to contain stderr output", err)
	}
}

func TestRunTimeout(t *testing.T) {
	r := NewRunner(withCommandBuilder(helperBuilder("slow")))
	_, err := r.Run(context.Background(), "hello", agentrunner.WithTimeout(100*time.Millisecond))
	if !errors.Is(err, agentrunner.ErrTimeout) {
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

	time.Sleep(50 * time.Millisecond)
	cancel()

	err := <-done
	if !errors.Is(err, agentrunner.ErrCancelled) {
		t.Errorf("err = %v, want ErrCancelled", err)
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

	expected := []string{"exec", "--json", "--", "hello world"}
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
		Model:                      "o3",
		WorkingDir:                 "/tmp/project",
		DangerouslySkipPermissions: true,
	}
	args := buildArgs("test prompt", opts)

	joined := strings.Join(args, " ")
	mustContain := []string{
		"exec", "--json",
		"--model o3",
		"--cd /tmp/project",
		"--dangerously-bypass-approvals-and-sandbox",
	}
	for _, s := range mustContain {
		if !strings.Contains(joined, s) {
			t.Errorf("args missing %q: %v", s, args)
		}
	}

	if args[len(args)-1] != "test prompt" || args[len(args)-2] != "--" {
		t.Errorf("prompt not at end: %v", args)
	}
}

func TestBuildArgsCodexOptions(t *testing.T) {
	opts := &agentrunner.Options{}
	WithSandbox("read-only")(opts)
	WithApproval("never")(opts)
	WithOutputSchema("/tmp/schema.json")(opts)
	WithImages("/tmp/a.png", "/tmp/b.png")(opts)
	WithProfile("fast")(opts)
	WithFullAuto()(opts)
	WithEphemeral()(opts)
	WithSearch()(opts)
	WithAddDirs("/extra1", "/extra2")(opts)

	args := buildArgs("test", opts)
	joined := strings.Join(args, " ")

	mustContain := []string{
		"--sandbox read-only",
		"--ask-for-approval never",
		"--output-schema /tmp/schema.json",
		"--image /tmp/a.png",
		"--image /tmp/b.png",
		"--profile fast",
		"--full-auto",
		"--ephemeral",
		"--search",
		"--add-dir /extra1",
		"--add-dir /extra2",
	}
	for _, s := range mustContain {
		if !strings.Contains(joined, s) {
			t.Errorf("args missing %q: %v", s, args)
		}
	}
}

func TestBuildArgsResume(t *testing.T) {
	opts := &agentrunner.Options{}
	WithResume("sess-abc-123")(opts)
	args := buildArgs("continue", opts)

	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "resume sess-abc-123") {
		t.Errorf("args missing resume: %v", args)
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
	if result.SessionID != "tid-1" {
		t.Errorf("session_id = %q, want %q", result.SessionID, "tid-1")
	}

	// thread.started + turn.started + item.completed + turn.completed = 4 messages
	if len(messages) != 4 {
		t.Errorf("got %d messages, want 4", len(messages))
	}
}

func TestStartMessagesAndResult(t *testing.T) {
	r := NewRunner(withCommandBuilder(helperBuilder("tool_use")))
	session, err := r.Start(context.Background(), "list files")
	if err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}

	var types []agentrunner.MessageType
	for msg := range session.Messages {
		types = append(types, msg.Type)
	}

	if len(types) == 0 {
		t.Fatal("no messages received")
	}
	if types[0] != agentrunner.MessageTypeSystem {
		t.Errorf("first type = %q, want system", types[0])
	}
	if types[len(types)-1] != agentrunner.MessageTypeResult {
		t.Errorf("last type = %q, want result", types[len(types)-1])
	}

	// Verify tool_use and tool_result types are present.
	var hasToolUse, hasToolResult bool
	for _, typ := range types {
		if typ == agentrunner.MessageTypeToolUse {
			hasToolUse = true
		}
		if typ == agentrunner.MessageTypeToolResult {
			hasToolResult = true
		}
	}
	if !hasToolUse {
		t.Error("missing tool_use message type")
	}
	if !hasToolResult {
		t.Error("missing tool_result message type")
	}

	result, err := session.Result()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "Found file.txt" {
		t.Errorf("text = %q, want %q", result.Text, "Found file.txt")
	}
}

func TestStartAbortMidStream(t *testing.T) {
	r := NewRunner(withCommandBuilder(helperBuilder("slow")))
	session, err := r.Start(context.Background(), "long task")
	if err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	session.Abort()

	for range session.Messages {
	}

	_, err = session.Result()
	if !errors.Is(err, agentrunner.ErrCancelled) {
		t.Errorf("err = %v, want ErrCancelled", err)
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
	if !errors.Is(err, agentrunner.ErrTimeout) {
		t.Errorf("err = %v, want ErrTimeout", err)
	}
}

func TestStartOnMessageCallback(t *testing.T) {
	r := NewRunner(withCommandBuilder(helperBuilder("happy")))

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
	r := NewRunner(withCommandBuilder(helperBuilder("happy")))
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

	for range session.Messages {
	}
	session.Result()
}

// --- Message accessor tests ---

func TestMessageAccessors(t *testing.T) {
	// Text accessor via Parsed
	textMsg := agentrunner.Message{
		Type: agentrunner.MessageTypeAssistant,
		Parsed: &StreamMessage{
			Type: "item.completed",
			Item: &Item{Type: "agent_message", Text: "hello"},
		},
	}
	if got := textMsg.Text(); got != "hello" {
		t.Errorf("Text() = %q, want %q", got, "hello")
	}

	// ToolName accessor
	toolMsg := agentrunner.Message{
		Type: agentrunner.MessageTypeToolUse,
		Parsed: &StreamMessage{
			Type: "item.started",
			Item: &Item{Type: "command_execution", Command: "ls -la"},
		},
	}
	if got := toolMsg.ToolName(); got != "ls -la" {
		t.Errorf("ToolName() = %q, want %q", got, "ls -la")
	}

	// IsError accessor
	errMsg := agentrunner.Message{
		Type: agentrunner.MessageTypeError,
		Parsed: &StreamMessage{
			Type:    "error",
			Message: "something broke",
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

// --- Version check tests ---

func TestIsVersionAtLeast(t *testing.T) {
	tests := []struct {
		version, min string
		want         bool
	}{
		{"0.118.0", "0.118.0", true},
		{"0.119.0", "0.118.0", true},
		{"1.0.0", "0.118.0", true},
		{"0.117.0", "0.118.0", false},
		{"0.118.1", "0.118.0", true},
	}
	for _, tt := range tests {
		if got := isVersionAtLeast(tt.version, tt.min); got != tt.want {
			t.Errorf("isVersionAtLeast(%q, %q) = %v, want %v", tt.version, tt.min, got, tt.want)
		}
	}
}
