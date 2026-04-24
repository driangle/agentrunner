package gemini

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
		fmt.Println(`{"type":"init","session_id":"sess-1","model":"gemini-2.0-flash","timestamp":"2026-01-01T00:00:00Z"}`)
		fmt.Println(`{"type":"message","role":"user","content":"say hello","timestamp":"2026-01-01T00:00:01Z"}`)
		fmt.Println(`{"type":"message","role":"assistant","content":"Hello world","timestamp":"2026-01-01T00:00:02Z"}`)
		fmt.Println(`{"type":"result","status":"success","stats":{"total_tokens":150,"input_tokens":100,"output_tokens":50,"cached":20,"duration_ms":1200,"tool_calls":0},"timestamp":"2026-01-01T00:00:03Z"}`)
	case "tool_use":
		fmt.Println(`{"type":"init","session_id":"sess-2","model":"gemini-2.0-flash","timestamp":"2026-01-01T00:00:00Z"}`)
		fmt.Println(`{"type":"message","role":"user","content":"list files","timestamp":"2026-01-01T00:00:01Z"}`)
		fmt.Println(`{"type":"message","role":"assistant","content":"Let me check.","timestamp":"2026-01-01T00:00:02Z"}`)
		fmt.Println(`{"type":"tool_use","tool_name":"Bash","tool_id":"bash-1","parameters":{"command":"ls -la"},"timestamp":"2026-01-01T00:00:03Z"}`)
		fmt.Println(`{"type":"tool_result","tool_id":"bash-1","status":"success","output":"file1.txt\nfile2.txt","timestamp":"2026-01-01T00:00:04Z"}`)
		fmt.Println(`{"type":"message","role":"assistant","content":"Found file1.txt and file2.txt","timestamp":"2026-01-01T00:00:05Z"}`)
		fmt.Println(`{"type":"result","status":"success","stats":{"total_tokens":300,"input_tokens":200,"output_tokens":100,"cached":40,"duration_ms":2500,"tool_calls":1},"timestamp":"2026-01-01T00:00:06Z"}`)
	case "error":
		fmt.Println(`{"type":"init","session_id":"sess-err","model":"gemini-2.0-flash","timestamp":"2026-01-01T00:00:00Z"}`)
		fmt.Println(`{"type":"error","severity":"error","message":"something went wrong","timestamp":"2026-01-01T00:00:01Z"}`)
		fmt.Println(`{"type":"result","status":"error","error":{"type":"api_error","message":"something went wrong"},"timestamp":"2026-01-01T00:00:02Z"}`)
	case "error_no_result":
		fmt.Println(`{"type":"init","session_id":"sess-enr","model":"gemini-2.0-flash","timestamp":"2026-01-01T00:00:00Z"}`)
		fmt.Println(`{"type":"error","severity":"error","message":"fatal error","timestamp":"2026-01-01T00:00:01Z"}`)
	case "no_result":
		fmt.Println(`{"type":"init","session_id":"sess-nr","model":"gemini-2.0-flash","timestamp":"2026-01-01T00:00:00Z"}`)
	case "nonzero_exit":
		fmt.Fprintln(os.Stderr, "fatal error from gemini")
		os.Exit(1)
	case "slow":
		time.Sleep(5 * time.Second)
		fmt.Println(`{"type":"result","status":"success","stats":{"total_tokens":10,"input_tokens":5,"output_tokens":5},"timestamp":"2026-01-01T00:00:00Z"}`)
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
	if result.SessionID != "sess-1" {
		t.Errorf("session_id = %q, want %q", result.SessionID, "sess-1")
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
	if result.Duration != 1200*time.Millisecond {
		t.Errorf("duration = %v, want 1200ms", result.Duration)
	}
}

func TestRunWithToolUse(t *testing.T) {
	r := NewRunner(withCommandBuilder(helperBuilder("tool_use")))
	result, err := r.Run(context.Background(), "list files")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "Found file1.txt and file2.txt" {
		t.Errorf("text = %q, want %q", result.Text, "Found file1.txt and file2.txt")
	}
	if result.SessionID != "sess-2" {
		t.Errorf("session_id = %q, want %q", result.SessionID, "sess-2")
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
	if result.SessionID != "sess-err" {
		t.Errorf("session_id = %q, want %q", result.SessionID, "sess-err")
	}
}

func TestRunErrorNoResult(t *testing.T) {
	r := NewRunner(withCommandBuilder(helperBuilder("error_no_result")))
	result, err := r.Run(context.Background(), "fail please")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("is_error = false, want true")
	}
	if result.Text != "fatal error" {
		t.Errorf("text = %q, want %q", result.Text, "fatal error")
	}
	if result.SessionID != "sess-enr" {
		t.Errorf("session_id = %q, want %q", result.SessionID, "sess-enr")
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
	if !strings.Contains(err.Error(), "fatal error from gemini") {
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

	expected := []string{"--output-format", "stream-json", "--prompt", "hello world"}
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
		Model:                      "gemini-2.0-flash",
		DangerouslySkipPermissions: true,
	}
	args := buildArgs("test prompt", opts)

	joined := strings.Join(args, " ")
	mustContain := []string{
		"--output-format stream-json",
		"--model gemini-2.0-flash",
		"--yolo",
		"--prompt test prompt",
	}
	for _, s := range mustContain {
		if !strings.Contains(joined, s) {
			t.Errorf("args missing %q: %v", s, args)
		}
	}
}

func TestBuildArgsGeminiOptions(t *testing.T) {
	opts := &agentrunner.Options{}
	WithApprovalMode("auto_edit")(opts)
	WithSandbox()(opts)
	WithExtensions("ext1", "ext2")(opts)
	WithAllowedTools("Bash", "ReadFile")(opts)
	WithResume("sess-123")(opts)
	WithIncludeDirs("/dir1", "/dir2")(opts)
	WithRawOutput()(opts)

	args := buildArgs("test", opts)
	joined := strings.Join(args, " ")

	mustContain := []string{
		"--approval-mode auto_edit",
		"--sandbox",
		"--extensions ext1",
		"--extensions ext2",
		"--allowed-tools Bash",
		"--allowed-tools ReadFile",
		"--resume sess-123",
		"--include-directories /dir1",
		"--include-directories /dir2",
		"--raw-output",
	}
	for _, s := range mustContain {
		if !strings.Contains(joined, s) {
			t.Errorf("args missing %q: %v", s, args)
		}
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

	// init + message(user) + message(assistant) + result = 4 messages
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
	if result.Text != "Found file1.txt and file2.txt" {
		t.Errorf("text = %q, want %q", result.Text, "Found file1.txt and file2.txt")
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
			Type:    "message",
			Role:    "assistant",
			Content: "hello",
		},
	}
	if got := textMsg.Text(); got != "hello" {
		t.Errorf("Text() = %q, want %q", got, "hello")
	}

	// ToolName accessor
	toolMsg := agentrunner.Message{
		Type: agentrunner.MessageTypeToolUse,
		Parsed: &StreamMessage{
			Type:          "tool_use",
			ToolNameField: "Bash",
			ToolID:        "bash-1",
		},
	}
	if got := toolMsg.ToolName(); got != "Bash" {
		t.Errorf("ToolName() = %q, want %q", got, "Bash")
	}

	// IsError accessor
	errMsg := agentrunner.Message{
		Type: agentrunner.MessageTypeError,
		Parsed: &StreamMessage{
			Type:         "error",
			Severity:     "error",
			MessageField: "something broke",
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
		{"0.1.0", "0.1.0", true},
		{"0.2.0", "0.1.0", true},
		{"1.0.0", "0.1.0", true},
		{"0.0.9", "0.1.0", false},
		{"0.1.1", "0.1.0", true},
	}
	for _, tt := range tests {
		if got := isVersionAtLeast(tt.version, tt.min); got != tt.want {
			t.Errorf("isVersionAtLeast(%q, %q) = %v, want %v", tt.version, tt.min, got, tt.want)
		}
	}
}
