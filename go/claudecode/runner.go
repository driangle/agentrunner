package claudecode

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"

	agentrunner "github.com/driangle/agentrunner-go"
)

// CommandBuilder creates an *exec.Cmd for the given binary and arguments.
// Inject a custom builder in tests to avoid spawning a real CLI process.
type CommandBuilder func(ctx context.Context, name string, args ...string) *exec.Cmd

// RunnerOption configures a Runner.
type RunnerOption func(*Runner)

// WithBinary overrides the CLI binary name (default "claude").
func WithBinary(name string) RunnerOption {
	return func(r *Runner) { r.binary = name }
}

// WithCommandBuilder injects a custom command builder for testing.
func WithCommandBuilder(cb CommandBuilder) RunnerOption {
	return func(r *Runner) { r.cmdBuilder = cb }
}

// WithLogger sets a structured logger for debug output (e.g. command args).
// When nil (the default), no logging is performed.
func WithLogger(l *slog.Logger) RunnerOption {
	return func(r *Runner) { r.logger = l }
}

// Runner implements agentrunner.Runner for Claude Code CLI.
type Runner struct {
	binary     string
	cmdBuilder CommandBuilder
	logger     *slog.Logger
}

// NewRunner creates a Runner with the given options.
func NewRunner(opts ...RunnerOption) *Runner {
	r := &Runner{binary: "claude"}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Start launches a Claude Code CLI process and returns a Session for full
// control over the lifecycle. Messages arrive on session.Messages; the final
// result is available via session.Result().
func (r *Runner) Start(ctx context.Context, prompt string, opts ...agentrunner.Option) *agentrunner.Session {
	var options agentrunner.Options
	for _, o := range opts {
		o(&options)
	}

	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		// cancel is deferred inside the goroutine
		_ = cancel
	}

	ctx, sessionCancel := context.WithCancel(ctx)

	msgCh := make(chan agentrunner.Message)
	resultCh := make(chan agentrunner.ResultOrError, 1)

	go func() {
		defer close(msgCh)
		defer close(resultCh)
		defer sessionCancel()

		args := buildArgs(prompt, &options)

		cmdBuilder := r.cmdBuilder
		if cmdBuilder == nil {
			if _, err := exec.LookPath(r.binary); err != nil {
				resultCh <- agentrunner.ResultOrError{Err: fmt.Errorf("%w: %s", agentrunner.ErrNotFound, r.binary)}
				return
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
			resultCh <- agentrunner.ResultOrError{Err: fmt.Errorf("stdout pipe: %w", err)}
			return
		}

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Start(); err != nil {
			if errors.Is(err, exec.ErrNotFound) {
				resultCh <- agentrunner.ResultOrError{Err: fmt.Errorf("%w: %s", agentrunner.ErrNotFound, r.binary)}
			} else {
				resultCh <- agentrunner.ResultOrError{Err: fmt.Errorf("start: %w", err)}
			}
			return
		}

		onMessage := GetOnMessage(&options)

		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

		var resultMsg *StreamMessage
		var initSessionID string
		var stdoutErrors []string

		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}
			parsed, parseErr := Parse(line)
			if parseErr != nil {
				stdoutErrors = append(stdoutErrors, line)
				continue
			}

			if parsed.Type == "system" && parsed.Subtype == "init" && parsed.SessionID != "" {
				initSessionID = parsed.SessionID
			}
			if parsed.Type == "result" {
				resultMsg = &parsed
			}

			msg := agentrunner.Message{
				Type: mapMessageType(parsed.Type),
				Raw:  []byte(line),
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
					resultCh <- agentrunner.ResultOrError{Err: agentrunner.ErrTimeout}
				} else {
					resultCh <- agentrunner.ResultOrError{Err: agentrunner.ErrCancelled}
				}
				return
			}
		}

		waitErr := cmd.Wait()

		if ctx.Err() != nil {
			if ctx.Err() == context.DeadlineExceeded {
				resultCh <- agentrunner.ResultOrError{Err: agentrunner.ErrTimeout}
			} else {
				resultCh <- agentrunner.ResultOrError{Err: agentrunner.ErrCancelled}
			}
			return
		}

		if resultMsg != nil {
			result := mapResult(resultMsg)
			if result.SessionID == "" && initSessionID != "" {
				result.SessionID = initSessionID
			}
			resultCh <- agentrunner.ResultOrError{Result: result}
			return
		}

		if waitErr != nil {
			var exitErr *exec.ExitError
			if errors.As(waitErr, &exitErr) {
				detail := collectErrorDetail(stderr.String(), stdoutErrors)
				r.logCmdFailure(ctx, exitErr.ExitCode(), stderr.String(), stdoutErrors)
				resultCh <- agentrunner.ResultOrError{Err: fmt.Errorf("%w: exit %d: %s", agentrunner.ErrNonZeroExit, exitErr.ExitCode(), detail)}
			} else {
				resultCh <- agentrunner.ResultOrError{Err: fmt.Errorf("wait: %w", waitErr)}
			}
			return
		}

		if scanErr := scanner.Err(); scanErr != nil {
			resultCh <- agentrunner.ResultOrError{Err: fmt.Errorf("%w: %v", agentrunner.ErrParseError, scanErr)}
			return
		}

		resultCh <- agentrunner.ResultOrError{Err: agentrunner.ErrNoResult}
	}()

	return agentrunner.NewSession(msgCh, resultCh, sessionCancel)
}

// Run executes a prompt against the Claude Code CLI and returns the final result.
// It delegates to Start, drains all messages, and returns the result.
func (r *Runner) Run(ctx context.Context, prompt string, opts ...agentrunner.Option) (*agentrunner.Result, error) {
	session := r.Start(ctx, prompt, opts...)

	// Drain all messages.
	for range session.Messages {
	}

	return session.Result()
}

// RunStream executes a prompt against the Claude Code CLI and streams parsed
// messages as they arrive. It delegates to Start and returns the session's
// message channel. The error channel receives at most one error (the session
// result error, if any) and is then closed.
func (r *Runner) RunStream(ctx context.Context, prompt string, opts ...agentrunner.Option) (<-chan agentrunner.Message, <-chan error) {
	session := r.Start(ctx, prompt, opts...)

	errCh := make(chan error, 1)

	go func() {
		defer close(errCh)
		result, err := session.Result()
		if err != nil {
			errCh <- err
		} else if result != nil && result.IsError {
			// Not a library error — just an error result from the CLI.
			// Don't send on errCh; caller reads it from the result message.
		}
	}()

	return session.Messages, errCh
}

// mapMessageType maps a Claude stream-json type string to the common MessageType.
func mapMessageType(typ string) agentrunner.MessageType {
	switch typ {
	case "system":
		return agentrunner.MessageTypeSystem
	case "assistant":
		return agentrunner.MessageTypeAssistant
	case "user":
		return agentrunner.MessageTypeUser
	case "result":
		return agentrunner.MessageTypeResult
	case "stream_event":
		// Map to assistant since stream events carry assistant content deltas.
		return agentrunner.MessageTypeAssistant
	default:
		return agentrunner.MessageType(typ)
	}
}

func buildArgs(prompt string, opts *agentrunner.Options) []string {
	args := []string{"--print", "--output-format", "stream-json", "--verbose"}

	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	if opts.SystemPrompt != "" {
		args = append(args, "--system-prompt", opts.SystemPrompt)
	}
	if opts.AppendSystemPrompt != "" {
		args = append(args, "--append-system-prompt", opts.AppendSystemPrompt)
	}
	if opts.MaxTurns > 0 {
		args = append(args, "--max-turns", strconv.Itoa(opts.MaxTurns))
	}
	if opts.SkipPermissions {
		args = append(args, "--dangerously-skip-permissions")
	}

	if co := GetClaudeOptions(opts); co != nil {
		for _, t := range co.AllowedTools {
			args = append(args, "--allowedTools", t)
		}
		for _, t := range co.DisallowedTools {
			args = append(args, "--disallowedTools", t)
		}
		if co.MCPConfig != "" {
			args = append(args, "--mcp-config", co.MCPConfig)
		}
		if co.JSONSchema != "" {
			args = append(args, "--json-schema", co.JSONSchema)
		}
		if co.MaxBudgetUSD > 0 {
			args = append(args, "--max-budget-usd", strconv.FormatFloat(co.MaxBudgetUSD, 'f', -1, 64))
		}
		if co.Resume != "" {
			args = append(args, "--resume", co.Resume)
		}
		if co.Continue {
			args = append(args, "--continue")
		}
		if co.SessionID != "" {
			args = append(args, "--session-id", co.SessionID)
		}
		if co.IncludePartialMessages {
			args = append(args, "--include-partial-messages")
		}
	}

	args = append(args, "--", prompt)
	return args
}

func mapResult(msg *StreamMessage) *agentrunner.Result {
	r := &agentrunner.Result{
		Text:       msg.Result,
		IsError:    msg.IsError,
		CostUSD:    msg.TotalCostUSD,
		DurationMs: int64(msg.DurationMs),
		SessionID:  msg.SessionID,
	}
	if msg.Usage != nil {
		r.Usage = agentrunner.Usage{
			InputTokens:              msg.Usage.InputTokens,
			OutputTokens:             msg.Usage.OutputTokens,
			CacheCreationInputTokens: msg.Usage.CacheCreationInputTokens,
			CacheReadInputTokens:     msg.Usage.CacheReadInputTokens,
		}
	}
	return r
}

// logCmd logs the command that is about to be executed, if a logger is set.
func (r *Runner) logCmd(ctx context.Context, cmd *exec.Cmd) {
	if r.logger == nil {
		return
	}
	// Build a shell-friendly command string for easy copy-paste reproduction.
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

// logCmdFailure logs stderr and unparseable stdout lines when the CLI exits
// with a non-zero code, if a logger is set.
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

// collectErrorDetail builds a human-readable error string from stderr and
// any unparseable stdout lines. The CLI sometimes writes errors to stdout
// as plain text rather than stream-json, so we capture both sources.
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
