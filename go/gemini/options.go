package gemini

import "github.com/driangle/agentrunner/go"

// GeminiOptions holds Gemini CLI-specific configuration that extends the common Options.
type GeminiOptions struct {
	// ApprovalMode sets the approval behavior: "default", "auto_edit", "yolo", or "plan".
	ApprovalMode string

	// Sandbox enables sandbox mode.
	Sandbox bool

	// Extensions lists extensions to use (all by default).
	Extensions []string

	// AllowedTools lists tools allowed without confirmation.
	AllowedTools []string

	// Resume is a session ID or "latest" to resume.
	Resume string

	// IncludeDirs lists additional directories to add to workspace context.
	IncludeDirs []string

	// RawOutput disables output sanitization.
	RawOutput bool
}

// WithApprovalMode sets the approval behavior.
func WithApprovalMode(mode string) agentrunner.Option {
	return func(o *agentrunner.Options) {
		getGeminiOpts(o).ApprovalMode = mode
	}
}

// WithSandbox enables sandbox mode.
func WithSandbox() agentrunner.Option {
	return func(o *agentrunner.Options) {
		getGeminiOpts(o).Sandbox = true
	}
}

// WithExtensions sets the extensions to use.
func WithExtensions(exts ...string) agentrunner.Option {
	return func(o *agentrunner.Options) {
		getGeminiOpts(o).Extensions = exts
	}
}

// WithAllowedTools sets tools allowed without confirmation.
func WithAllowedTools(tools ...string) agentrunner.Option {
	return func(o *agentrunner.Options) {
		getGeminiOpts(o).AllowedTools = tools
	}
}

// WithResume sets the session ID to resume.
func WithResume(sessionID string) agentrunner.Option {
	return func(o *agentrunner.Options) {
		getGeminiOpts(o).Resume = sessionID
	}
}

// WithIncludeDirs sets additional directories to add to workspace context.
func WithIncludeDirs(dirs ...string) agentrunner.Option {
	return func(o *agentrunner.Options) {
		getGeminiOpts(o).IncludeDirs = dirs
	}
}

// WithRawOutput disables output sanitization.
func WithRawOutput() agentrunner.Option {
	return func(o *agentrunner.Options) {
		getGeminiOpts(o).RawOutput = true
	}
}

// OnMessageFunc is a callback invoked for each streaming message.
type OnMessageFunc func(agentrunner.Message)

type onMessageKey struct{}

// WithOnMessage sets a callback that is invoked for each streaming message
// during Start/Run. The callback is called before the message is sent on the
// channel, so it can be used for logging, progress display, etc.
func WithOnMessage(fn OnMessageFunc) agentrunner.Option {
	return func(o *agentrunner.Options) {
		o.SetExtra(onMessageKey{}, fn)
	}
}

// GetOnMessage extracts the OnMessage callback from resolved Options.
// Returns nil if no callback was set.
func GetOnMessage(o *agentrunner.Options) OnMessageFunc {
	v, ok := o.GetExtra(onMessageKey{})
	if !ok {
		return nil
	}
	if fn, ok := v.(OnMessageFunc); ok {
		return fn
	}
	return nil
}

type geminiOptsKey struct{}

// getGeminiOpts retrieves or initializes GeminiOptions from the common Options.
func getGeminiOpts(o *agentrunner.Options) *GeminiOptions {
	v, ok := o.GetExtra(geminiOptsKey{})
	if ok {
		return v.(*GeminiOptions)
	}
	opts := &GeminiOptions{}
	o.SetExtra(geminiOptsKey{}, opts)
	return opts
}

// GetGeminiOptions extracts Gemini-specific options from resolved Options.
// Returns nil if no Gemini-specific options were set.
func GetGeminiOptions(o *agentrunner.Options) *GeminiOptions {
	v, ok := o.GetExtra(geminiOptsKey{})
	if !ok {
		return nil
	}
	return v.(*GeminiOptions)
}
