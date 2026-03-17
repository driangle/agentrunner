# agentrunner (TypeScript)

TypeScript library for programmatically invoking AI coding agents. Part of the [agentrunner](../) monorepo.

## Supported CLIs

| Runner      | CLI Version | Status |
|-------------|-------------|--------|
| Claude Code | >= 1.0.12   | ✅      |

## Installation

```bash
npm install agentrunner
```

## Quick Start

```typescript
import { createClaudeRunner } from "agentrunner/claudecode";

const runner = createClaudeRunner();

// Simple run
const result = await runner.run("What files are in this directory?", {
  workingDir: "/path/to/project",
  skipPermissions: true,
});
console.log(result.text);

// Streaming
for await (const message of runner.runStream("Explain this codebase")) {
  console.log(message.type, message.raw);
}
```

## API

### `createClaudeRunner(config?)`

Creates a runner for the Claude Code CLI.

**Config options:**

| Field    | Type       | Default    | Description                          |
|----------|------------|------------|--------------------------------------|
| `binary` | `string`   | `"claude"` | CLI binary name or path              |
| `spawn`  | `SpawnFn`  | —          | Custom spawn function (for testing)  |
| `logger` | `Logger`   | —          | Logger for debug output (opt-in)     |

### `runner.run(prompt, options?)`

Execute a prompt and return the final result.

### `runner.runStream(prompt, options?)`

Execute a prompt and stream messages as they arrive. Returns `AsyncIterable<Message>`.

### Run Options

Common options (all runners):

| Field               | Type                    | Description                          |
|---------------------|-------------------------|--------------------------------------|
| `model`             | `string`                | Model name or alias                  |
| `systemPrompt`      | `string`                | System prompt override               |
| `appendSystemPrompt`| `string`                | Appended to default system prompt    |
| `workingDir`        | `string`                | Working directory for subprocess     |
| `env`               | `Record<string,string>` | Additional environment variables     |
| `maxTurns`          | `number`                | Maximum agentic turns                |
| `timeout`           | `number`                | Timeout in milliseconds              |
| `signal`            | `AbortSignal`           | Cancellation signal                  |
| `skipPermissions`   | `boolean`               | Skip permission prompts              |

Claude-specific options (extend `RunOptions`):

| Field             | Type       | Description                        |
|-------------------|------------|------------------------------------|
| `allowedTools`    | `string[]` | Tools the agent may use            |
| `disallowedTools` | `string[]` | Tools the agent may not use        |
| `mcpConfig`       | `string`   | Path to MCP server config          |
| `jsonSchema`      | `string`   | JSON Schema for structured output  |
| `maxBudgetUSD`    | `number`   | Cost limit in USD                  |
| `resume`          | `string`   | Session ID to resume               |
| `continue`        | `boolean`  | Continue most recent session       |
| `sessionId`       | `string`   | Specific session ID                |
| `onMessage`       | `function` | Callback for each streamed message |

### Result

| Field        | Type     | Description                     |
|--------------|----------|---------------------------------|
| `text`       | `string` | Final response text             |
| `isError`    | `boolean`| Whether the run ended in error  |
| `exitCode`   | `number` | Process exit code               |
| `usage`      | `Usage`  | Token counts                    |
| `costUSD`    | `number` | Estimated cost in USD           |
| `durationMs` | `number` | Wall-clock duration in ms       |
| `sessionId`  | `string` | Session ID for resumption       |

### Error Classes

All errors extend `RunnerError`:

- `NotFoundError` — CLI binary not found
- `TimeoutError` — execution timed out
- `NonZeroExitError` — CLI exited with non-zero code
- `ParseError` — failed to parse CLI output
- `CancelledError` — execution cancelled via AbortSignal
- `NoResultError` — stream ended without a result message

```typescript
import { TimeoutError } from "agentrunner";

try {
  await runner.run("complex task", { timeout: 30_000 });
} catch (err) {
  if (err instanceof TimeoutError) {
    console.log("Timed out!");
  }
}
```

## Development

```bash
npm install    # install dependencies
npm run build  # compile TypeScript
npm run lint   # type-check only
npm test       # run tests
```

Or from the repo root:

```bash
make check-ts  # build + lint + test
make check     # all libraries
```
