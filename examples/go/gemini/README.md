# Gemini CLI Example

Demonstrates the agentrunner Go library with the [Gemini CLI](https://github.com/google-gemini/gemini-cli), covering:

1. **Simple Run** — single prompt, print result fields
2. **Streaming** — real-time message streaming with `Start()` and message type handling
3. **Session Resume** — multi-turn conversation via session IDs

## Prerequisites

- [Gemini CLI](https://github.com/google-gemini/gemini-cli) installed (>= 0.1.0)
- Authenticated with Gemini CLI
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
--gemini <path>   Path to the Gemini CLI binary (default: "gemini")
--verbose         Enable debug logging
```

### Examples

```sh
# Use a custom binary path
go run . --gemini /usr/local/bin/gemini

# Enable debug logging to see the exact CLI command
go run . --verbose
```
