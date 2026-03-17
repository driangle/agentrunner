package claudecode

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"

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

// Runner implements agentrunner.Runner for Claude Code CLI.
type Runner struct {
	binary     string
	cmdBuilder CommandBuilder
}

// NewRunner creates a Runner with the given options.
func NewRunner(opts ...RunnerOption) *Runner {
	r := &Runner{binary: "claude"}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Run executes a prompt against the Claude Code CLI and returns the final result.
func (r *Runner) Run(ctx context.Context, prompt string, opts ...agentrunner.Option) (*agentrunner.Result, error) {
	var options agentrunner.Options
	for _, o := range opts {
		o(&options)
	}

	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}

	args := buildArgs(prompt, &options)

	cmdBuilder := r.cmdBuilder
	if cmdBuilder == nil {
		if _, err := exec.LookPath(r.binary); err != nil {
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

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, fmt.Errorf("%w: %s", agentrunner.ErrNotFound, r.binary)
		}
		return nil, fmt.Errorf("start: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	var resultMsg *StreamMessage
	var initSessionID string
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		msg, parseErr := Parse(line)
		if parseErr != nil {
			continue
		}
		if msg.Type == "system" && msg.Subtype == "init" && msg.SessionID != "" {
			initSessionID = msg.SessionID
		}
		if msg.Type == "result" {
			resultMsg = &msg
		}
	}

	waitErr := cmd.Wait()

	// Check errors in priority order.
	if ctx.Err() != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, agentrunner.ErrTimeout
		}
		return nil, agentrunner.ErrCancelled
	}

	if waitErr != nil {
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			return nil, fmt.Errorf("%w: exit %d: %s", agentrunner.ErrNonZeroExit, exitErr.ExitCode(), stderr.String())
		}
		return nil, fmt.Errorf("wait: %w", waitErr)
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return nil, fmt.Errorf("%w: %v", agentrunner.ErrParseError, scanErr)
	}

	if resultMsg == nil {
		return nil, agentrunner.ErrNoResult
	}

	result := mapResult(resultMsg)
	// Fall back to session ID from the system/init message if the result
	// message didn't include one.
	if result.SessionID == "" && initSessionID != "" {
		result.SessionID = initSessionID
	}
	return result, nil
}

// RunStream executes a prompt against the Claude Code CLI and streams parsed
// messages as they arrive. The message channel emits each parsed message in
// order and is closed after the final result message or on error. The error
// channel receives at most one error and is then closed.
//
// If an OnMessage callback was provided via options, it is also called for
// each message before it is sent on the channel.
func (r *Runner) RunStream(ctx context.Context, prompt string, opts ...agentrunner.Option) (<-chan agentrunner.Message, <-chan error) {
	msgCh := make(chan agentrunner.Message)
	errCh := make(chan error, 1)

	var options agentrunner.Options
	for _, o := range opts {
		o(&options)
	}

	var cancel context.CancelFunc
	if options.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
	}

	go func() {
		if cancel != nil {
			defer cancel()
		}
		defer close(msgCh)
		defer close(errCh)

		args := buildArgs(prompt, &options)

		cmdBuilder := r.cmdBuilder
		if cmdBuilder == nil {
			if _, err := exec.LookPath(r.binary); err != nil {
				errCh <- fmt.Errorf("%w: %s", agentrunner.ErrNotFound, r.binary)
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

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			errCh <- fmt.Errorf("stdout pipe: %w", err)
			return
		}

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Start(); err != nil {
			if errors.Is(err, exec.ErrNotFound) {
				errCh <- fmt.Errorf("%w: %s", agentrunner.ErrNotFound, r.binary)
			} else {
				errCh <- fmt.Errorf("start: %w", err)
			}
			return
		}

		// Get the OnMessage callback if set.
		onMessage := GetOnMessage(&options)

		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}
			parsed, parseErr := Parse(line)
			if parseErr != nil {
				continue
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
				// Context cancelled while sending; stop streaming.
				_ = cmd.Process.Kill()
				_ = cmd.Wait()
				if ctx.Err() == context.DeadlineExceeded {
					errCh <- agentrunner.ErrTimeout
				} else {
					errCh <- agentrunner.ErrCancelled
				}
				return
			}
		}

		waitErr := cmd.Wait()

		if ctx.Err() != nil {
			if ctx.Err() == context.DeadlineExceeded {
				errCh <- agentrunner.ErrTimeout
			} else {
				errCh <- agentrunner.ErrCancelled
			}
			return
		}

		if waitErr != nil {
			var exitErr *exec.ExitError
			if errors.As(waitErr, &exitErr) {
				errCh <- fmt.Errorf("%w: exit %d: %s", agentrunner.ErrNonZeroExit, exitErr.ExitCode(), stderr.String())
			} else {
				errCh <- fmt.Errorf("wait: %w", waitErr)
			}
			return
		}

		if scanErr := scanner.Err(); scanErr != nil {
			errCh <- fmt.Errorf("%w: %v", agentrunner.ErrParseError, scanErr)
		}
	}()

	return msgCh, errCh
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
	args := []string{"--print", "--output-format", "stream-json"}

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
