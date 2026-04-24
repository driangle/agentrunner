# Ollama Example

Demonstrates the agentrunner TypeScript library with [Ollama](https://ollama.com), covering:

1. **Simple Run** — single prompt, print result fields
2. **Streaming** — real-time token streaming with `runStream()` and system prompt/temperature options
3. **Thinking Model** — streaming with thinking-enabled models (e.g. qwen3), displaying reasoning and final answer
4. **Session Object** — full lifecycle control with `start()`, message iteration, and `session.result`

## Prerequisites

- [Ollama](https://ollama.com) installed and running
- A model pulled (e.g. `ollama pull llama3.2`)
- Node.js >= 18

## Setup

```sh
npm install
```

## Run

```sh
npx tsx main.ts --model llama3.2
```

### Options

```
--model <name>       Ollama model name (required)
--base-url <url>     Ollama API base URL (default: "http://localhost:11434")
--verbose            Enable debug logging
```

### Examples

```sh
# Use a different model
npx tsx main.ts --model codellama

# Use a custom Ollama server
npx tsx main.ts --model llama3.2 --base-url http://192.168.1.100:11434

# Enable debug logging
npx tsx main.ts --verbose
```
