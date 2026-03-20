# agentrunner/go

Go library for programmatically invoking AI coding agents. Currently supports [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code).

## Requirements

- Go 1.22+
- Claude Code CLI >= 1.0.12

## Installation

```bash
go get github.com/driangle/agent-runner/go
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"

	agentrunner "github.com/driangle/agent-runner/go"
	"github.com/driangle/agent-runner/go/claudecode"
)

func main() {
	runner := claudecode.NewRunner()

	result, err := runner.Run(context.Background(), "What files are in the current directory?",
		agentrunner.WithMaxTurns(3),
		agentrunner.WithSkipPermissions(true),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result.Text)
	fmt.Printf("Cost: $%.4f | Tokens: %d in, %d out\n",
		result.CostUSD, result.Usage.InputTokens, result.Usage.OutputTokens)
}
```

## API Overview

### Common Interface

All runners implement `agentrunner.Runner`:

```go
type Runner interface {
    Run(ctx context.Context, prompt string, opts ...Option) (*Result, error)
    RunStream(ctx context.Context, prompt string, opts ...Option) (<-chan Message, <-chan error)
}
```

- **`Run`** — execute a prompt and block until completion.
- **`RunStream`** — execute a prompt and stream messages as they arrive.

### Result

```go
type Result struct {
    Text       string   // Final response text
    IsError    bool     // Whether the run ended in error
    ExitCode   int      // Process exit code
    Usage      Usage    // Token counts (input, output, cache)
    CostUSD    float64  // Estimated cost in USD
    DurationMs int64    // Wall-clock duration in milliseconds
    SessionID  string   // Session ID for resumption
}
```

### Common Options

```go
agentrunner.WithModel("claude-sonnet-4-6")
agentrunner.WithSystemPrompt("You are a helpful assistant")
agentrunner.WithAppendSystemPrompt("Be concise")
agentrunner.WithWorkingDir("/path/to/project")
agentrunner.WithEnv(map[string]string{"KEY": "value"})
agentrunner.WithMaxTurns(5)
agentrunner.WithTimeout(30 * time.Second)
agentrunner.WithSkipPermissions(true)
```

### Claude Code Options

```go
claudecode.WithAllowedTools("Read", "Bash")
claudecode.WithDisallowedTools("Write")
claudecode.WithMCPConfig("/path/to/mcp.json")
claudecode.WithJSONSchema(`{"type": "object", "properties": {"answer": {"type": "string"}}}`)
claudecode.WithMaxBudgetUSD(1.0)
```

## Usage Examples

### Basic Run

```go
runner := claudecode.NewRunner()

result, err := runner.Run(ctx, "Explain what this project does",
    agentrunner.WithWorkingDir("/path/to/project"),
    agentrunner.WithMaxTurns(1),
)
if err != nil {
    log.Fatal(err)
}
fmt.Println(result.Text)
```

### Streaming

```go
runner := claudecode.NewRunner()

msgCh, errCh := runner.RunStream(ctx, "Refactor the main function",
    agentrunner.WithModel("claude-sonnet-4-6"),
    agentrunner.WithSkipPermissions(true),
)

for msg := range msgCh {
    fmt.Printf("[%s] %s\n", msg.Type, string(msg.Raw))
}
if err := <-errCh; err != nil {
    log.Fatal(err)
}
```

### Streaming with Callback

```go
runner := claudecode.NewRunner()

msgCh, errCh := runner.RunStream(ctx, "Write unit tests",
    claudecode.WithOnMessage(func(msg agentrunner.Message) {
        // Called for each message before it's sent on the channel.
        // Use for logging, progress bars, etc.
        fmt.Printf("  > %s\n", msg.Type)
    }),
)

for range msgCh {
}
if err := <-errCh; err != nil {
    log.Fatal(err)
}
```

### Session Resume

```go
runner := claudecode.NewRunner()

// First run — capture the session ID.
result, err := runner.Run(ctx, "Set up the project structure")
if err != nil {
    log.Fatal(err)
}
sessionID := result.SessionID

// Resume the same session later.
result, err = runner.Run(ctx, "Now add tests",
    claudecode.WithResume(sessionID),
)
if err != nil {
    log.Fatal(err)
}

// Or continue the most recent session.
result, err = runner.Run(ctx, "Fix the failing test",
    claudecode.WithContinue(true),
)
```

### Error Handling

```go
result, err := runner.Run(ctx, "do something")
switch {
case errors.Is(err, agentrunner.ErrNotFound):
    log.Fatal("claude CLI not found — install it first")
case errors.Is(err, agentrunner.ErrTimeout):
    log.Fatal("execution timed out")
case errors.Is(err, agentrunner.ErrCancelled):
    log.Fatal("execution was cancelled")
case errors.Is(err, agentrunner.ErrNonZeroExit):
    log.Fatalf("CLI exited with error: %v", err)
case errors.Is(err, agentrunner.ErrNoResult):
    log.Fatal("no result message received")
case err != nil:
    log.Fatal(err)
}

// Also check result-level errors (the CLI returned a result but flagged it as an error).
if result.IsError {
    log.Fatalf("agent error: %s", result.Text)
}
```

### Structured Output

```go
runner := claudecode.NewRunner()

schema := `{
    "type": "object",
    "properties": {
        "summary": {"type": "string"},
        "language": {"type": "string"},
        "loc": {"type": "integer"}
    },
    "required": ["summary", "language", "loc"]
}`

result, err := runner.Run(ctx, "Analyze this codebase",
    agentrunner.WithWorkingDir("/path/to/project"),
    claudecode.WithJSONSchema(schema),
)
if err != nil {
    log.Fatal(err)
}
// result.Text contains JSON matching the schema.
fmt.Println(result.Text)
```

## Testing

```bash
cd go && go test -race ./...
```

Tests use a mock `CommandBuilder` to simulate CLI output without requiring the real `claude` binary. See `claudecode/runner_test.go` for examples of how to write tests with the mock builder.

## Releasing

Go modules are published automatically by the Go module proxy when a tag exists. This project uses an `agentrunner/v*` tag convention (matching the module subdirectory) to enable independent releases per language.

To release a new version:

1. Ensure `main` is clean and all checks pass (`make check-go`).
2. Tag the release:
   ```bash
   git tag agentrunner/v0.1.0
   git push origin agentrunner/v0.1.0
   ```
3. The `publish-go` GitHub Actions workflow will:
   - Run `make check-go` to validate the module.
   - Verify the module is fetchable on the Go module proxy.
   - Create a GitHub Release with auto-generated notes.
4. The module will be available on [pkg.go.dev](https://pkg.go.dev/github.com/driangle/agent-runner/agentrunner) shortly after.
