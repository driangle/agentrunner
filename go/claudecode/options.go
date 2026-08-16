package claudecode

import "github.com/driangle/agentrunner/go"

// WithAllowedTools specifies which tools the agent may use.
func WithAllowedTools(tools ...string) agentrunner.Option {
	return func(o *agentrunner.Options) {
		opts := getClaudeOpts(o)
		opts.AllowedTools = tools
	}
}

// WithTools restricts the session to an exclusive set of built-in tools
// (--tools). Unlike WithAllowedTools, which only pre-approves tools and leaves
// unlisted built-ins available, --tools replaces the built-in tool set, so every
// unlisted built-in is denied at registration — including built-ins added by
// future CLI versions.
//
// Pass "" to disable all tools, "default" to use all tools, or explicit tool
// names (e.g. WithTools("Read", "Grep")). Calling WithTools with no arguments
// leaves the CLI default in place.
//
// Requires Claude Code CLI >= MinCLIVersion.
func WithTools(tools ...string) agentrunner.Option {
	return func(o *agentrunner.Options) {
		opts := getClaudeOpts(o)
		opts.Tools = tools
	}
}

// WithDisallowedTools specifies which tools the agent may not use.
func WithDisallowedTools(tools ...string) agentrunner.Option {
	return func(o *agentrunner.Options) {
		opts := getClaudeOpts(o)
		opts.DisallowedTools = tools
	}
}

// WithMCPConfig sets the path to the MCP server configuration file.
func WithMCPConfig(path string) agentrunner.Option {
	return func(o *agentrunner.Options) {
		opts := getClaudeOpts(o)
		opts.MCPConfig = path
	}
}

// WithJSONSchema sets the JSON Schema for structured output.
func WithJSONSchema(schema string) agentrunner.Option {
	return func(o *agentrunner.Options) {
		opts := getClaudeOpts(o)
		opts.JSONSchema = schema
	}
}

// WithMaxBudgetUSD sets the cost limit for the invocation.
func WithMaxBudgetUSD(budget float64) agentrunner.Option {
	return func(o *agentrunner.Options) {
		opts := getClaudeOpts(o)
		opts.MaxBudgetUSD = budget
	}
}

// WithResume resumes a previous session by ID.
func WithResume(sessionID string) agentrunner.Option {
	return func(o *agentrunner.Options) {
		opts := getClaudeOpts(o)
		opts.Resume = sessionID
	}
}

// WithContinue continues the most recent session.
func WithContinue() agentrunner.Option {
	return func(o *agentrunner.Options) {
		opts := getClaudeOpts(o)
		opts.Continue = true
	}
}

// WithSessionID sets a specific session ID for the conversation.
func WithSessionID(id string) agentrunner.Option {
	return func(o *agentrunner.Options) {
		opts := getClaudeOpts(o)
		opts.SessionID = id
	}
}

// WithIncludePartialMessages enables streaming of partial/incremental messages.
func WithIncludePartialMessages() agentrunner.Option {
	return func(o *agentrunner.Options) {
		opts := getClaudeOpts(o)
		opts.IncludePartialMessages = true
	}
}

// WithChannelEnabled enables two-way channel communication (experimental). When enabled,
// the runner automatically starts the agentrunner-mcp MCP server and
// wires it into the Claude CLI invocation. Use session.Send() to deliver
// channel.ChannelMessage values to the agent.
func WithChannelEnabled() agentrunner.Option {
	return func(o *agentrunner.Options) {
		opts := getClaudeOpts(o)
		opts.ChannelEnabled = true
	}
}

// WithChannelLogFile sets the file path for channel MCP server logs.
// Only used when channel is enabled.
func WithChannelLogFile(path string) agentrunner.Option {
	return func(o *agentrunner.Options) {
		opts := getClaudeOpts(o)
		opts.ChannelLogFile = path
	}
}

// WithPermissionMode sets the --permission-mode flag for the Claude Code CLI.
// Valid modes: "default", "acceptEdits", "auto", "plan", "dontAsk", "bypassPermissions".
// When set, this takes precedence over DangerouslySkipPermissions.
func WithPermissionMode(mode string) agentrunner.Option {
	return func(o *agentrunner.Options) {
		opts := getClaudeOpts(o)
		opts.PermissionMode = mode
	}
}

// WithChannelLogLevel sets the log level for the channel MCP server.
// Valid values: "debug", "info", "warn", "error". Defaults to "info".
// Only used when channel is enabled.
func WithChannelLogLevel(level string) agentrunner.Option {
	return func(o *agentrunner.Options) {
		opts := getClaudeOpts(o)
		opts.ChannelLogLevel = level
	}
}

// ClaudeOptions holds Claude Code-specific configuration that extends
// the common Options.
type ClaudeOptions struct {
	// AllowedTools specifies which tools the agent may use.
	AllowedTools []string

	// DisallowedTools specifies which tools the agent may not use.
	DisallowedTools []string

	// Tools is an exclusive whitelist of built-in tools (--tools). When set,
	// unlisted built-ins are denied at registration. A single element of ""
	// disables all tools; "default" restores the full built-in set. Nil or
	// empty leaves the CLI default.
	Tools []string

	// MCPConfig is the path to the MCP server configuration file.
	MCPConfig string

	// JSONSchema is a JSON Schema for structured output.
	JSONSchema string

	// MaxBudgetUSD sets a cost limit for the invocation.
	MaxBudgetUSD float64

	// Resume is a session ID to resume.
	Resume string

	// Continue indicates whether to continue the most recent session.
	Continue bool

	// SessionID sets a specific session ID for the conversation.
	SessionID string

	// IncludePartialMessages enables streaming of partial/incremental messages.
	IncludePartialMessages bool

	// PermissionMode sets the --permission-mode flag.
	// Valid modes: "default", "acceptEdits", "auto", "plan", "dontAsk", "bypassPermissions".
	// When set, takes precedence over DangerouslySkipPermissions.
	PermissionMode string

	// ChannelEnabled enables two-way channel communication via the
	// agentrunner-mcp MCP server (experimental).
	ChannelEnabled bool

	// ChannelLogFile is the file path for channel MCP server logs.
	ChannelLogFile string

	// ChannelLogLevel is the log level for the channel MCP server
	// ("debug", "info", "warn", "error"). Defaults to "info".
	ChannelLogLevel string
}

// OnMessageFunc is a callback invoked for each streaming message.
type OnMessageFunc func(agentrunner.Message)

// WithOnMessage sets a callback that is invoked for each streaming message
// during Start/Run. The callback is called before the message is sent on the
// channel, so it can be used for logging, progress display, etc.
func WithOnMessage(fn OnMessageFunc) agentrunner.Option {
	return func(o *agentrunner.Options) {
		o.SetExtra(onMessageKey{}, fn)
	}
}

type onMessageKey struct{}

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

// claudeOptsKey is the key used to store ClaudeOptions in Options.
type claudeOptsKey struct{}

// getClaudeOpts retrieves or initializes ClaudeOptions from the common Options.
func getClaudeOpts(o *agentrunner.Options) *ClaudeOptions {
	v, ok := o.GetExtra(claudeOptsKey{})
	if ok {
		return v.(*ClaudeOptions)
	}
	opts := &ClaudeOptions{}
	o.SetExtra(claudeOptsKey{}, opts)
	return opts
}

// GetClaudeOptions extracts Claude-specific options from resolved Options.
// Returns nil if no Claude-specific options were set.
func GetClaudeOptions(o *agentrunner.Options) *ClaudeOptions {
	v, ok := o.GetExtra(claudeOptsKey{})
	if !ok {
		return nil
	}
	return v.(*ClaudeOptions)
}
