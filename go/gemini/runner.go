// Package gemini provides a Runner implementation for invoking the Gemini CLI
// programmatically. It implements the common Runner interface using
// `gemini --output-format stream-json` with newline-delimited JSON events.
package gemini

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/driangle/agentrunner/go"
)

// Compile-time interface assertion.
var _ agentrunner.Runner = (*Runner)(nil)

// MinCLIVersion is the minimum supported Gemini CLI version.
const MinCLIVersion = "0.1.0"

// commandBuilder creates an *exec.Cmd for the given binary and arguments.
// Inject a custom builder in tests to avoid spawning a real CLI process.
type commandBuilder func(ctx context.Context, name string, args ...string) *exec.Cmd

// RunnerOption configures a Runner.
type RunnerOption func(*Runner)

// WithBinary overrides the CLI binary name (default "gemini").
func WithBinary(name string) RunnerOption {
	return func(r *Runner) { r.binary = name }
}

// withCommandBuilder injects a custom command builder for testing.
func withCommandBuilder(cb commandBuilder) RunnerOption {
	return func(r *Runner) { r.cmdBuilder = cb }
}

// WithLogger sets a structured logger for debug output (e.g. command args).
// When nil (the default), no logging is performed.
func WithLogger(l *slog.Logger) RunnerOption {
	return func(r *Runner) { r.logger = l }
}

// Runner implements agentrunner.Runner for the Gemini CLI.
type Runner struct {
	binary       string
	cmdBuilder   commandBuilder
	logger       *slog.Logger
	versionOnce  sync.Once
	versionError error
}

// NewRunner creates a Runner with the given options.
func NewRunner(opts ...RunnerOption) *Runner {
	r := &Runner{binary: "gemini"}
	for _, o := range opts {
		o(r)
	}
	return r
}

// checkVersion verifies the CLI version is within the supported range.
// It runs once per Runner instance and caches the result.
func (r *Runner) checkVersion() error {
	if r.cmdBuilder != nil {
		return nil
	}
	r.versionOnce.Do(func() {
		out, err := exec.Command(r.binary, "--version").Output()
		if err != nil {
			return
		}
		version := strings.TrimSpace(string(out))
		// Version output may be prefixed (e.g. "gemini-cli 1.0.0") — extract the last token.
		if i := strings.LastIndex(version, " "); i >= 0 {
			version = version[i+1:]
		}
		if !isVersionAtLeast(version, MinCLIVersion) {
			r.versionError = fmt.Errorf("%w: gemini CLI version %s is below minimum %s", agentrunner.ErrNotFound, version, MinCLIVersion)
		}
	})
	return r.versionError
}

// isVersionAtLeast returns true if version >= minVersion using simple
// numeric comparison of dot-separated components.
func isVersionAtLeast(version, minVersion string) bool {
	parse := func(v string) []int {
		var parts []int
		for _, s := range strings.Split(v, ".") {
			n, _ := strconv.Atoi(s)
			parts = append(parts, n)
		}
		return parts
	}
	v := parse(version)
	min := parse(minVersion)
	for i := 0; i < len(min); i++ {
		vi := 0
		if i < len(v) {
			vi = v[i]
		}
		if vi < min[i] {
			return false
		}
		if vi > min[i] {
			return true
		}
	}
	return true
}

// Start launches a Gemini CLI process and returns a Session for full control
// over the lifecycle. Pre-flight errors (version check, binary lookup,
// process start) are returned immediately. Messages arrive on session.Messages;
// the final result is available via session.Result().
func (r *Runner) Start(ctx context.Context, prompt string, opts ...agentrunner.Option) (*agentrunner.Session, error) {
	var options agentrunner.Options
	for _, o := range opts {
		o(&options)
	}

	var timeoutCancel context.CancelFunc
	if options.Timeout > 0 {
		ctx, timeoutCancel = context.WithTimeout(ctx, options.Timeout)
	}

	ctx, sessionCancel := context.WithCancel(ctx)

	cleanup := func() {
		sessionCancel()
		if timeoutCancel != nil {
			timeoutCancel()
		}
	}

	if err := r.checkVersion(); err != nil {
		cleanup()
		return nil, err
	}

	args := buildArgs(prompt, &options)

	cmdBuilder := r.cmdBuilder
	if cmdBuilder == nil {
		if _, err := exec.LookPath(r.binary); err != nil {
			cleanup()
			return nil, fmt.Errorf("%w: %s binary not found on PATH", agentrunner.ErrNotFound, r.binary)
		}
		cmdBuilder = exec.CommandContext
	}

	cmd := cmdBuilder(ctx, r.binary, args...)
	cmd.Dir = options.WorkingDir

	if len(options.Env) > 0 {
		cmd.Env = cmd.Environ()
		for k, v := range options.Env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}

	r.logCmd(ctx, cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		cleanup()
		if errors.Is(err, exec.ErrNotFound) {
			return nil, fmt.Errorf("%w: %s", agentrunner.ErrNotFound, r.binary)
		}
		return nil, fmt.Errorf("start: %w", err)
	}

	onMessage := GetOnMessage(&options)

	return agentrunner.NewSession(ctx, sessionCancel, func(ctx context.Context, msgCh chan<- agentrunner.Message) (*agentrunner.Result, error) {
		if timeoutCancel != nil {
			defer timeoutCancel()
		}

		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

		var lastAssistantText string
		var sessionID string
		var resultMsg *StreamMessage
		var hadError bool
		var errorMsg string
		var stdoutErrors []string

		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			lineCopy := make([]byte, len(line))
			copy(lineCopy, line)

			parsed, parseErr := Parse(lineCopy)
			if parseErr != nil {
				stdoutErrors = append(stdoutErrors, string(lineCopy))
				continue
			}

			if parsed.Type == "init" && parsed.SessionID != "" {
				sessionID = parsed.SessionID
			}
			if parsed.Type == "message" && parsed.Role == "assistant" && !parsed.Delta {
				lastAssistantText = parsed.Content
			}
			if parsed.Type == "result" {
				resultMsg = &parsed
			}
			if parsed.Type == "error" {
				hadError = true
				errorMsg = parsed.MessageField
			}

			msg := agentrunner.Message{
				Type:   mapMessageType(&parsed),
				Raw:    lineCopy,
				Parsed: &parsed,
			}

			if onMessage != nil {
				onMessage(msg)
			}

			select {
			case msgCh <- msg:
			case <-ctx.Done():
				_ = cmd.Process.Kill()
				_ = cmd.Wait()
				if ctx.Err() == context.DeadlineExceeded {
					return nil, wrapWithStderr(agentrunner.ErrTimeout, &stderr, stdoutErrors)
				}
				return nil, wrapWithStderr(agentrunner.ErrCancelled, &stderr, stdoutErrors)
			}
		}

		scanErr := scanner.Err()
		waitErr := cmd.Wait()

		if ctx.Err() != nil {
			if ctx.Err() == context.DeadlineExceeded {
				return nil, wrapWithStderr(agentrunner.ErrTimeout, &stderr, stdoutErrors)
			}
			return nil, wrapWithStderr(agentrunner.ErrCancelled, &stderr, stdoutErrors)
		}

		if hadError && resultMsg == nil {
			return &agentrunner.Result{
				Text:      errorMsg,
				IsError:   true,
				SessionID: sessionID,
			}, nil
		}

		if resultMsg != nil {
			result := &agentrunner.Result{
				Text:      lastAssistantText,
				IsError:   resultMsg.Status == "error" || hadError,
				SessionID: sessionID,
			}
			if resultMsg.Stats != nil {
				result.Usage = agentrunner.Usage{
					InputTokens:         resultMsg.Stats.InputTokens,
					OutputTokens:        resultMsg.Stats.OutputTokens,
					CacheReadInputTokens: resultMsg.Stats.Cached,
				}
				result.Duration = time.Duration(resultMsg.Stats.DurationMs) * time.Millisecond
			}
			if resultMsg.Status == "error" && resultMsg.Error != nil && result.Text == "" {
				result.Text = resultMsg.Error.Message
			}
			return result, nil
		}

		if scanErr != nil {
			return nil, fmt.Errorf("%w: %v", agentrunner.ErrParseError, scanErr)
		}

		if waitErr != nil {
			var exitErr *exec.ExitError
			if errors.As(waitErr, &exitErr) {
				detail := collectErrorDetail(stderr.String(), stdoutErrors)
				r.logCmdFailure(ctx, exitErr.ExitCode(), stderr.String(), stdoutErrors)
				return nil, fmt.Errorf("%w: exit %d: %s", agentrunner.ErrNonZeroExit, exitErr.ExitCode(), detail)
			}
			return nil, fmt.Errorf("wait: %w", waitErr)
		}

		return nil, wrapWithStderr(agentrunner.ErrNoResult, &stderr, stdoutErrors)
	}), nil
}

// Run executes a prompt against the Gemini CLI and returns the final result.
func (r *Runner) Run(ctx context.Context, prompt string, opts ...agentrunner.Option) (*agentrunner.Result, error) {
	session, err := r.Start(ctx, prompt, opts...)
	if err != nil {
		return nil, err
	}
	for range session.Messages {
	}
	return session.Result()
}

// Parse parses a single JSONL line from the Gemini CLI stream-json output.
func Parse(line []byte) (StreamMessage, error) {
	var msg StreamMessage
	if err := json.Unmarshal(line, &msg); err != nil {
		return StreamMessage{}, err
	}
	return msg, nil
}

// ParseMessage extracts the Gemini-specific StreamMessage from a common Message.
func ParseMessage(msg agentrunner.Message) (*StreamMessage, bool) {
	if sm, ok := msg.Parsed.(*StreamMessage); ok {
		return sm, true
	}
	return nil, false
}

func mapMessageType(msg *StreamMessage) agentrunner.MessageType {
	switch msg.Type {
	case "init":
		return agentrunner.MessageTypeSystem
	case "message":
		if msg.Role == "user" {
			return agentrunner.MessageTypeUser
		}
		return agentrunner.MessageTypeAssistant
	case "tool_use":
		return agentrunner.MessageTypeToolUse
	case "tool_result":
		return agentrunner.MessageTypeToolResult
	case "error":
		return agentrunner.MessageTypeError
	case "result":
		return agentrunner.MessageTypeResult
	default:
		return agentrunner.MessageType(msg.Type)
	}
}

func buildArgs(prompt string, opts *agentrunner.Options) []string {
	args := []string{"--output-format", "stream-json"}

	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	if opts.DangerouslySkipPermissions {
		args = append(args, "--yolo")
	}

	if go_ := GetGeminiOptions(opts); go_ != nil {
		if go_.ApprovalMode != "" {
			args = append(args, "--approval-mode", go_.ApprovalMode)
		}
		if go_.Sandbox {
			args = append(args, "--sandbox")
		}
		for _, ext := range go_.Extensions {
			args = append(args, "--extensions", ext)
		}
		for _, tool := range go_.AllowedTools {
			args = append(args, "--allowed-tools", tool)
		}
		if go_.Resume != "" {
			args = append(args, "--resume", go_.Resume)
		}
		for _, dir := range go_.IncludeDirs {
			args = append(args, "--include-directories", dir)
		}
		if go_.RawOutput {
			args = append(args, "--raw-output")
		}
	}

	args = append(args, "--prompt", prompt)
	return args
}

func (r *Runner) logCmd(ctx context.Context, cmd *exec.Cmd) {
	if r.logger == nil {
		return
	}
	quoted := make([]string, len(cmd.Args))
	for i, a := range cmd.Args {
		if strings.ContainsAny(a, " \t\n\"'\\") {
			quoted[i] = "'" + strings.ReplaceAll(a, "'", "'\\''") + "'"
		} else {
			quoted[i] = a
		}
	}
	r.logger.DebugContext(ctx, "executing CLI command",
		slog.String("cmd", strings.Join(quoted, " ")),
		slog.String("dir", cmd.Dir),
	)
}

func (r *Runner) logCmdFailure(ctx context.Context, exitCode int, stderr string, stdoutErrors []string) {
	if r.logger == nil {
		return
	}
	r.logger.ErrorContext(ctx, "CLI command failed",
		slog.Int("exit_code", exitCode),
		slog.String("stderr", strings.TrimSpace(stderr)),
		slog.Any("stdout_errors", stdoutErrors),
	)
}

func wrapWithStderr(sentinel error, stderr *bytes.Buffer, stdoutErrors []string) error {
	if stderr.Len() == 0 && len(stdoutErrors) == 0 {
		return sentinel
	}
	return fmt.Errorf("%w: %s", sentinel, collectErrorDetail(stderr.String(), stdoutErrors))
}

func collectErrorDetail(stderr string, stdoutErrors []string) string {
	stderr = strings.TrimSpace(stderr)
	var parts []string
	if stderr != "" {
		parts = append(parts, stderr)
	}
	if len(stdoutErrors) > 0 {
		parts = append(parts, strings.Join(stdoutErrors, "\n"))
	}
	if len(parts) == 0 {
		return "unknown error (no output from CLI)"
	}
	return strings.Join(parts, "\n")
}
