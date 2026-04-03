# Claude Code Runner

The Claude Code runner invokes the `claude` CLI in non-interactive print mode (`claude -p`) with `--output-format stream-json`.

## Requirements

- Claude Code CLI >= 1.0.12
- The `claude` binary must be on your `PATH` (or specify a custom path)

## How It Works

The runner spawns `claude -p --output-format stream-json` as a subprocess and parses the newline-delimited JSON stream. Each line is a typed message (system, assistant, tool_use, tool_result, result, error).

## Runner-Specific Options

| Option | Type | Description |
|--------|------|-------------|
| `allowedTools` | `string[]` | Tools the agent may use |
| `disallowedTools` | `string[]` | Tools the agent may not use |
| `mcpConfig` | `string` | Path to MCP server configuration file |
| `jsonSchema` | `string` | JSON Schema for structured output |
| `maxBudgetUSD` | `float` | Cost limit in USD |
| `resume` | `string` | Session ID to resume |
| `continue` | `bool` | Continue the most recent session |
| `appendSystemPrompt` | `string` | Appended to the default system prompt |
| `includePartialMessages` | `bool` | Stream partial/incremental messages |
| `channelEnabled` | `bool` | Enable two-way [channel](/guide/channels) communication |
| `channelLogFile` | `string` | File path for channel server logs |
| `channelLogLevel` | `string` | Channel server log level (`debug`, `info`, `warn`, `error`) |

## Stream Message Types

| Type | Description |
|------|-------------|
| `system` | Init message with model, session_id, tools |
| `assistant` | Text, thinking, or tool_use content |
| `user` | Tool results |
| `result` | Final result with cost, usage, duration |
| `stream_event` | Raw API streaming events |
| `channel_reply` | Reply from the agent to a [channel](/guide/channels) message |
| `rate_limit_event` | Rate limit notifications |

## Examples

### Simple Run

::: code-group

```go [Go]
runner := claudecode.NewRunner()
result, err := runner.Run(ctx, "Summarize the README")
```

```ts [TypeScript]
const runner = createClaudeRunner();
const result = await runner.run("Summarize the README");
```

```python [Python]
runner = ClaudeRunner()
result = await runner.run("Summarize the README")
```

:::

### With Tool Restrictions

::: code-group

```go [Go]
result, err := runner.Run(ctx, "Read the config file",
    claudecode.WithAllowedTools("Read"),
    claudecode.WithDisallowedTools("Write", "Bash"),
)
```

```ts [TypeScript]
const result = await runner.run("Read the config file", {
  allowedTools: ["Read"],
  disallowedTools: ["Write", "Bash"],
});
```

```python [Python]
result = await runner.run("Read the config file", ClaudeRunOptions(
    allowed_tools=["Read"],
    disallowed_tools=["Write", "Bash"],
))
```

:::

### Session Resume

::: code-group

```go [Go]
// First run
result, _ := runner.Run(ctx, "Start a task")

// Resume later
result, _ = runner.Run(ctx, "Continue",
    claudecode.WithResume(result.SessionID),
)
```

```ts [TypeScript]
const result = await runner.run("Start a task");
const resumed = await runner.run("Continue", {
  resume: result.sessionId,
});
```

```python [Python]
result = await runner.run("Start a task")
resumed = await runner.run("Continue", ClaudeRunOptions(
    resume=result.session_id,
))
```

:::

### Streaming with Partial Messages

::: code-group

```go [Go]
session, _ := runner.Start(ctx, "Write a story",
    claudecode.WithIncludePartialMessages(),
)
for msg := range session.Messages {
    fmt.Print(msg.Text())
}
```

```ts [TypeScript]
const session = runner.start("Write a story", {
  includePartialMessages: true,
});
for await (const msg of session.messages) {
  const delta = messageTextDelta(msg);
  if (delta) process.stdout.write(delta);
}
```

```python [Python]
session = runner.start("Write a story", ClaudeRunOptions(
    include_partial_messages=True,
))
async for msg in session:
    if msg.text:
        print(msg.text, end="")
```

:::
