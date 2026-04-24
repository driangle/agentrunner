# Gemini CLI Example

Demonstrates the agentrunner TypeScript library with the [Gemini CLI](https://github.com/google-gemini/gemini-cli), covering:

1. **Simple Run** — single prompt, print result fields
2. **Streaming** — real-time message streaming with `runStream()` and async iterators
3. **Session Resume** — multi-turn conversation via session IDs
4. **Session Object** — full lifecycle control with `start()`, message iteration, and `session.result`

## Prerequisites

- [Gemini CLI](https://github.com/google-gemini/gemini-cli) installed (>= 0.1.0)
- Authenticated with Gemini CLI
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
--binary <path>   Path to the Gemini CLI binary (default: "gemini")
--verbose         Enable debug logging
```

### Examples

```sh
# Use a custom binary path
npx tsx main.ts --binary /usr/local/bin/gemini

# Enable debug logging to see the exact CLI command
npx tsx main.ts --verbose
```
