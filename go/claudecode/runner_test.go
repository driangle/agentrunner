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

func TestBuildArgsMinimal(t *testing.T) {
	opts := &agentrunner.Options{}
	args := buildArgs("hello world", opts)

	expected := []string{"--print", "--output-format", "stream-json", "--", "hello world"}
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
