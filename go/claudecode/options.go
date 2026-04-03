package claudecode

import "github.com/driangle/agentrunner/go"

// WithAllowedTools specifies which tools the agent may use.
func WithAllowedTools(tools ...string) agentrunner.Option {
	return func(o *agentrunner.Options) {
		opts := getClaudeOpts(o)
		opts.AllowedTools = tools
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

// WithChannelEnabled enables two-way channel communication. When enabled,
// the runner automatically starts the agentrunner-channel MCP server and
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

	// ChannelEnabled enables two-way channel communication via the
	// agentrunner-channel MCP server.
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
