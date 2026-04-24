# Claude Code Example

Demonstrates the agentrunner Python library with the [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code), covering:

1. **Simple Run** — single prompt, print result fields
2. **Streaming** — real-time message streaming with `run_stream()` and `include_partial_messages`
3. **Session Resume** — multi-turn conversation via session IDs
4. **Session Object** — full lifecycle control with `start()`, async iteration, and `session.result`

## Prerequisites

- [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code) installed (>= 1.0.12)
- Authenticated with `claude login`
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
python main.py
```

Or with uv:

```sh
uv run main.py
```

### Options

```
--binary <path>   Path to the Claude Code CLI binary (default: "claude")
--verbose         Enable debug logging
```

### Examples

```sh
# Use a custom binary path
python main.py --binary /usr/local/bin/claude

# Enable debug logging to see the exact CLI command
python main.py --verbose
```
