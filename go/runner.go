// Package agentrunner defines the common Runner interface for invoking AI coding agents.
// Each runner (Claude Code, Gemini, Codex, Ollama) implements this interface,
// allowing callers to swap runners without changing their code.
package agentrunner

import (
	"context"
	"encoding/json"
	"errors"
	"runtime"
	"time"
)

// Session encapsulates a running agent process. It exposes the read side
// (messages iterable, result, abort) while reserving a send method for future
// write-side support.
type Session struct {
	// Messages receives parsed messages as they arrive. The channel is closed
	// when the agent process finishes or is aborted.
	Messages <-chan Message

	// resultCh receives the final result (or error) once the process completes.
	resultCh chan ResultOrError

	// abort cancels the underlying context, terminating the agent process.
	abort context.CancelFunc
}

// ResultOrError holds either a result or an error from a completed session.
type ResultOrError struct {
	Result *Result
	Err    error
}

// Result blocks until the agent finishes and returns the final result.
func (s *Session) Result() (*Result, error) {
	r, ok := <-s.resultCh
	if !ok {
		return nil, ErrNoResult
	}
	return r.Result, r.Err
}

// Abort terminates the agent process.
func (s *Session) Abort() {
	if s.abort != nil {
		s.abort()
	}
}

// Send is reserved for future write-side support (e.g. permission responses).
// It currently returns ErrNotSupported.
func (s *Session) Send(_ any) error {
	return ErrNotSupported
}

// NewSession creates a Session with the given channels and abort function.
// The session sets a runtime finalizer so that abandoning a session (not draining
// messages, not calling Abort) will still terminate the underlying process.
func NewSession(messages <-chan Message, resultCh chan ResultOrError, abort context.CancelFunc) *Session {
	s := &Session{
		Messages: messages,
		resultCh: resultCh,
		abort:    abort,
	}
	runtime.SetFinalizer(s, func(s *Session) {
		s.Abort()
	})
	return s
}

// Runner executes prompts against an AI coding agent and returns results.
type Runner interface {
	// Start launches an agent process and returns a Session for full control.
	Start(ctx context.Context, prompt string, opts ...Option) *Session

	// Run sends a prompt and blocks until the agent finishes.
	Run(ctx context.Context, prompt string, opts ...Option) (*Result, error)

	// RunStream sends a prompt and returns channels for streaming messages.
	// The message channel emits messages as they arrive and is closed when done.
	// The error channel receives at most one error and is then closed.
	RunStream(ctx context.Context, prompt string, opts ...Option) (<-chan Message, <-chan error)
}

// Option configures a runner invocation using the functional options pattern.
type Option func(*Options)

// Options holds the resolved configuration for a runner invocation.
// All fields are optional — runners provide sensible defaults.
type Options struct {
	// Model name or alias.
	Model string

	// SystemPrompt overrides the default system prompt.
	SystemPrompt string

	// AppendSystemPrompt is appended to the default system prompt.
	AppendSystemPrompt string

	// WorkingDir sets the working directory for the subprocess.
	WorkingDir string

	// Env provides additional environment variables for the subprocess.
	Env map[string]string

	// MaxTurns limits the number of agentic turns.
	MaxTurns int

	// Timeout sets the overall execution timeout.
	Timeout time.Duration

	// SkipPermissions bypasses interactive permission prompts.
	SkipPermissions bool

	// Extra holds runner-specific options keyed by runner-defined types.
	Extra map[any]any
}

// WithModel sets the model name or alias.
func WithModel(model string) Option {
	return func(o *Options) { o.Model = model }
}

// WithSystemPrompt overrides the default system prompt.
func WithSystemPrompt(prompt string) Option {
	return func(o *Options) { o.SystemPrompt = prompt }
}

// WithAppendSystemPrompt appends to the default system prompt.
func WithAppendSystemPrompt(prompt string) Option {
	return func(o *Options) { o.AppendSystemPrompt = prompt }
}

// WithWorkingDir sets the subprocess working directory.
func WithWorkingDir(dir string) Option {
	return func(o *Options) { o.WorkingDir = dir }
}

// WithEnv sets additional environment variables.
func WithEnv(env map[string]string) Option {
	return func(o *Options) { o.Env = env }
}

// WithMaxTurns limits the number of agentic turns.
func WithMaxTurns(n int) Option {
	return func(o *Options) { o.MaxTurns = n }
}

// WithTimeout sets the overall execution timeout.
func WithTimeout(d time.Duration) Option {
	return func(o *Options) { o.Timeout = d }
}

// WithSkipPermissions bypasses interactive permission prompts.
func WithSkipPermissions(skip bool) Option {
	return func(o *Options) { o.SkipPermissions = skip }
}

// Result is the final output from a runner invocation.
type Result struct {
	// Text is the final response text.
	Text string

	// IsError indicates whether the run ended in error.
	IsError bool

	// ExitCode is the process exit code (CLI runners) or 0 (API runners).
	ExitCode int

	// Usage contains token counts.
	Usage Usage

	// CostUSD is the estimated cost in USD (0 for local runners).
	CostUSD float64

	// DurationMs is the wall-clock duration in milliseconds.
	DurationMs int64

	// SessionID identifies the session for resumption.
	SessionID string
}

// Usage holds token consumption counts.
type Usage struct {
	// InputTokens is the number of prompt/input tokens consumed.
	InputTokens int

	// OutputTokens is the number of completion/output tokens generated.
	OutputTokens int

	// CacheCreationInputTokens is the number of tokens written to cache.
	CacheCreationInputTokens int

	// CacheReadInputTokens is the number of tokens read from cache.
	CacheReadInputTokens int
}

// MessageType identifies the kind of streaming message.
type MessageType string

const (
	// MessageTypeSystem is the initial system/init message from the CLI.
	MessageTypeSystem MessageType = "system"
	// MessageTypeAssistant is an assistant response (text, thinking, or tool_use).
	MessageTypeAssistant MessageType = "assistant"
	// MessageTypeUser is a user-originated message (typically tool results).
	MessageTypeUser MessageType = "user"
	// MessageTypeToolUse indicates the assistant is invoking a tool.
	MessageTypeToolUse MessageType = "tool_use"
	// MessageTypeToolResult is the result returned from a tool invocation.
	MessageTypeToolResult MessageType = "tool_result"
	// MessageTypeResult is the final result message with cost/usage/duration.
	MessageTypeResult MessageType = "result"
	// MessageTypeError indicates an error occurred during execution.
	MessageTypeError MessageType = "error"
)

// Message is the unit of streaming output from RunStream.
type Message struct {
	// Type identifies the message kind.
	Type MessageType

	// Raw is the original JSON line for runner-specific parsing.
	Raw json.RawMessage

	// Parsed is an optional runner-specific typed representation of this message.
	// Runners populate this field so callers can access structured data without
	// re-parsing Raw. The concrete type depends on the runner (e.g. *claudecode.StreamMessage).
	Parsed any
}

// Text returns the text content from the message, if available.
// Returns empty string for message types that don't carry text.
func (m Message) Text() string {
	if a, ok := m.Parsed.(interface{ Text() string }); ok {
		return a.Text()
	}
	return ""
}

// Thinking returns reasoning/thinking content from the message, if available.
func (m Message) Thinking() string {
	if a, ok := m.Parsed.(interface{ Thinking() string }); ok {
		return a.Thinking()
	}
	return ""
}

// IsResult reports whether this message is a final result message.
func (m Message) IsResult() bool {
	return m.Type == MessageTypeResult
}

// Sentinel errors for common failure modes.
var (
	// ErrNotFound indicates the runner binary or API endpoint is not reachable.
	ErrNotFound = errors.New("runner not found")

	// ErrTimeout indicates execution exceeded the configured timeout.
	ErrTimeout = errors.New("execution timed out")

	// ErrNonZeroExit indicates the CLI process exited with a non-zero code.
	ErrNonZeroExit = errors.New("non-zero exit code")

	// ErrParseError indicates failure to parse runner output.
	ErrParseError = errors.New("failed to parse output")

	// ErrCancelled indicates execution was cancelled by the caller.
	ErrCancelled = errors.New("execution cancelled")

	// ErrNoResult indicates the stream ended without a result message.
	ErrNoResult = errors.New("no result in output")

	// ErrHTTPError indicates an HTTP API returned a non-OK status code.
	ErrHTTPError = errors.New("HTTP error")

	// ErrNotSupported indicates the operation is not yet supported.
	ErrNotSupported = errors.New("not yet supported")
)
