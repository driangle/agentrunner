# Codex CLI Example

Demonstrates the agentrunner TypeScript library with the [Codex CLI](https://github.com/openai/codex), covering:

1. **Simple Run** — single prompt, print result fields
2. **Streaming** — real-time message streaming with `runStream()` and async iterators
3. **Session Resume** — multi-turn conversation via session IDs
4. **Session Object** — full lifecycle control with `start()`, message iteration, and `session.result`

## Prerequisites

- [Codex CLI](https://github.com/openai/codex) installed (>= 0.118.0)
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
--binary <path>   Path to the Codex CLI binary (default: "codex")
--verbose         Enable debug logging
```

### Examples

```sh
# Use a custom binary path
npx tsx main.ts --binary /usr/local/bin/codex

# Enable debug logging to see the exact CLI command
npx tsx main.ts --verbose
```
