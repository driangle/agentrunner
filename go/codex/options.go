package codex

import "github.com/driangle/agentrunner/go"

// CodexOptions holds Codex CLI-specific configuration that extends the common Options.
type CodexOptions struct {
	// Sandbox sets the sandbox policy: "read-only", "workspace-write", or "danger-full-access".
	Sandbox string

	// Approval sets the approval policy: "untrusted", "on-request", or "never".
	Approval string

	// OutputSchema is a path to a JSON Schema file for structured output validation.
	OutputSchema string

	// Images is a list of image file paths for multimodal input.
	Images []string

	// Profile is a named config profile from config.toml.
	Profile string

	// Resume is a session ID to resume.
	Resume string

	// Search enables live web search.
	Search bool

	// FullAuto enables automatic execution with workspace-write sandbox.
	FullAuto bool

	// Ephemeral runs without persisting session files to disk.
	Ephemeral bool

	// AddDirs lists additional directories that should be writable.
	AddDirs []string
}

// WithSandbox sets the sandbox policy.
func WithSandbox(mode string) agentrunner.Option {
	return func(o *agentrunner.Options) {
		getCodexOpts(o).Sandbox = mode
	}
}

// WithApproval sets the approval policy.
func WithApproval(policy string) agentrunner.Option {
	return func(o *agentrunner.Options) {
		getCodexOpts(o).Approval = policy
	}
}

// WithOutputSchema sets the path to a JSON Schema file for structured output.
func WithOutputSchema(path string) agentrunner.Option {
	return func(o *agentrunner.Options) {
		getCodexOpts(o).OutputSchema = path
	}
}

// WithImages sets image file paths for multimodal input.
func WithImages(paths ...string) agentrunner.Option {
	return func(o *agentrunner.Options) {
		getCodexOpts(o).Images = paths
	}
}

// WithProfile sets the named config profile.
func WithProfile(name string) agentrunner.Option {
	return func(o *agentrunner.Options) {
		getCodexOpts(o).Profile = name
	}
}

// WithResume sets the session ID to resume.
func WithResume(sessionID string) agentrunner.Option {
	return func(o *agentrunner.Options) {
		getCodexOpts(o).Resume = sessionID
	}
}

// WithSearch enables live web search.
func WithSearch() agentrunner.Option {
	return func(o *agentrunner.Options) {
		getCodexOpts(o).Search = true
	}
}

// WithFullAuto enables automatic execution with workspace-write sandbox.
func WithFullAuto() agentrunner.Option {
	return func(o *agentrunner.Options) {
		getCodexOpts(o).FullAuto = true
	}
}

// WithEphemeral runs without persisting session files to disk.
func WithEphemeral() agentrunner.Option {
	return func(o *agentrunner.Options) {
		getCodexOpts(o).Ephemeral = true
	}
}

// WithAddDirs sets additional writable directories.
func WithAddDirs(dirs ...string) agentrunner.Option {
	return func(o *agentrunner.Options) {
		getCodexOpts(o).AddDirs = dirs
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

type codexOptsKey struct{}

// getCodexOpts retrieves or initializes CodexOptions from the common Options.
func getCodexOpts(o *agentrunner.Options) *CodexOptions {
	v, ok := o.GetExtra(codexOptsKey{})
	if ok {
		return v.(*CodexOptions)
	}
	opts := &CodexOptions{}
	o.SetExtra(codexOptsKey{}, opts)
	return opts
}

// GetCodexOptions extracts Codex-specific options from resolved Options.
// Returns nil if no Codex-specific options were set.
func GetCodexOptions(o *agentrunner.Options) *CodexOptions {
	v, ok := o.GetExtra(codexOptsKey{})
	if !ok {
		return nil
	}
	return v.(*CodexOptions)
}
