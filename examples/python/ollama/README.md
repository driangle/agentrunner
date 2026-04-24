# Ollama Example

Demonstrates the agentrunner Python library with [Ollama](https://ollama.com), covering:

1. **Simple Run** — single prompt, print result fields
2. **Streaming** — real-time token streaming with `run_stream()` and system prompt/temperature options
3. **Thinking Model** — streaming with thinking-enabled models (e.g. qwen3), displaying reasoning and final answer
4. **Session Object** — full lifecycle control with `start()`, async iteration, and `session.result`

## Prerequisites

- [Ollama](https://ollama.com) installed and running
- A model pulled (e.g. `ollama pull llama3.2`)
- Python >= 3.11

## Setup

```sh
pip install -e .
```

Or with [uv](https://docs.astral.sh/uv/):

```sh
uv sync
```

## Run

```sh
python main.py --model llama3.2
```

Or with uv:

```sh
uv run main.py --model llama3.2
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
python main.py --model codellama

# Use a custom Ollama server
python main.py --model llama3.2 --base-url http://192.168.1.100:11434

# Enable debug logging
python main.py --verbose
```
