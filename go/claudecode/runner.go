// Package claudecode provides a Runner implementation for invoking Claude Code CLI
// programmatically. It implements the common Runner interface using claude -p with
// stream-json output format.
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
	"sync"
	"time"

	"github.com/driangle/agentrunner/go"
	"github.com/driangle/agentrunner/go/channel"
)

// Compile-time interface assertion.
var _ agentrunner.Runner = (*Runner)(nil)

// MinCLIVersion is the minimum supported Claude Code CLI version.
const MinCLIVersion = "1.0.12"

// commandBuilder creates an *exec.Cmd for the given binary and arguments.
// Inject a custom builder in tests to avoid spawning a real CLI process.
type commandBuilder func(ctx context.Context, name string, args ...string) *exec.Cmd

// RunnerOption configures a Runner.
type RunnerOption func(*Runner)

// WithBinary overrides the CLI binary name (default "claude").
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

// Runner implements agentrunner.Runner for Claude Code CLI.
type Runner struct {
	binary       string
	cmdBuilder   commandBuilder
	logger       *slog.Logger
	versionOnce  sync.Once
	versionError error
}

// NewRunner creates a Runner with the given options.
func NewRunner(opts ...RunnerOption) *Runner {
	r := &Runner{binary: "claude"}
	for _, o := range opts {
		o(r)
	}
	return r
}

// checkVersion verifies the CLI version is within the supported range.
// It runs once per Runner instance and caches the result.
func (r *Runner) checkVersion() error {
	// Skip version check when using a custom command builder (tests).
	if r.cmdBuilder != nil {
		return nil
	}
	r.versionOnce.Do(func() {
		out, err := exec.Command(r.binary, "--version").Output()
		if err != nil {
			// Don't fail hard — the binary may not be found, which Start handles.
			return
		}
		version := strings.TrimSpace(string(out))
		if !isVersionAtLeast(version, MinCLIVersion) {
			r.versionError = fmt.Errorf("%w: claude CLI version %s is below minimum %s", agentrunner.ErrNotFound, version, MinCLIVersion)
		}
	})
	return r.versionError
}

// isVersionAtLeast returns true if version >= minVersion using simple
// numeric comparison of dot-separated components (e.g., "1.0.12" >= "1.0.12").
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

// Start launches a Claude Code CLI process and returns a Session for full
// control over the lifecycle. Pre-flight errors (version check, binary lookup,
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

	// Pre-flight: version check.
	if err := r.checkVersion(); err != nil {
		cleanup()
		return nil, err
	}

	// Channel setup: create socket, MCP config, resolve binary.
	var chSetup *channelSetup
	if co := GetClaudeOptions(&options); co != nil && co.ChannelEnabled {
		var err error
		chSetup, err = setupChannel(ctx, co)
		if err != nil {
			cleanup()
			return nil, fmt.Errorf("channel setup: %w", err)
		}
		// Point --mcp-config at our merged config.
		co.MCPConfig = chSetup.mcpConfigPath
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

	// Build session options (e.g. send function for channels).
	var sessionOpts []agentrunner.SessionOption
	if chSetup != nil {
		sockPath := chSetup.sockPath
		sessionOpts = append(sessionOpts, agentrunner.WithSendFunc(func(v any) error {
			msg, ok := v.(channel.ChannelMessage)
			if !ok {
				return fmt.Errorf("expected channel.ChannelMessage, got %T", v)
			}
			return channel.SendMessage(ctx, sockPath, msg)
		}))
	}

	// All pre-flight passed. Create session with the streaming goroutine.
	return agentrunner.NewSession(ctx, sessionCancel, func(ctx context.Context, msgCh chan<- agentrunner.Message) (*agentrunner.Result, error) {
		if timeoutCancel != nil {
			defer timeoutCancel()
		}
		if chSetup != nil {
			defer chSetup.cleanup()
		}

		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

		var resultMsg *StreamMessage
		var initSessionID string
		var stdoutErrors []string

		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}
			// Copy the line since scanner reuses the buffer.
			lineCopy := make([]byte, len(line))
			copy(lineCopy, line)

			parsed, parseErr := Parse(lineCopy)
			if parseErr != nil {
				stdoutErrors = append(stdoutErrors, string(lineCopy))
				continue
			}

			if parsed.Type == "system" && parsed.Subtype == "init" && parsed.SessionID != "" {
				initSessionID = parsed.SessionID
			}
			if parsed.Type == "result" {
				resultMsg = &parsed
			}

			parsedCopy := parsed
			msg := agentrunner.Message{
				Type:   mapMessageType(parsed.Type),
				Raw:    lineCopy,
				Parsed: &parsedCopy,
			}

			// Detect channel reply tool calls and remap message type.
			if msg.Type == agentrunner.MessageTypeAssistant && parsedCopy.IsChannelReply() {
				msg.Type = agentrunner.MessageTypeChannelReply
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

		// Check scanner error before cmd.Wait — if the scanner hit an I/O error,
		// we want to report it rather than masking it as ErrNoResult.
		scanErr := scanner.Err()

		waitErr := cmd.Wait()

		if ctx.Err() != nil {
			if ctx.Err() == context.DeadlineExceeded {
				return nil, wrapWithStderr(agentrunner.ErrTimeout, &stderr, stdoutErrors)
			}
			return nil, wrapWithStderr(agentrunner.ErrCancelled, &stderr, stdoutErrors)
		}

		if resultMsg != nil {
			result := mapResult(resultMsg)
			if result.SessionID == "" && initSessionID != "" {
				result.SessionID = initSessionID
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
	}, sessionOpts...), nil
}

// Run executes a prompt against the Claude Code CLI and returns the final result.
// It delegates to Start, drains all messages, and returns the result.
func (r *Runner) Run(ctx context.Context, prompt string, opts ...agentrunner.Option) (*agentrunner.Result, error) {
	session, err := r.Start(ctx, prompt, opts...)
	if err != nil {
		return nil, err
	}

	// Drain all messages.
	for range session.Messages {
	}

	return session.Result()
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
	// --verbose is required with --output-format stream-json.
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
		if co.ChannelEnabled {
			// --channels registers the MCP server as a channel source.
			// --dangerously-load-development-channels marks it as a dev channel,
			// bypassing the production allowlist.
			// Both flags are required: --channels alone fails the allowlist,
			// and --dangerously-load-development-channels alone is ignored in -p mode.
			//
			// NOTE: The channels feature is also gated behind a server-side feature
			// flag in Claude Code. In -p mode the bypass that works in interactive
			// mode is not applied, so channels only work if the flag is enabled on
			// the user's account. See docs/guide/channels.md.
			args = append(args,
				"--channels", "server:agentrunner-channel",
				"--dangerously-load-development-channels", "server:agentrunner-channel",
				"--strict-mcp-config",
			)
		}
	}

	args = append(args, "--", prompt)
	return args
}

func mapResult(msg *StreamMessage) *agentrunner.Result {
	r := &agentrunner.Result{
		Text:      msg.Result,
		IsError:   msg.IsError,
		CostUSD:   msg.TotalCostUSD,
		Duration:  time.Duration(msg.DurationMs) * time.Millisecond,
		SessionID: msg.SessionID,
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

// wrapWithStderr wraps a sentinel error with stderr/stdout diagnostic output
// when available. If there's no diagnostic output, it returns the sentinel as-is,
// preserving errors.Is compatibility.
func wrapWithStderr(sentinel error, stderr *bytes.Buffer, stdoutErrors []string) error {
	if stderr.Len() == 0 && len(stdoutErrors) == 0 {
		return sentinel
	}
	return fmt.Errorf("%w: %s", sentinel, collectErrorDetail(stderr.String(), stdoutErrors))
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
