# Claude Code Example

Demonstrates the agentrunner Go library with the [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code), covering:

1. **Simple Run** — single prompt, print result fields
2. **Streaming** — real-time message streaming with `Start()` and `--include-partial-messages`
3. **Session Resume** — multi-turn conversation via session IDs
4. **Session Object** — full lifecycle control with `Start()`, message iteration, and `session.Result()`

## Prerequisites

- [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code) installed (>= 1.0.12)
- Authenticated with `claude login`
- Go >= 1.22

## Setup

```sh
go mod tidy
```

## Run

```sh
go run .
```

### Options

```
--claude <path>   Path to the Claude Code CLI binary (default: "claude")
--verbose         Enable debug logging
```

### Examples

```sh
# Use a custom binary path
go run . --claude /usr/local/bin/claude

# Enable debug logging to see the exact CLI command
go run . --verbose
```
