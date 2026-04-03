# Go Library

The Go library lives under `go/` and follows standard Go module conventions.

## Installation

```sh
go get github.com/anthropics/agentrunner/go
```

## Creating a Runner

### Claude Code

```go
import "github.com/anthropics/agentrunner/go/claudecode"

runner := claudecode.NewRunner()

// With options
runner := claudecode.NewRunner(
    claudecode.WithBinary("/usr/local/bin/claude"),
    claudecode.WithLogger(slog.Default()),
)
```

### Ollama

```go
import "github.com/anthropics/agentrunner/go/ollama"

runner := ollama.NewRunner()

// With options
runner := ollama.NewRunner(
    ollama.WithBaseURL("http://localhost:11434"),
    ollama.WithLogger(slog.Default()),
)
```

## Running a Prompt

```go
result, err := runner.Run(ctx, "What is 2+2?",
    agentrunner.WithModel("claude-sonnet-4-20250514"),
    agentrunner.WithTimeout(30 * time.Second),
)
if err != nil {
    log.Fatal(err)
}

fmt.Println(result.Text)
fmt.Printf("Cost: $%.4f\n", result.CostUSD)
fmt.Printf("Tokens: %d in, %d out\n", result.Usage.InputTokens, result.Usage.OutputTokens)
```

## Streaming

### Using Start (full session control)

```go
session, err := runner.Start(ctx, "Explain this codebase",
    agentrunner.WithWorkingDir("/path/to/project"),
)
if err != nil {
    log.Fatal(err)
}

for msg := range session.Messages {
    switch msg.Type {
    case agentrunner.MessageTypeAssistant:
        fmt.Print(msg.Text())
    case agentrunner.MessageTypeToolUse:
        fmt.Printf("\n[tool: %s]\n", msg.ToolName())
    case agentrunner.MessageTypeToolResult:
        fmt.Printf("[result: %s]\n", msg.ToolOutput())
    }
}

result, err := session.Result()
```

### Using RunStream (convenience wrapper)

`RunStream` is not a separate method in Go — use `Start` and iterate over `session.Messages`.

## Common Options

All options use the functional options pattern:

```go
agentrunner.WithModel("claude-sonnet-4-20250514")
agentrunner.WithSystemPrompt("You are a helpful assistant.")
agentrunner.WithWorkingDir("/path/to/project")
agentrunner.WithEnv(map[string]string{"DEBUG": "1"})
agentrunner.WithMaxTurns(10)
agentrunner.WithTimeout(5 * time.Minute)
agentrunner.WithSkipPermissions()
```

## Claude Code Options

```go
claudecode.WithAllowedTools("Read", "Write", "Bash")
claudecode.WithDisallowedTools("WebSearch")
claudecode.WithMCPConfig("/path/to/mcp.json")
claudecode.WithJSONSchema(`{"type": "object", ...}`)
claudecode.WithMaxBudgetUSD(1.0)
claudecode.WithResume("session-id")
claudecode.WithContinue()
claudecode.WithIncludePartialMessages()
claudecode.WithChannelEnabled()
claudecode.WithChannelLogFile("/tmp/channel.log")
claudecode.WithChannelLogLevel("debug")
```

See the [Channels guide](/guide/channels) for two-way communication details.

## Ollama Options

```go
ollama.WithTemperature(0.7)
ollama.WithNumCtx(4096)
ollama.WithNumPredict(256)
ollama.WithSeed(42)
ollama.WithStop("END")
ollama.WithTopK(40)
ollama.WithTopP(0.9)
ollama.WithFormat("json")
ollama.WithKeepAlive("5m")
ollama.WithThink(true)
```

## Session Resume

```go
// First run — capture the session ID
result, _ := runner.Run(ctx, "Start a new task")
sessionID := result.SessionID

// Later — resume the session
result, _ = runner.Run(ctx, "Continue where you left off",
    claudecode.WithResume(sessionID),
)
```

## Error Handling

Errors are returned as Go `error` values. Use `errors.Is` to check categories:

```go
import "github.com/anthropics/agentrunner/go"

result, err := runner.Run(ctx, "prompt")
if err != nil {
    switch {
    case errors.Is(err, agentrunner.ErrNotFound):
        log.Fatal("CLI binary not found")
    case errors.Is(err, agentrunner.ErrTimeout):
        log.Fatal("Execution timed out")
    case errors.Is(err, agentrunner.ErrNonZeroExit):
        log.Fatal("CLI exited with error")
    default:
        log.Fatal(err)
    }
}
```

## Message Accessors

```go
msg.Text()         // text content
msg.Thinking()     // reasoning content
msg.ToolName()     // tool being called
msg.ToolInput()    // tool arguments (json.RawMessage)
msg.ToolOutput()   // tool result
msg.IsError()      // is this an error?
msg.ErrorMessage() // error description
```

## Logging

Pass a `*slog.Logger` to enable debug logging of CLI commands:

```go
logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))
runner := claudecode.NewRunner(claudecode.WithLogger(logger))
```
