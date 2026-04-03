# Runner Interface

All agentrunner libraries implement a common, language-agnostic Runner interface. Each language expresses this interface idiomatically, but the conceptual shape is the same.

## Runner

A Runner executes prompts against an AI coding agent and returns results. The primary primitive is `Start`, which returns a **Session** object. `Run` and `RunStream` are convenience wrappers over `Start`.

```
Runner:
  Start(prompt, options)     → Session
  Run(prompt, options)       → Result
  RunStream(prompt, options) → Stream<Message>
```

- **Start** launches an agent process and returns a Session for full lifecycle control.
- **Run** starts a session, drains all messages internally, and returns the final Result.
- **RunStream** starts a session and returns the messages iterable.

## Session

A Session encapsulates a running agent process.

```
Session:
  messages → Stream<Message>   — iterable of messages as they arrive
  result   → Result            — blocks/resolves when the agent finishes
  abort()  → void              — terminates the agent process
  send()   → error             — send a message to the running agent (requires channelEnabled)
```

See the [Channels guide](/guide/channels) for details on two-way communication.

| Language   | Messages                  | Result                      | Abort             |
|------------|---------------------------|-----------------------------|--------------------|
| Go         | `<-chan Message`          | `Result() (*Result, error)` | `Abort()`          |
| TypeScript | `AsyncIterable<Message>`  | `Promise<Result>`           | `abort(): void`    |
| Python     | `async for msg in session`| `await session.result`      | `session.abort()`  |

Sessions are safe to abandon — abandoning a session terminates the underlying process.

## Common Options

Every runner accepts these options. All are optional.

| Field             | Type                | Description                            |
|-------------------|---------------------|----------------------------------------|
| `model`           | `string`            | Model name or alias                    |
| `systemPrompt`    | `string`            | System prompt override                 |
| `workingDir`      | `string`            | Working directory for the subprocess   |
| `env`             | `map[string]string` | Additional environment variables       |
| `maxTurns`        | `int`               | Maximum number of agentic turns        |
| `timeout`         | `duration`          | Overall execution timeout              |
| `skipPermissions` | `bool`              | Bypass interactive permission prompts  |

Each runner extends these with runner-specific options. See the individual [runner pages](/runners/claude-code) for details.

## Result

Returned by `Run` and available via `Session.Result()`.

| Field        | Type     | Description                                        |
|--------------|----------|----------------------------------------------------|
| `text`       | `string` | Final response text                                |
| `isError`    | `bool`   | Whether the run ended in error                     |
| `exitCode`   | `int`    | Process exit code (CLI runners) or 0 (API runners) |
| `usage`      | `Usage`  | Token counts                                       |
| `costUSD`    | `float`  | Estimated cost in USD (0 for local runners)        |
| `durationMs` | `int64`  | Wall-clock duration in milliseconds                |
| `sessionID`  | `string` | Session identifier for resumption                  |

### Usage

| Field                        | Type  | Description                              |
|------------------------------|-------|------------------------------------------|
| `inputTokens`               | `int` | Prompt / input tokens consumed           |
| `outputTokens`              | `int` | Completion / output tokens generated     |
| `cacheCreationInputTokens`  | `int` | Tokens written to cache (if supported)   |
| `cacheReadInputTokens`      | `int` | Tokens read from cache (if supported)    |

## Message

The unit of streaming output. Each message has a type and carries content or metadata.

| Field  | Type     | Description                                           |
|--------|----------|-------------------------------------------------------|
| `type` | `string` | Message type                                          |
| `raw`  | `bytes`  | Original JSON for runner-specific parsing              |

### Message Types

| Type          | Description                                          | When emitted         |
|---------------|------------------------------------------------------|----------------------|
| `system`      | Initialization metadata (model, session ID, tools)   | Start of stream      |
| `assistant`   | Model-generated content (text, thinking, tool calls) | During generation    |
| `tool_use`    | The model is invoking a tool                         | During agentic turns |
| `tool_result` | Output from a tool execution                         | After tool execution |
| `result`      | Final result with text, usage, cost, duration        | End of stream        |
| `error`       | Error or warning                                     | Any time             |

### Typed Accessors

| Accessor         | Available on          | Returns                          |
|------------------|-----------------------|----------------------------------|
| `Text()`         | `assistant`, `result` | The text content                 |
| `Thinking()`     | `assistant`           | Thinking/reasoning content       |
| `ToolName()`     | `tool_use`            | Name of the tool being called    |
| `ToolInput()`    | `tool_use`            | Tool call arguments              |
| `ToolOutput()`   | `tool_result`         | Tool execution output            |
| `IsError()`      | `error`, `result`     | Whether this is an error         |
| `ErrorMessage()` | `error`               | Error description                |

## Error Handling

Each language uses its native error conventions. All implementations distinguish these categories:

| Category      | Description                             |
|---------------|-----------------------------------------|
| `NotFound`    | Runner binary or API endpoint not found |
| `Timeout`     | Execution exceeded the timeout          |
| `NonZeroExit` | CLI process exited with non-zero code   |
| `ParseError`  | Failed to parse runner output           |
| `Cancelled`   | Execution was cancelled by the caller   |
