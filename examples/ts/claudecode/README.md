# Claude Code Example

Demonstrates the agentrunner TypeScript library with the [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code), covering:

1. **Simple Run** — single prompt, print result fields
2. **Streaming** — real-time message streaming with `runStream()` and `includePartialMessages`
3. **Session Resume** — multi-turn conversation via session IDs
4. **Session Object** — full lifecycle control with `start()`, message iteration, and `session.result`

## Prerequisites

- [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code) installed (>= 1.0.12)
- Authenticated with `claude login`
- Node.js >= 18

## Setup

```sh
npm install
```

## Run

```sh
npx tsx main.ts
```

### Options

```
--binary <path>   Path to the Claude Code CLI binary (default: "claude")
--verbose         Enable debug logging
```

### Examples

```sh
# Use a custom binary path
npx tsx main.ts --binary /usr/local/bin/claude

# Enable debug logging to see the exact CLI command
npx tsx main.ts --verbose
```
