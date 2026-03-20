package ollama

import agentrunner "github.com/driangle/agent-runner/go"

// OllamaOptions holds Ollama-specific configuration that extends the common Options.
type OllamaOptions struct {
	Temperature *float64
	NumCtx      int
	NumPredict  int
	Seed        int
	Stop        []string
	TopK        int
	TopP        *float64
	MinP        *float64
	Format      string
	KeepAlive   string
	Think       *bool
}

// WithTemperature sets the sampling temperature.
func WithTemperature(t float64) agentrunner.Option {
	return func(o *agentrunner.Options) {
		opts := getOllamaOpts(o)
		opts.Temperature = &t
	}
}

// WithNumCtx sets the context window size.
func WithNumCtx(n int) agentrunner.Option {
	return func(o *agentrunner.Options) {
		opts := getOllamaOpts(o)
		opts.NumCtx = n
	}
}

// WithNumPredict sets the maximum number of tokens to generate.
func WithNumPredict(n int) agentrunner.Option {
	return func(o *agentrunner.Options) {
		opts := getOllamaOpts(o)
		opts.NumPredict = n
	}
}

// WithSeed sets the random seed for reproducible generation.
func WithSeed(seed int) agentrunner.Option {
	return func(o *agentrunner.Options) {
		opts := getOllamaOpts(o)
		opts.Seed = seed
	}
}

// WithStop sets stop sequences.
func WithStop(sequences ...string) agentrunner.Option {
	return func(o *agentrunner.Options) {
		opts := getOllamaOpts(o)
		opts.Stop = sequences
	}
}

// WithTopK sets the top-k sampling parameter.
func WithTopK(k int) agentrunner.Option {
	return func(o *agentrunner.Options) {
		opts := getOllamaOpts(o)
		opts.TopK = k
	}
}

// WithTopP sets the top-p (nucleus) sampling parameter.
func WithTopP(p float64) agentrunner.Option {
	return func(o *agentrunner.Options) {
		opts := getOllamaOpts(o)
		opts.TopP = &p
	}
}

// WithMinP sets the min-p sampling parameter.
func WithMinP(p float64) agentrunner.Option {
	return func(o *agentrunner.Options) {
		opts := getOllamaOpts(o)
		opts.MinP = &p
	}
}

// WithFormat sets the response format (e.g. "json").
func WithFormat(format string) agentrunner.Option {
	return func(o *agentrunner.Options) {
		opts := getOllamaOpts(o)
		opts.Format = format
	}
}

// WithKeepAlive sets how long the model stays loaded in memory (e.g. "5m", "0" to unload immediately).
func WithKeepAlive(duration string) agentrunner.Option {
	return func(o *agentrunner.Options) {
		opts := getOllamaOpts(o)
		opts.KeepAlive = duration
	}
}

// WithThink enables or disables thinking/reasoning for supported models (e.g. qwen3).
// When enabled, streaming chunks include a "thinking" field with reasoning content
// before the final answer appears in "content".
func WithThink(enabled bool) agentrunner.Option {
	return func(o *agentrunner.Options) {
		opts := getOllamaOpts(o)
		opts.Think = &enabled
	}
}

// OnMessageFunc is a callback invoked for each streaming message.
type OnMessageFunc func(agentrunner.Message)

type onMessageKey struct{}

// WithOnMessage sets a callback that is invoked for each streaming message
// during RunStream. The callback is called before the message is sent on the
// channel, so it can be used for logging, progress display, etc.
func WithOnMessage(fn OnMessageFunc) agentrunner.Option {
	return func(o *agentrunner.Options) {
		if o.Extra == nil {
			o.Extra = make(map[any]any)
		}
		o.Extra[onMessageKey{}] = fn
	}
}

// GetOnMessage extracts the OnMessage callback from resolved Options.
// Returns nil if no callback was set.
func GetOnMessage(o *agentrunner.Options) OnMessageFunc {
	if o.Extra == nil {
		return nil
	}
	if v, ok := o.Extra[onMessageKey{}]; ok {
		if fn, ok := v.(OnMessageFunc); ok {
			return fn
		}
	}
	return nil
}

type ollamaOptsKey struct{}

// getOllamaOpts retrieves or initializes OllamaOptions from the common Options.
func getOllamaOpts(o *agentrunner.Options) *OllamaOptions {
	if o.Extra == nil {
		o.Extra = make(map[any]any)
	}
	key := ollamaOptsKey{}
	if v, ok := o.Extra[key]; ok {
		return v.(*OllamaOptions)
	}
	opts := &OllamaOptions{}
	o.Extra[key] = opts
	return opts
}

// GetOllamaOptions extracts Ollama-specific options from resolved Options.
// Returns nil if no Ollama-specific options were set.
func GetOllamaOptions(o *agentrunner.Options) *OllamaOptions {
	if o.Extra == nil {
		return nil
	}
	if v, ok := o.Extra[ollamaOptsKey{}]; ok {
		return v.(*OllamaOptions)
	}
	return nil
}
