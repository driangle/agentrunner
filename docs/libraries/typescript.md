# TypeScript Library

The TypeScript library lives under `ts/` and ships as an ESM-only npm package.

## Installation

```sh
npm install agentrunner
```

Requires Node.js >= 18.

## Creating a Runner

### Claude Code

```ts
import { createClaudeRunner } from "agentrunner/claudecode";

const runner = createClaudeRunner();

// With config
const runner = createClaudeRunner({
  binary: "/usr/local/bin/claude",
  logger: console,
});
```

### Ollama

```ts
import { createOllamaRunner } from "agentrunner/ollama";

const runner = createOllamaRunner();

// With config
const runner = createOllamaRunner({
  baseURL: "http://localhost:11434",
  logger: console,
});
```

## Running a Prompt

```ts
const result = await runner.run("What is 2+2?", {
  model: "claude-sonnet-4-20250514",
  timeout: 30_000,
});

console.log(result.text);
console.log(`Cost: $${result.costUSD.toFixed(4)}`);
console.log(`Tokens: ${result.usage.inputTokens} in, ${result.usage.outputTokens} out`);
```

## Streaming

### Using start (full session control)

```ts
import { messageText, messageToolName } from "agentrunner/claudecode";

const session = runner.start("Explain this codebase", {
  workingDir: "/path/to/project",
});

for await (const msg of session.messages) {
  switch (msg.type) {
    case "assistant":
      process.stdout.write(messageText(msg) ?? "");
      break;
    case "tool_use":
      console.log(`\n[tool: ${messageToolName(msg)}]`);
      break;
  }
}

const result = await session.result;
```

### Using runStream (convenience wrapper)

```ts
for await (const msg of runner.runStream("Explain this codebase")) {
  const text = messageText(msg);
  if (text) process.stdout.write(text);
}
```

## Common Options

```ts
await runner.run("prompt", {
  model: "claude-sonnet-4-20250514",
  systemPrompt: "You are a helpful assistant.",
  workingDir: "/path/to/project",
  env: { DEBUG: "1" },
  maxTurns: 10,
  timeout: 300_000,
  signal: controller.signal,
});
```

## Claude Code Options

```ts
await runner.run("prompt", {
  skipPermissions: true,
  allowedTools: ["Read", "Write", "Bash"],
  disallowedTools: ["WebSearch"],
  mcpConfig: "/path/to/mcp.json",
  jsonSchema: '{"type": "object", ...}',
  maxBudgetUSD: 1.0,
  resume: "session-id",
  continueSession: true,
  includePartialMessages: true,
  channelEnabled: true,
  channelLogFile: "/tmp/channel.log",
  channelLogLevel: "debug",
  onMessage: (msg) => console.log(msg.type),
});
```

See the [Channels guide](/guide/channels) for two-way communication details.

## Ollama Options

```ts
await runner.run("prompt", {
  temperature: 0.7,
  numCtx: 4096,
  numPredict: 256,
  seed: 42,
  stop: ["END"],
  topK: 40,
  topP: 0.9,
  format: "json",
  keepAlive: "5m",
  think: true,
  onMessage: (msg) => console.log(msg.type),
});
```

## Session Resume

```ts
// First run — capture the session ID
const result = await runner.run("Start a new task");
const sessionId = result.sessionId;

// Later — resume the session
const resumed = await runner.run("Continue where you left off", {
  resume: sessionId,
});
```

## Error Handling

Errors are thrown as typed error classes:

```ts
import {
  NotFoundError,
  TimeoutError,
  NonZeroExitError,
  CancelledError,
} from "agentrunner";

try {
  await runner.run("prompt");
} catch (err) {
  if (err instanceof NotFoundError) {
    console.error("CLI binary not found");
  } else if (err instanceof TimeoutError) {
    console.error("Execution timed out");
  } else if (err instanceof NonZeroExitError) {
    console.error(`CLI exited with code ${err.exitCode}`);
  } else if (err instanceof CancelledError) {
    console.error("Cancelled");
  }
}
```

## Message Accessors

Claude Code provides typed accessor functions:

```ts
import {
  messageText,
  messageThinking,
  messageToolName,
  messageToolInput,
  messageToolOutput,
  messageIsError,
  messageTextDelta,
  messageThinkingDelta,
} from "agentrunner/claudecode";

messageText(msg);          // text content
messageThinking(msg);      // reasoning content
messageToolName(msg);      // tool being called
messageToolInput(msg);     // tool arguments
messageToolOutput(msg);    // tool result
messageIsError(msg);       // is this an error?
messageTextDelta(msg);     // incremental text (partial messages)
messageThinkingDelta(msg); // incremental thinking (partial messages)
```

## Exports

The package uses subpath exports:

```ts
import { ... } from "agentrunner";           // common types, errors
import { ... } from "agentrunner/claudecode"; // Claude Code runner
import { ... } from "agentrunner/ollama";     // Ollama runner
```

## Logging

Pass any object with `debug` and `error` methods:

```ts
const runner = createClaudeRunner({
  logger: console,
});
```
