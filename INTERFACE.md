# Runner Interface Specification

This document defines the language-agnostic Runner interface that all implementations must follow. Each language library expresses this interface idiomatically, but the conceptual shape is the same.

## Runner

A Runner executes prompts against an AI coding agent and returns results. The primary primitive is `Start`, which returns a **Session** object representing a running agent process. `Run` and `RunStream` are thin convenience wrappers over `Start`.

```
Runner:
  Start(prompt, options)     → Session
  Run(prompt, options)       → Result
  RunStream(prompt, options) → Stream<Message>
```

- **Start** launches an agent process and returns a Session for full control over the lifecycle.
- **Run** starts a session, drains all messages internally, and returns the final result.
- **RunStream** starts a session and returns the messages iterable (the session's result is accessible separately).

Both `Run` and `RunStream` delegate to `Start`. All three accept the same options.

---

## Session

A Session encapsulates a running agent process. It exposes the read side — messages, result, and abort — while reserving a `send` method for future write-side support.

```
Session:
  messages → Stream<Message>   — iterable of messages as they arrive
  result   → Result            — blocks/resolves when the agent finishes
  abort()  → void              — terminates the agent process
  send()   → error             — reserved, throws "not yet supported"
```

### Language-Idiomatic Session

| Language   | Messages (read)                        | Result                        | Abort               |
|------------|----------------------------------------|-------------------------------|----------------------|
| TypeScript | `AsyncIterable<Message>`               | `Promise<Result>`             | `abort(): void`      |
| Go         | `<-chan Message`                        | `Result() (Result, error)`    | `context.Cancel()`   |
| Python     | `async for msg in session`             | `await session.result`        | `session.abort()`    |
| Java       | `Iterator<Message>` / `Flow.Publisher` | `CompletableFuture<Result>`   | `session.abort()`    |

The session must be safe to abandon. Abandoning a session (not draining messages, not awaiting result) should terminate the underlying process.

---

## Options

### Common Options

Every runner accepts these options. All are optional — runners provide sensible defaults.

| Field             | Type              | Description                                          |
|-------------------|-------------------|------------------------------------------------------|
| `model`           | `string`          | Model name or alias                                  |
| `systemPrompt`    | `string`          | System prompt override                               |
| `workingDir`      | `string`          | Working directory for the subprocess / agent          |
| `env`             | `map[string]string` | Additional environment variables                   |
| `maxTurns`        | `int`             | Maximum number of agentic turns                      |
| `timeout`         | `duration`        | Overall execution timeout                            |
| `skipPermissions` | `bool`            | Bypass interactive permission prompts                |

### Runner-Specific Options

Each runner extends the common options with its own fields. These are passed through a runner-specific options type that embeds or extends the common options.

#### Claude Code

| Field              | Type       | Description                                      |
|--------------------|------------|--------------------------------------------------|
| `allowedTools`     | `[]string` | Tools the agent may use                          |
| `disallowedTools`  | `[]string` | Tools the agent may not use                      |
| `mcpConfig`        | `string`   | Path to MCP server configuration file            |
| `jsonSchema`       | `string`   | JSON Schema for structured output                |
| `maxBudgetUSD`     | `float64`  | Cost limit                                       |
| `resume`           | `string`   | Session ID to resume                             |
| `continue`         | `bool`     | Continue the most recent session                 |
| `appendSystemPrompt` | `string` | Appended to the default system prompt            |

#### Gemini CLI

| Field              | Type       | Description                                      |
|--------------------|------------|--------------------------------------------------|
| `approvalMode`     | `string`   | `"default"`, `"auto_edit"`, or `"yolo"`          |
| `extensions`       | `[]string` | Extensions to enable                             |
| `mcpServers`       | `[]string` | Allowed MCP server names                         |
| `includeDirectories` | `[]string` | Additional directories to include in workspace |
| `resume`           | `string`   | Session ID to resume (`"latest"` or ID)          |
| `sandbox`          | `bool`     | Run in sandboxed mode                            |

#### Codex CLI

| Field              | Type       | Description                                      |
|--------------------|------------|--------------------------------------------------|
| `sandbox`          | `string`   | `"read-only"`, `"workspace-write"`, `"danger-full-access"` |
| `approval`         | `string`   | `"untrusted"`, `"on-request"`, `"never"`         |
| `outputSchema`     | `string`   | JSON Schema for structured output validation     |
| `images`           | `[]string` | Image file paths for multimodal input            |
| `profile`          | `string`   | Named config profile                             |
| `resume`           | `string`   | Session ID to resume                             |
| `search`           | `bool`     | Enable live web search                           |

#### Ollama

| Field              | Type              | Description                                |
|--------------------|-------------------|--------------------------------------------|
| `baseURL`          | `string`          | API base URL (default `http://localhost:11434`) |
| `format`           | `string`          | `"json"` or a JSON Schema for structured output |
| `keepAlive`        | `duration`        | How long to keep the model loaded in memory |
| `temperature`      | `float64`         | Sampling temperature                       |
| `topK`             | `int`             | Top-K sampling parameter                   |
| `topP`             | `float64`         | Top-P (nucleus) sampling parameter         |
| `seed`             | `int`             | Random seed for reproducibility            |
| `numCtx`           | `int`             | Context window size                        |
| `tools`            | `[]Tool`          | Function definitions for tool calling      |

---

## Result

Returned by `Run` and included in the final `Message` from `RunStream`.

### Common Fields

| Field        | Type     | Description                                         |
|--------------|----------|-----------------------------------------------------|
| `text`       | `string` | Final response text                                 |
| `isError`    | `bool`   | Whether the run ended in error                      |
| `exitCode`   | `int`    | Process exit code (CLI runners) or 0 (API runners)  |
| `usage`      | `Usage`  | Token counts                                        |
| `costUSD`    | `float64`| Estimated cost in USD (0 for local runners)         |
| `durationMs` | `int64`  | Wall-clock duration in milliseconds                 |
| `sessionID`  | `string` | Session identifier for resumption                   |

### Usage

| Field                    | Type  | Description                         |
|--------------------------|-------|-------------------------------------|
| `inputTokens`            | `int` | Prompt / input tokens consumed      |
| `outputTokens`           | `int` | Completion / output tokens generated |
| `cacheCreationInputTokens` | `int` | Tokens written to cache (if supported) |
| `cacheReadInputTokens`   | `int` | Tokens read from cache (if supported) |

Runners that don't support a field leave it at zero.

---

## Message

The unit of streaming output from `RunStream`. Each message has a type and carries either content or metadata.

### Common Envelope

| Field  | Type     | Description                                         |
|--------|----------|-----------------------------------------------------|
| `type` | `string` | Message type (see taxonomy below)                   |
| `raw`  | `bytes`  | Original JSON line / event for runner-specific parsing |

### Message Type Taxonomy

These types are common across all runners. Runner-specific subtypes are accessed through `raw`.

| Type          | Description                                              | When emitted          |
|---------------|----------------------------------------------------------|-----------------------|
| `system`      | Initialization metadata (model, session ID, tools)       | Start of stream       |
| `assistant`   | Model-generated content (text, thinking, tool calls)     | During generation     |
| `tool_use`    | The model is invoking a tool                             | During agentic turns  |
| `tool_result` | Output from a tool execution                             | After tool execution  |
| `result`      | Final result with text, usage, cost, duration            | End of stream         |
| `error`       | Error or warning                                         | Any time              |

### Typed Accessors

Each message type has typed accessor methods/properties for its data, avoiding the need to parse `raw` for common fields:

| Accessor          | Available on   | Returns                                    |
|-------------------|----------------|--------------------------------------------|
| `Text()`          | `assistant`, `result` | The text content                      |
| `Thinking()`      | `assistant`    | Thinking/reasoning content (if present)    |
| `ToolName()`      | `tool_use`     | Name of the tool being called              |
| `ToolInput()`     | `tool_use`     | Tool call arguments (as string or bytes)   |
| `ToolOutput()`    | `tool_result`  | Tool execution output                      |
| `Result()`        | `result`       | The full `Result` struct                   |
| `IsError()`       | `error`, `result` | Whether this is an error                |
| `ErrorMessage()`  | `error`        | Error description                          |

---

## Stream

The return type of `RunStream` and `Session.messages` — a language-native streaming primitive.

| Language   | Type                       | Notes                              |
|------------|----------------------------|------------------------------------|
| Go         | `<-chan Message`           | Channel-based streaming            |
| TypeScript | `AsyncIterable<Message>`   | `for await (const msg of stream)`  |
| Python     | `AsyncIterator[Message]`   | `async for msg in stream`          |
| Java       | `Stream<Message>`          | Or `Flux<Message>` for reactive    |

The stream must be safe to abandon mid-iteration. Abandoning a stream should terminate the underlying process or cancel the request.

---

## Error Handling

Each language should use its native error conventions:

| Language   | Convention                                                    |
|------------|---------------------------------------------------------------|
| Go         | Return `error` as second value; sentinel errors for common cases |
| TypeScript | Throw typed error classes; reject promises                    |
| Python     | Raise typed exceptions                                        |
| Java       | Throw checked/unchecked exceptions                            |

### Common Error Categories

Implementations should distinguish these error categories using language-appropriate mechanisms (sentinel errors, error types, exception classes):

| Category      | Description                                          |
|---------------|------------------------------------------------------|
| `NotFound`    | Runner binary or API endpoint not reachable          |
| `Timeout`     | Execution exceeded the timeout                       |
| `NonZeroExit` | CLI process exited with non-zero code                |
| `ParseError`  | Failed to parse runner output                        |
| `Cancelled`   | Execution was cancelled by the caller                |

---

## Language-Idiomatic Expression

### Go

```go
type Session struct {
    Messages <-chan Message
}

func (s *Session) Result() (*Result, error)  // blocks until done
func (s *Session) Abort()                    // cancels context + kills process
func (s *Session) Send(input any) error      // returns ErrNotSupported

type Runner interface {
    Start(ctx context.Context, prompt string, opts ...Option) *Session
    Run(ctx context.Context, prompt string, opts ...Option) (*Result, error)
    RunStream(ctx context.Context, prompt string, opts ...Option) (<-chan Message, <-chan error)
}

// Options via functional options pattern
type Option func(*Options)

func WithModel(model string) Option
func WithSystemPrompt(prompt string) Option
func WithWorkingDir(dir string) Option
func WithTimeout(d time.Duration) Option
// ... etc

// Runner-specific options (e.g. for Claude)
func WithAllowedTools(tools ...string) Option
func WithMCPConfig(path string) Option
```

### TypeScript

```typescript
interface Session {
  messages: AsyncIterable<Message>;
  result: Promise<Result>;
  abort(): void;
  send(input: unknown): void; // reserved — throws "not yet supported"
}

interface Runner {
  start(prompt: string, options?: RunOptions): Session;
  run(prompt: string, options?: RunOptions): Promise<Result>;
  runStream(prompt: string, options?: RunOptions): AsyncIterable<Message>;
}

// Options as a typed object with optional fields
interface RunOptions {
  model?: string;
  systemPrompt?: string;
  workingDir?: string;
  timeout?: number;
  // ...common fields
}

// Runner-specific options extend the base
interface ClaudeRunOptions extends RunOptions {
  allowedTools?: string[];
  mcpConfig?: string;
  // ...
}
```

### Python

```python
class Session:
    async def __aiter__(self) -> AsyncIterator[Message]: ...
    result: Awaitable[Result]
    def abort(self) -> None: ...
    def send(self, input: Any) -> None: ...  # reserved — raises NotImplementedError

class Runner(Protocol):
    def start(self, prompt: str, **options: Unpack[RunOptions]) -> Session: ...
    async def run(self, prompt: str, **options: Unpack[RunOptions]) -> Result: ...
    def run_stream(self, prompt: str, **options: Unpack[RunOptions]) -> AsyncIterator[Message]: ...

@dataclass
class RunOptions:
    model: str | None = None
    system_prompt: str | None = None
    working_dir: str | None = None
    timeout: float | None = None
    # ...common fields

# Runner-specific options extend the base
@dataclass
class ClaudeRunOptions(RunOptions):
    allowed_tools: list[str] | None = None
    mcp_config: str | None = None
    # ...
```

### Java

```java
public interface Session {
    Iterator<Message> messages();
    CompletableFuture<Result> result();
    void abort();
    void send(Object input); // reserved — throws UnsupportedOperationException
}

public interface Runner {
    Session start(String prompt, RunOptions options);
    Result run(String prompt, RunOptions options);
    Stream<Message> runStream(String prompt, RunOptions options);
}

// Options via builder pattern
RunOptions options = RunOptions.builder()
    .model("claude-sonnet-4-20250514")
    .systemPrompt("You are a helpful assistant.")
    .workingDir("/path/to/project")
    .timeout(Duration.ofMinutes(5))
    .build();

// Runner-specific options extend the base
ClaudeRunOptions options = ClaudeRunOptions.builder()
    .model("claude-sonnet-4-20250514")
    .allowedTools(List.of("Read", "Write"))
    .build();
```

---

## Runner Registration

Each language library provides factory functions or constructors for creating runners:

```
NewClaudeRunner(binaryPath?) → Runner
NewGeminiRunner(binaryPath?) → Runner
NewCodexRunner(binaryPath?)  → Runner
NewOllamaRunner(baseURL?)    → Runner
```

The binary path / base URL defaults to the standard location but can be overridden for testing or custom installations.
