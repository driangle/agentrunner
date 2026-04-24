# Ollama Example

Demonstrates the agentrunner Go library with [Ollama](https://ollama.com), covering:

1. **Simple Run** — single prompt, print result fields
2. **Streaming** — real-time token streaming with `Start()` and system prompt/temperature options
3. **Thinking Model** — streaming with thinking-enabled models (e.g. qwen3), displaying reasoning and final answer
4. **Session Object** — full lifecycle control with `Start()`, message iteration, and `session.Result()`

## Prerequisites

- [Ollama](https://ollama.com) installed and running
- A model pulled (e.g. `ollama pull llama3.2`)
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
--model <name>       Ollama model name (default: "llama3.2")
--base-url <url>     Ollama API base URL (default: "http://localhost:11434")
--verbose            Enable debug logging
```

### Examples

```sh
# Use a different model
go run . --model codellama

# Use a custom Ollama server
go run . --model llama3.2 --base-url http://192.168.1.100:11434

# Enable debug logging
go run . --verbose
```
