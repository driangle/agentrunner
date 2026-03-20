// Package ollama provides a Runner implementation for invoking Ollama models
// via the Ollama HTTP API. It implements the common Runner interface using
// POST /api/chat with streaming ndjson responses, enabling fully local/offline
// agent execution with locally-hosted models.
package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"

	agentrunner "github.com/driangle/agent-runner/go"
)

// Compile-time interface assertion.
var _ agentrunner.Runner = (*Runner)(nil)

const defaultBaseURL = "http://localhost:11434"

// RunnerOption configures a Runner.
type RunnerOption func(*Runner)

// WithBaseURL overrides the Ollama API base URL (default "http://localhost:11434").
func WithBaseURL(url string) RunnerOption {
	return func(r *Runner) { r.baseURL = url }
}

// WithHTTPClient sets a custom HTTP client for API requests.
func WithHTTPClient(client *http.Client) RunnerOption {
	return func(r *Runner) { r.client = client }
}

// WithLogger sets a structured logger for debug output.
// When nil (the default), no logging is performed.
func WithLogger(l *slog.Logger) RunnerOption {
	return func(r *Runner) { r.logger = l }
}

// Runner implements agentrunner.Runner for the Ollama HTTP API.
type Runner struct {
	baseURL string
	client  *http.Client
	logger  *slog.Logger
}

// NewRunner creates a Runner with the given options.
func NewRunner(opts ...RunnerOption) *Runner {
	r := &Runner{
		baseURL: defaultBaseURL,
		client:  http.DefaultClient,
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Start launches an Ollama chat request and returns a Session for full control
// over the lifecycle. Messages arrive on session.Messages; the final result is
// available via session.Result().
func (r *Runner) Start(ctx context.Context, prompt string, opts ...agentrunner.Option) *agentrunner.Session {
	var options agentrunner.Options
	for _, o := range opts {
		o(&options)
	}

	var timeoutCancel context.CancelFunc
	if options.Timeout > 0 {
		ctx, timeoutCancel = context.WithTimeout(ctx, options.Timeout)
	}

	ctx, sessionCancel := context.WithCancel(ctx)

	msgCh := make(chan agentrunner.Message)
	resultCh := make(chan agentrunner.ResultOrError, 1)

	go func() {
		defer close(msgCh)
		defer close(resultCh)
		defer sessionCancel()
		if timeoutCancel != nil {
			defer timeoutCancel()
		}

		req, err := r.buildRequest(ctx, prompt, &options)
		if err != nil {
			resultCh <- agentrunner.ResultOrError{Err: err}
			return
		}

		r.logRequest(ctx, req)

		resp, err := r.client.Do(req)
		if err != nil {
			resultCh <- agentrunner.ResultOrError{Err: mapHTTPError(err)}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			resultCh <- agentrunner.ResultOrError{Err: mapStatusError(resp.StatusCode)}
			return
		}

		onMessage := GetOnMessage(&options)
		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

		var textParts []string
		var finalResp *chatResponse

		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			var chunk chatResponse
			if err := json.Unmarshal([]byte(line), &chunk); err != nil {
				resultCh <- agentrunner.ResultOrError{Err: fmt.Errorf("%w: %v", agentrunner.ErrParseError, err)}
				return
			}

			if chunk.Message.Content != "" {
				textParts = append(textParts, chunk.Message.Content)
			}

			if chunk.Done {
				finalResp = &chunk
			}

			raw := json.RawMessage(line)
			msgType := agentrunner.MessageTypeAssistant
			if chunk.Done {
				msgType = agentrunner.MessageTypeResult
			}

			msg := agentrunner.Message{
				Type: msgType,
				Raw:  raw,
			}

			if onMessage != nil {
				onMessage(msg)
			}

			select {
			case msgCh <- msg:
			case <-ctx.Done():
				resultCh <- agentrunner.ResultOrError{Err: mapContextError(ctx.Err())}
				return
			}
		}

		if err := scanner.Err(); err != nil {
			if ctx.Err() != nil {
				resultCh <- agentrunner.ResultOrError{Err: mapContextError(ctx.Err())}
				return
			}
			resultCh <- agentrunner.ResultOrError{Err: fmt.Errorf("%w: %v", agentrunner.ErrParseError, err)}
			return
		}

		if finalResp == nil {
			resultCh <- agentrunner.ResultOrError{Err: agentrunner.ErrNoResult}
			return
		}

		result := &agentrunner.Result{
			Text:       strings.Join(textParts, ""),
			DurationMs: finalResp.TotalDuration / 1e6,
			Usage: agentrunner.Usage{
				InputTokens:  finalResp.PromptEvalCount,
				OutputTokens: finalResp.EvalCount,
			},
		}
		resultCh <- agentrunner.ResultOrError{Result: result}
	}()

	return agentrunner.NewSession(msgCh, resultCh, sessionCancel)
}

// Run executes a prompt against Ollama and returns the final result.
func (r *Runner) Run(ctx context.Context, prompt string, opts ...agentrunner.Option) (*agentrunner.Result, error) {
	session := r.Start(ctx, prompt, opts...)
	for range session.Messages {
	}
	return session.Result()
}

// RunStream executes a prompt against Ollama and streams messages as they arrive.
func (r *Runner) RunStream(ctx context.Context, prompt string, opts ...agentrunner.Option) (<-chan agentrunner.Message, <-chan error) {
	session := r.Start(ctx, prompt, opts...)

	errCh := make(chan error, 1)
	go func() {
		defer close(errCh)
		result, err := session.Result()
		if err != nil {
			errCh <- err
		} else if result != nil && result.IsError {
			// Not a library error — caller reads it from the result message.
		}
	}()

	return session.Messages, errCh
}

func (r *Runner) buildRequest(ctx context.Context, prompt string, opts *agentrunner.Options) (*http.Request, error) {
	messages := buildMessages(prompt, opts)

	chatReq := chatRequest{
		Model:    opts.Model,
		Messages: messages,
		Stream:   true,
	}

	if oo := GetOllamaOptions(opts); oo != nil {
		chatReq.Format = oo.Format
		chatReq.KeepAlive = oo.KeepAlive
		chatReq.Think = oo.Think
		chatReq.Options = buildModelOptions(oo)
	}

	body, err := json.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := r.baseURL + "/api/chat"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

func buildMessages(prompt string, opts *agentrunner.Options) []chatMessage {
	var messages []chatMessage

	systemPrompt := opts.SystemPrompt
	if opts.AppendSystemPrompt != "" {
		if systemPrompt != "" {
			systemPrompt += "\n" + opts.AppendSystemPrompt
		} else {
			systemPrompt = opts.AppendSystemPrompt
		}
	}

	if systemPrompt != "" {
		messages = append(messages, chatMessage{Role: "system", Content: systemPrompt})
	}

	messages = append(messages, chatMessage{Role: "user", Content: prompt})
	return messages
}

func buildModelOptions(oo *OllamaOptions) *modelOptions {
	mo := &modelOptions{
		Temperature: oo.Temperature,
		NumCtx:      oo.NumCtx,
		NumPredict:  oo.NumPredict,
		Seed:        oo.Seed,
		Stop:        oo.Stop,
		TopK:        oo.TopK,
		TopP:        oo.TopP,
		MinP:        oo.MinP,
	}
	// Return nil if all fields are zero/nil to avoid sending empty options.
	if mo.Temperature == nil && mo.NumCtx == 0 && mo.NumPredict == 0 &&
		mo.Seed == 0 && mo.Stop == nil && mo.TopK == 0 &&
		mo.TopP == nil && mo.MinP == nil {
		return nil
	}
	return mo
}

func mapHTTPError(err error) error {
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return fmt.Errorf("%w: %v", agentrunner.ErrNotFound, err)
	}
	// Connection refused shows up as a different type on some systems.
	if errors.Is(err, context.DeadlineExceeded) {
		return agentrunner.ErrTimeout
	}
	if errors.Is(err, context.Canceled) {
		return agentrunner.ErrCancelled
	}
	return fmt.Errorf("%w: %v", agentrunner.ErrNotFound, err)
}

func mapStatusError(code int) error {
	if code == http.StatusNotFound {
		return fmt.Errorf("%w: model not found (HTTP 404)", agentrunner.ErrNotFound)
	}
	return fmt.Errorf("%w: HTTP %d", agentrunner.ErrHTTPError, code)
}

func mapContextError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return agentrunner.ErrTimeout
	}
	return agentrunner.ErrCancelled
}

func (r *Runner) logRequest(ctx context.Context, req *http.Request) {
	if r.logger == nil {
		return
	}
	r.logger.DebugContext(ctx, "executing Ollama API request",
		slog.String("method", req.Method),
		slog.String("url", req.URL.String()),
	)
}
