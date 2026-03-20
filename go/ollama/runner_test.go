package ollama

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	agentrunner "github.com/driangle/agent-runner/go"
)

// newTestRunner creates a Runner pointing at the given test server.
func newTestRunner(server *httptest.Server) *Runner {
	return NewRunner(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
}

// streamLine builds a JSON line for a non-final chat response chunk.
func streamLine(content string) string {
	resp := chatResponse{
		Model:   "llama3",
		Message: chatMessage{Role: "assistant", Content: content},
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

// doneLine builds a JSON line for the final (done=true) chat response.
func doneLine(content string, totalDuration int64, promptEvalCount, evalCount int) string {
	resp := chatResponse{
		Model:              "llama3",
		Message:            chatMessage{Role: "assistant", Content: content},
		Done:               true,
		TotalDuration:      totalDuration,
		PromptEvalCount:    promptEvalCount,
		EvalCount:          evalCount,
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

func happyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/x-ndjson")
	fmt.Fprintln(w, streamLine("Hello"))
	fmt.Fprintln(w, streamLine(" world"))
	fmt.Fprintln(w, doneLine("", 2_000_000_000, 100, 50)) // 2s = 2000ms
}

// --- Run tests ---

func TestRunHappyPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(happyHandler))
	defer server.Close()

	r := newTestRunner(server)
	result, err := r.Run(context.Background(), "say hello", agentrunner.WithModel("llama3"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "Hello world" {
		t.Errorf("text = %q, want %q", result.Text, "Hello world")
	}
	if result.DurationMs != 2000 {
		t.Errorf("duration_ms = %d, want 2000", result.DurationMs)
	}
	if result.Usage.InputTokens != 100 {
		t.Errorf("input_tokens = %d, want 100", result.Usage.InputTokens)
	}
	if result.Usage.OutputTokens != 50 {
		t.Errorf("output_tokens = %d, want 50", result.Usage.OutputTokens)
	}
	if result.CostUSD != 0 {
		t.Errorf("cost = %f, want 0", result.CostUSD)
	}
	if result.IsError {
		t.Error("is_error = true, want false")
	}
	if result.SessionID != "" {
		t.Errorf("session_id = %q, want empty", result.SessionID)
	}
}

func TestRunWithSystemPrompt(t *testing.T) {
	var gotBody chatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &gotBody)
		happyHandler(w, r)
	}))
	defer server.Close()

	r := newTestRunner(server)
	r.Run(context.Background(), "hello",
		agentrunner.WithModel("llama3"),
		agentrunner.WithSystemPrompt("You are helpful"),
	)

	if len(gotBody.Messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(gotBody.Messages))
	}
	if gotBody.Messages[0].Role != "system" || gotBody.Messages[0].Content != "You are helpful" {
		t.Errorf("system message = %+v", gotBody.Messages[0])
	}
	if gotBody.Messages[1].Role != "user" || gotBody.Messages[1].Content != "hello" {
		t.Errorf("user message = %+v", gotBody.Messages[1])
	}
}

func TestRunWithModel(t *testing.T) {
	var gotBody chatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &gotBody)
		happyHandler(w, r)
	}))
	defer server.Close()

	r := newTestRunner(server)
	r.Run(context.Background(), "hello", agentrunner.WithModel("codellama"))

	if gotBody.Model != "codellama" {
		t.Errorf("model = %q, want %q", gotBody.Model, "codellama")
	}
}

func TestRunWithOllamaOptions(t *testing.T) {
	var gotBody chatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &gotBody)
		happyHandler(w, r)
	}))
	defer server.Close()

	r := newTestRunner(server)
	r.Run(context.Background(), "hello",
		agentrunner.WithModel("llama3"),
		WithTemperature(0.7),
		WithNumCtx(4096),
		WithNumPredict(256),
		WithSeed(42),
		WithStop("END", "STOP"),
		WithTopK(40),
		WithTopP(0.9),
		WithMinP(0.05),
		WithFormat("json"),
		WithKeepAlive("5m"),
	)

	if gotBody.Format != "json" {
		t.Errorf("format = %q, want %q", gotBody.Format, "json")
	}
	if gotBody.KeepAlive != "5m" {
		t.Errorf("keep_alive = %q, want %q", gotBody.KeepAlive, "5m")
	}
	if gotBody.Options == nil {
		t.Fatal("options is nil")
	}
	if gotBody.Options.Temperature == nil || *gotBody.Options.Temperature != 0.7 {
		t.Errorf("temperature = %v, want 0.7", gotBody.Options.Temperature)
	}
	if gotBody.Options.NumCtx != 4096 {
		t.Errorf("num_ctx = %d, want 4096", gotBody.Options.NumCtx)
	}
	if gotBody.Options.NumPredict != 256 {
		t.Errorf("num_predict = %d, want 256", gotBody.Options.NumPredict)
	}
	if gotBody.Options.Seed != 42 {
		t.Errorf("seed = %d, want 42", gotBody.Options.Seed)
	}
	if len(gotBody.Options.Stop) != 2 || gotBody.Options.Stop[0] != "END" {
		t.Errorf("stop = %v, want [END STOP]", gotBody.Options.Stop)
	}
	if gotBody.Options.TopK != 40 {
		t.Errorf("top_k = %d, want 40", gotBody.Options.TopK)
	}
	if gotBody.Options.TopP == nil || *gotBody.Options.TopP != 0.9 {
		t.Errorf("top_p = %v, want 0.9", gotBody.Options.TopP)
	}
	if gotBody.Options.MinP == nil || *gotBody.Options.MinP != 0.05 {
		t.Errorf("min_p = %v, want 0.05", gotBody.Options.MinP)
	}
}

func TestRunServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	r := newTestRunner(server)
	_, err := r.Run(context.Background(), "hello", agentrunner.WithModel("llama3"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, agentrunner.ErrHTTPError) {
		t.Errorf("err = %v, want ErrHTTPError", err)
	}
}

func TestRunModelNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	r := newTestRunner(server)
	_, err := r.Run(context.Background(), "hello", agentrunner.WithModel("nonexistent"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, agentrunner.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestRunConnectionRefused(t *testing.T) {
	// Point at a port that is not listening.
	r := NewRunner(WithBaseURL("http://127.0.0.1:1"))
	_, err := r.Run(context.Background(), "hello", agentrunner.WithModel("llama3"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, agentrunner.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestRunTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer server.Close()

	r := newTestRunner(server)
	_, err := r.Run(context.Background(), "hello",
		agentrunner.WithModel("llama3"),
		agentrunner.WithTimeout(100*time.Millisecond),
	)
	if err != agentrunner.ErrTimeout {
		t.Errorf("err = %v, want ErrTimeout", err)
	}
}

func TestRunCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	r := newTestRunner(server)

	done := make(chan error, 1)
	go func() {
		_, err := r.Run(ctx, "hello", agentrunner.WithModel("llama3"))
		done <- err
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	err := <-done
	if err != agentrunner.ErrCancelled {
		t.Errorf("err = %v, want ErrCancelled", err)
	}
}

func TestRunParseError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		fmt.Fprintln(w, "not valid json")
	}))
	defer server.Close()

	r := newTestRunner(server)
	_, err := r.Run(context.Background(), "hello", agentrunner.WithModel("llama3"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, agentrunner.ErrParseError) {
		t.Errorf("err = %v, want ErrParseError", err)
	}
}

func TestRunNoResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		// Stream chunks but never send done=true.
		fmt.Fprintln(w, streamLine("partial"))
	}))
	defer server.Close()

	r := newTestRunner(server)
	_, err := r.Run(context.Background(), "hello", agentrunner.WithModel("llama3"))
	if err != agentrunner.ErrNoResult {
		t.Errorf("err = %v, want ErrNoResult", err)
	}
}

func TestRunEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		// Empty body.
	}))
	defer server.Close()

	r := newTestRunner(server)
	_, err := r.Run(context.Background(), "hello", agentrunner.WithModel("llama3"))
	if err != agentrunner.ErrNoResult {
		t.Errorf("err = %v, want ErrNoResult", err)
	}
}

// --- RunStream tests ---

func TestRunStreamHappyPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(happyHandler))
	defer server.Close()

	r := newTestRunner(server)
	msgCh, errCh := r.RunStream(context.Background(), "say hello", agentrunner.WithModel("llama3"))

	var messages []agentrunner.Message
	for msg := range msgCh {
		messages = append(messages, msg)
	}

	err := <-errCh
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(messages) != 3 {
		t.Fatalf("got %d messages, want 3", len(messages))
	}

	// First two are assistant chunks, last is result.
	if messages[0].Type != agentrunner.MessageTypeAssistant {
		t.Errorf("messages[0].Type = %q, want assistant", messages[0].Type)
	}
	if messages[1].Type != agentrunner.MessageTypeAssistant {
		t.Errorf("messages[1].Type = %q, want assistant", messages[1].Type)
	}
	if messages[2].Type != agentrunner.MessageTypeResult {
		t.Errorf("messages[2].Type = %q, want result", messages[2].Type)
	}
}

func TestRunStreamOnMessageCallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(happyHandler))
	defer server.Close()

	r := newTestRunner(server)

	var callbackMessages []agentrunner.Message
	msgCh, errCh := r.RunStream(context.Background(), "test callback",
		agentrunner.WithModel("llama3"),
		WithOnMessage(func(msg agentrunner.Message) {
			callbackMessages = append(callbackMessages, msg)
		}),
	)

	var channelMessages []agentrunner.Message
	for msg := range msgCh {
		channelMessages = append(channelMessages, msg)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(callbackMessages) != len(channelMessages) {
		t.Fatalf("callback got %d messages, channel got %d", len(callbackMessages), len(channelMessages))
	}

	for i := range callbackMessages {
		if callbackMessages[i].Type != channelMessages[i].Type {
			t.Errorf("message[%d]: callback type %q != channel type %q",
				i, callbackMessages[i].Type, channelMessages[i].Type)
		}
	}
}

func TestRunStreamRawJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(happyHandler))
	defer server.Close()

	r := newTestRunner(server)
	msgCh, errCh := r.RunStream(context.Background(), "test raw", agentrunner.WithModel("llama3"))

	for msg := range msgCh {
		if len(msg.Raw) == 0 {
			t.Errorf("message type %q has empty Raw field", msg.Type)
		}
	}

	if err := <-errCh; err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunStreamTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer server.Close()

	r := newTestRunner(server)
	msgCh, errCh := r.RunStream(context.Background(), "hello",
		agentrunner.WithModel("llama3"),
		agentrunner.WithTimeout(100*time.Millisecond),
	)

	for range msgCh {
	}

	err := <-errCh
	if err != agentrunner.ErrTimeout {
		t.Errorf("err = %v, want ErrTimeout", err)
	}
}

func TestRunStreamCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	r := newTestRunner(server)
	msgCh, errCh := r.RunStream(ctx, "hello", agentrunner.WithModel("llama3"))

	time.Sleep(50 * time.Millisecond)
	cancel()

	for range msgCh {
	}

	err := <-errCh
	if err != agentrunner.ErrCancelled {
		t.Errorf("err = %v, want ErrCancelled", err)
	}
}

// --- Start tests ---

func TestStartHappyPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(happyHandler))
	defer server.Close()

	r := newTestRunner(server)
	session := r.Start(context.Background(), "say hello", agentrunner.WithModel("llama3"))

	var messages []agentrunner.Message
	for msg := range session.Messages {
		messages = append(messages, msg)
	}

	result, err := session.Result()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Text != "Hello world" {
		t.Errorf("text = %q, want %q", result.Text, "Hello world")
	}
	if len(messages) != 3 {
		t.Errorf("got %d messages, want 3", len(messages))
	}
}

func TestStartAbortMidStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		fmt.Fprintln(w, streamLine("partial"))
		flusher.Flush()
		// Block until client disconnects.
		<-r.Context().Done()
	}))
	defer server.Close()

	r := newTestRunner(server)
	session := r.Start(context.Background(), "long task", agentrunner.WithModel("llama3"))

	// Read one message then abort.
	<-session.Messages
	session.Abort()

	// Drain remaining.
	for range session.Messages {
	}

	_, err := session.Result()
	if err != agentrunner.ErrCancelled {
		t.Errorf("err = %v, want ErrCancelled", err)
	}
}

// --- Request building tests ---

func TestBuildMessagesMinimal(t *testing.T) {
	opts := &agentrunner.Options{}
	messages := buildMessages("hello world", opts)

	if len(messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(messages))
	}
	if messages[0].Role != "user" || messages[0].Content != "hello world" {
		t.Errorf("message = %+v, want user/hello world", messages[0])
	}
}

func TestBuildMessagesWithSystemPrompt(t *testing.T) {
	opts := &agentrunner.Options{SystemPrompt: "Be helpful"}
	messages := buildMessages("hello", opts)

	if len(messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(messages))
	}
	if messages[0].Role != "system" || messages[0].Content != "Be helpful" {
		t.Errorf("system message = %+v", messages[0])
	}
	if messages[1].Role != "user" || messages[1].Content != "hello" {
		t.Errorf("user message = %+v", messages[1])
	}
}

func TestBuildMessagesWithAppendSystemPrompt(t *testing.T) {
	opts := &agentrunner.Options{
		SystemPrompt:       "Be helpful",
		AppendSystemPrompt: "Be concise",
	}
	messages := buildMessages("hello", opts)

	if len(messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(messages))
	}
	if messages[0].Content != "Be helpful\nBe concise" {
		t.Errorf("system content = %q, want %q", messages[0].Content, "Be helpful\nBe concise")
	}
}

func TestBuildMessagesAppendSystemPromptOnly(t *testing.T) {
	opts := &agentrunner.Options{AppendSystemPrompt: "Be concise"}
	messages := buildMessages("hello", opts)

	if len(messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(messages))
	}
	if messages[0].Role != "system" || messages[0].Content != "Be concise" {
		t.Errorf("system message = %+v", messages[0])
	}
}

func TestBuildRequestWithAllOptions(t *testing.T) {
	var gotBody chatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &gotBody)
		happyHandler(w, r)
	}))
	defer server.Close()

	r := newTestRunner(server)
	r.Run(context.Background(), "test",
		agentrunner.WithModel("codellama"),
		agentrunner.WithSystemPrompt("You are helpful"),
		agentrunner.WithAppendSystemPrompt("Be concise"),
		WithTemperature(0.5),
		WithNumCtx(2048),
		WithFormat("json"),
		WithKeepAlive("10m"),
	)

	if gotBody.Model != "codellama" {
		t.Errorf("model = %q, want codellama", gotBody.Model)
	}
	if !gotBody.Stream {
		t.Error("stream = false, want true")
	}
	if gotBody.Format != "json" {
		t.Errorf("format = %q, want json", gotBody.Format)
	}
	if gotBody.KeepAlive != "10m" {
		t.Errorf("keep_alive = %q, want 10m", gotBody.KeepAlive)
	}
	if len(gotBody.Messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(gotBody.Messages))
	}
	if gotBody.Messages[0].Content != "You are helpful\nBe concise" {
		t.Errorf("system = %q", gotBody.Messages[0].Content)
	}
	if gotBody.Options == nil {
		t.Fatal("options is nil")
	}
	if gotBody.Options.Temperature == nil || *gotBody.Options.Temperature != 0.5 {
		t.Errorf("temperature = %v, want 0.5", gotBody.Options.Temperature)
	}
	if gotBody.Options.NumCtx != 2048 {
		t.Errorf("num_ctx = %d, want 2048", gotBody.Options.NumCtx)
	}
}

// --- Thinking model tests ---

func thinkingHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/x-ndjson")
	// Phase 1: thinking chunks
	fmt.Fprintln(w, `{"model":"qwen3","message":{"role":"assistant","content":"","thinking":"Let me think"},"done":false}`)
	fmt.Fprintln(w, `{"model":"qwen3","message":{"role":"assistant","content":"","thinking":" about this"},"done":false}`)
	// Phase 2: content chunks
	fmt.Fprintln(w, `{"model":"qwen3","message":{"role":"assistant","content":"The answer","thinking":""},"done":false}`)
	fmt.Fprintln(w, `{"model":"qwen3","message":{"role":"assistant","content":" is 42","thinking":""},"done":false}`)
	// Done
	fmt.Fprintln(w, `{"model":"qwen3","message":{"role":"assistant","content":""},"done":true,"done_reason":"stop","total_duration":3000000000,"prompt_eval_count":50,"eval_count":100}`)
}

func TestRunThinkingModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(thinkingHandler))
	defer server.Close()

	r := newTestRunner(server)
	result, err := r.Run(context.Background(), "hello", agentrunner.WithModel("qwen3"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "The answer is 42" {
		t.Errorf("text = %q, want %q", result.Text, "The answer is 42")
	}
}

func TestRunStreamThinkingModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(thinkingHandler))
	defer server.Close()

	r := newTestRunner(server)
	msgCh, errCh := r.RunStream(context.Background(), "hello", agentrunner.WithModel("qwen3"))

	var thinkingParts, contentParts []string
	for msg := range msgCh {
		var chunk struct {
			Message struct {
				Content  string `json:"content"`
				Thinking string `json:"thinking"`
			} `json:"message"`
		}
		if err := json.Unmarshal(msg.Raw, &chunk); err == nil {
			if chunk.Message.Thinking != "" {
				thinkingParts = append(thinkingParts, chunk.Message.Thinking)
			}
			if chunk.Message.Content != "" {
				contentParts = append(contentParts, chunk.Message.Content)
			}
		}
	}

	if err := <-errCh; err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	thinking := ""
	for _, p := range thinkingParts {
		thinking += p
	}
	if thinking != "Let me think about this" {
		t.Errorf("thinking = %q, want %q", thinking, "Let me think about this")
	}

	content := ""
	for _, p := range contentParts {
		content += p
	}
	if content != "The answer is 42" {
		t.Errorf("content = %q, want %q", content, "The answer is 42")
	}
}

func TestWithThinkSendsRequestField(t *testing.T) {
	var gotBody chatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &gotBody)
		happyHandler(w, r)
	}))
	defer server.Close()

	r := newTestRunner(server)
	r.Run(context.Background(), "hello",
		agentrunner.WithModel("qwen3"),
		WithThink(true),
	)

	if gotBody.Think == nil || !*gotBody.Think {
		t.Errorf("think = %v, want true", gotBody.Think)
	}
}

// --- GetOllamaOptions tests ---

func TestGetOllamaOptionsNil(t *testing.T) {
	opts := &agentrunner.Options{}
	if got := GetOllamaOptions(opts); got != nil {
		t.Errorf("GetOllamaOptions = %v, want nil", got)
	}
}

func TestGetOllamaOptionsAfterSet(t *testing.T) {
	opts := &agentrunner.Options{}
	WithTemperature(0.5)(opts)

	oo := GetOllamaOptions(opts)
	if oo == nil {
		t.Fatal("GetOllamaOptions returned nil")
	}
	if oo.Temperature == nil || *oo.Temperature != 0.5 {
		t.Errorf("temperature = %v, want 0.5", oo.Temperature)
	}
}
