// Package codex provides a Runner implementation for invoking the Codex CLI
// programmatically. It implements the common Runner interface using
// `codex exec --json` with newline-delimited JSON output.
package codex

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

	"github.com/driangle/agentrunner/go"
)

// Compile-time interface assertion.
var _ agentrunner.Runner = (*Runner)(nil)

// MinCLIVersion is the minimum supported Codex CLI version.
const MinCLIVersion = "0.118.0"

// commandBuilder creates an *exec.Cmd for the given binary and arguments.
// Inject a custom builder in tests to avoid spawning a real CLI process.
type commandBuilder func(ctx context.Context, name string, args ...string) *exec.Cmd

// RunnerOption configures a Runner.
type RunnerOption func(*Runner)

// WithBinary overrides the CLI binary name (default "codex").
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

// Runner implements agentrunner.Runner for the Codex CLI.
type Runner struct {
	binary       string
	cmdBuilder   commandBuilder
	logger       *slog.Logger
	versionOnce  sync.Once
	versionError error
}

// NewRunner creates a Runner with the given options.
func NewRunner(opts ...RunnerOption) *Runner {
	r := &Runner{binary: "codex"}
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
		// Output is "codex-cli 0.118.0" — extract the version number.
		version := strings.TrimSpace(string(out))
		if i := strings.LastIndex(version, " "); i >= 0 {
			version = version[i+1:]
		}
		if !isVersionAtLeast(version, MinCLIVersion) {
			r.versionError = fmt.Errorf("%w: codex CLI version %s is below minimum %s", agentrunner.ErrNotFound, version, MinCLIVersion)
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

// Start launches a Codex CLI process and returns a Session for full control
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
			return nil, fmt.Errorf("%w: %s", agentrunner.ErrNotFound, r.binary)
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

		var lastText string
		var threadID string
		var turnUsage *TurnUsage
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

			if parsed.Type == "thread.started" && parsed.ThreadID != "" {
				threadID = parsed.ThreadID
			}
			if parsed.Item != nil && parsed.Item.Type == "agent_message" && parsed.Type == "item.completed" {
				lastText = parsed.Item.Text
			}
			if parsed.Type == "turn.completed" && parsed.Usage != nil {
				turnUsage = parsed.Usage
			}
			if parsed.Type == "error" {
				hadError = true
				errorMsg = parsed.Message
			}
			if parsed.Type == "turn.failed" {
				hadError = true
				if parsed.Error != nil {
					errorMsg = parsed.Error.Message
				}
			}

			parsedCopy := parsed
			msg := agentrunner.Message{
				Type:   mapMessageType(&parsed),
				Raw:    lineCopy,
				Parsed: &parsedCopy,
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

		if hadError && turnUsage == nil {
			return &agentrunner.Result{
				Text:      errorMsg,
				IsError:   true,
				SessionID: threadID,
			}, nil
		}

		if turnUsage != nil {
			result := &agentrunner.Result{
				Text:      lastText,
				IsError:   hadError,
				SessionID: threadID,
				Usage: agentrunner.Usage{
					InputTokens:         turnUsage.InputTokens,
					OutputTokens:        turnUsage.OutputTokens,
					CacheReadInputTokens: turnUsage.CachedInputTokens,
				},
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

// Run executes a prompt against the Codex CLI and returns the final result.
func (r *Runner) Run(ctx context.Context, prompt string, opts ...agentrunner.Option) (*agentrunner.Result, error) {
	session, err := r.Start(ctx, prompt, opts...)
	if err != nil {
		return nil, err
	}
	for range session.Messages {
	}
	return session.Result()
}

// Parse parses a single JSONL line from the Codex CLI output.
func Parse(line []byte) (StreamMessage, error) {
	var msg StreamMessage
	if err := json.Unmarshal(line, &msg); err != nil {
		return StreamMessage{}, err
	}
	return msg, nil
}

// ParseMessage extracts the Codex-specific StreamMessage from a common Message.
func ParseMessage(msg agentrunner.Message) (*StreamMessage, bool) {
	if sm, ok := msg.Parsed.(*StreamMessage); ok {
		return sm, true
	}
	return nil, false
}

func mapMessageType(msg *StreamMessage) agentrunner.MessageType {
	switch msg.Type {
	case "thread.started", "turn.started":
		return agentrunner.MessageTypeSystem
	case "item.started":
		if msg.Item != nil && msg.Item.Type == "command_execution" {
			return agentrunner.MessageTypeToolUse
		}
		return agentrunner.MessageTypeAssistant
	case "item.completed":
		if msg.Item != nil {
			switch msg.Item.Type {
			case "agent_message":
				return agentrunner.MessageTypeAssistant
			case "command_execution":
				return agentrunner.MessageTypeToolResult
			}
		}
		return agentrunner.MessageTypeAssistant
	case "turn.completed":
		return agentrunner.MessageTypeResult
	case "error", "turn.failed":
		return agentrunner.MessageTypeError
	default:
		return agentrunner.MessageType(msg.Type)
	}
}

func buildArgs(prompt string, opts *agentrunner.Options) []string {
	args := []string{"exec", "--json"}

	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	if opts.WorkingDir != "" {
		args = append(args, "--cd", opts.WorkingDir)
	}
	if opts.DangerouslySkipPermissions {
		args = append(args, "--dangerously-bypass-approvals-and-sandbox")
	}

	if co := GetCodexOptions(opts); co != nil {
		if co.Sandbox != "" {
			args = append(args, "--sandbox", co.Sandbox)
		}
		if co.Approval != "" {
			args = append(args, "--ask-for-approval", co.Approval)
		}
		if co.OutputSchema != "" {
			args = append(args, "--output-schema", co.OutputSchema)
		}
		for _, img := range co.Images {
			args = append(args, "--image", img)
		}
		if co.Profile != "" {
			args = append(args, "--profile", co.Profile)
		}
		if co.FullAuto {
			args = append(args, "--full-auto")
		}
		if co.Ephemeral {
			args = append(args, "--ephemeral")
		}
		if co.Search {
			args = append(args, "--search")
		}
		for _, dir := range co.AddDirs {
			args = append(args, "--add-dir", dir)
		}
		if co.Resume != "" {
			args = append(args, "resume", co.Resume)
		}
	}

	args = append(args, "--", prompt)
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
