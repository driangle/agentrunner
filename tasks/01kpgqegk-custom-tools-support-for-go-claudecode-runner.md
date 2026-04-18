---
title: "Custom tools support for Go ClaudeCode runner"
id: "01kpgqegk"
status: pending
priority: high
type: feature
tags: ["go", "claudecode", "mcp"]
effort: large
created: "2026-04-18"
---

# Custom tools support for Go ClaudeCode runner

## Objective

Allow clients to define custom tools (name, description, input schema, handler function) that the Claude CLI agent can call. The library should automatically spin up a dynamic MCP server exposing these tools, so that when Claude invokes a tool, the call is routed back to the client's Go handler. This enables use cases like progress notifications, custom approval gates, data lookups, etc. — similar to how tool definitions work in the OpenAI and Anthropic SDKs, but adapted for the CLI-based architecture.

## Background

### SDK tool definition patterns

| SDK | Schema Type | Notes |
|-----|------------|-------|
| **OpenAI Go** | `map[string]any` (alias `FunctionParameters`) | Fully untyped |
| **OpenAI TS** | `Record<string, unknown>` | Fully untyped |
| **Anthropic Go** | `ToolInputSchemaParam` struct with `Properties any`, `Required []string`, `ExtraFields map[string]any` | Semi-typed, enforces `type: "object"` |
| **Anthropic TS** | `Tool.InputSchema` interface with `type: 'object'`, `properties?: unknown`, `[k: string]: unknown` | Semi-typed with index signature |

### Proposed Go API

```go
progressTool := claudecode.Tool{
    Name:        "notify_progress",
    Description: "Notify the caller about progress on the current task",
    InputSchema: claudecode.InputSchema{
        Properties: map[string]any{
            "message": map[string]any{
                "type":        "string",
                "description": "Progress message",
            },
        },
        Required: []string{"message"},
    },
    Handler: func(ctx context.Context, input json.RawMessage) (string, error) {
        var params struct{ Message string `json:"message"` }
        json.Unmarshal(input, &params)
        fmt.Println("Progress:", params.Message)
        return "acknowledged", nil
    },
}

result, err := runner.Run(ctx, "Do something complex",
    claudecode.WithTools(progressTool),
)
```

Design choice: Use an `InputSchema` struct (like Anthropic's approach) with `Properties map[string]any` and `Required []string`. This is more ergonomic than raw `json.RawMessage` — callers build schemas with Go map literals rather than embedding JSON strings. The struct enforces `type: "object"` (since MCP tools always take object params) while `Properties map[string]any` keeps property definitions flexible.

### Architecture

Extends the existing `agentrunner-channel` MCP server infrastructure:

1. **Tool registration**: On startup, write tool definitions (name, description, inputSchema) to a JSON file. Pass path via `AGENTRUNNER_TOOLS_FILE` env var.
2. **Bidirectional socket protocol**: Extend the Unix socket from one-way (Go → MCP server for channel messages) to request/response. MCP server sends `tool_call` requests over socket, Go process dispatches to handlers and replies.
3. **MCP server changes**: `agentrunner-channel` reads tool definitions on init, exposes them via `tools/list`, and forwards `tools/call` over the socket.
4. **Runner integration**: `setupChannel` (or new `setupTools`) writes the tool defs file, starts a socket listener with a tool call dispatcher, and registers handlers.

## Tasks

- [ ] Define `Tool` and `InputSchema` types in `claudecode/` package
- [ ] Add `WithTools(tools ...Tool)` option function
- [ ] Define bidirectional socket protocol for tool call request/response
- [ ] Extend `agentrunner-channel` binary to read tool definitions from `AGENTRUNNER_TOOLS_FILE`
- [ ] Extend `agentrunner-channel` to expose custom tools via `tools/list` alongside existing `reply` tool
- [ ] Extend `agentrunner-channel` to forward `tools/call` for custom tools over the Unix socket and wait for response
- [ ] Implement Go-side socket listener that dispatches tool calls to handlers
- [ ] Wire tool setup into runner's `Start()` method (tool defs file, socket listener, MCP config)
- [ ] Ensure custom tools and channel features can coexist (same MCP server, same socket)
- [ ] Add unit tests for `Tool`/`InputSchema` types and serialization
- [ ] Add unit tests for bidirectional socket protocol
- [ ] Add integration tests with a fake MCP server exercising the full tool call round-trip
- [ ] Add example program demonstrating custom tool usage
- [ ] Run `make check-go` and ensure all checks pass

## Acceptance Criteria

- Client can define tools with name, description, typed input schema, and a handler function
- `InputSchema` uses `Properties map[string]any` and `Required []string` (Anthropic-style semi-typed approach)
- Library automatically spins up a dynamic MCP server exposing client-defined tools
- When Claude calls a custom tool, the handler executes in the client's Go process and the result is returned to Claude
- Custom tools and channel features work together without conflict
- Tool definitions are validated at startup (e.g. name required, handler required)
- Existing tests continue to pass, new unit and integration tests cover the feature
- `make check-go` passes
