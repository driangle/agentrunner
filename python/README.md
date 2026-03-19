# agentrunner (Python)

Python library for programmatically invoking AI coding agents. Part of the [agentrunner](../) monorepo.

## Supported CLIs

| Runner      | CLI Version | Status |
|-------------|-------------|--------|
| Claude Code | >= 1.0.12   | ✅      |

## Requirements

- Python >= 3.11
- Claude Code CLI >= 1.0.12

## Installation

```bash
pip install agentrunner
```

## Quick Start

```python
import asyncio
from agentrunner.claudecode import create_claude_runner, ClaudeRunOptions

runner = create_claude_runner()

async def main():
    # Simple run
    result = await runner.run("What files are in this directory?", ClaudeRunOptions(
        working_dir="/path/to/project",
        skip_permissions=True,
    ))
    print(result.text)

    # Streaming
    stream = await runner.run_stream("Explain this codebase")
    async for message in stream:
        print(message.type, message.raw)

asyncio.run(main())
```

## API

### `create_claude_runner(config?)`

Creates a runner for the Claude Code CLI.

**Config options (`ClaudeRunnerConfig`):**

| Field    | Type       | Default    | Description                          |
|----------|------------|------------|--------------------------------------|
| `binary` | `str`      | `"claude"` | CLI binary name or path              |
| `spawn`  | `SpawnFn`  | —          | Custom spawn function (for testing)  |
| `logger` | `Logger`   | —          | Logger for debug output (opt-in)     |

### `runner.run(prompt, options?)`

Execute a prompt and return the final `Result`.

### `runner.run_stream(prompt, options?)`

Execute a prompt and stream messages as they arrive. Returns `AsyncIterable[Message]`.

### `runner.start(prompt, options?)`

Launch an agent process and return a `Session` for full lifecycle control.

### Run Options

Common options (`RunOptions`):

| Field                 | Type             | Description                          |
|-----------------------|------------------|--------------------------------------|
| `model`               | `str`            | Model name or alias                  |
| `system_prompt`       | `str`            | System prompt override               |
| `append_system_prompt`| `str`            | Appended to default system prompt    |
| `working_dir`         | `str`            | Working directory for subprocess     |
| `env`                 | `dict[str, str]` | Additional environment variables     |
| `max_turns`           | `int`            | Maximum agentic turns                |
| `timeout`             | `float`          | Timeout in milliseconds              |
| `skip_permissions`    | `bool`           | Skip permission prompts              |

Claude-specific options (`ClaudeRunOptions` extends `RunOptions`):

| Field                    | Type        | Description                        |
|--------------------------|-------------|------------------------------------|
| `allowed_tools`          | `list[str]` | Tools the agent may use            |
| `disallowed_tools`       | `list[str]` | Tools the agent may not use        |
| `mcp_config`             | `str`       | Path to MCP server config          |
| `json_schema`            | `str`       | JSON Schema for structured output  |
| `max_budget_usd`         | `float`     | Cost limit in USD                  |
| `resume`                 | `str`       | Session ID to resume               |
| `continue_session`       | `bool`      | Continue most recent session       |
| `session_id`             | `str`       | Specific session ID                |
| `include_partial_messages`| `bool`     | Stream partial/incremental messages|
| `on_message`             | `callable`  | Callback for each streamed message |

### Result

| Field        | Type    | Description                     |
|--------------|---------|---------------------------------|
| `text`       | `str`   | Final response text             |
| `is_error`   | `bool`  | Whether the run ended in error  |
| `exit_code`  | `int`   | Process exit code               |
| `usage`      | `Usage` | Token counts                    |
| `cost_usd`   | `float` | Estimated cost in USD           |
| `duration_ms`| `float` | Wall-clock duration in ms       |
| `session_id` | `str`   | Session ID for resumption       |

### Session

| Attribute  | Type                     | Description                           |
|------------|--------------------------|---------------------------------------|
| `messages` | `AsyncIterable[Message]` | Iterate messages as they arrive       |
| `result`   | `Future[Result]`         | Resolves when the agent finishes      |
| `abort()`  | —                        | Terminate the agent process           |
| `send()`   | —                        | Reserved (raises `RuntimeError`)      |

### Error Classes

All errors extend `RunnerError`:

- `NotFoundError` — CLI binary not found
- `TimeoutError` — execution timed out
- `NonZeroExitError` — CLI exited with non-zero code (has `.exit_code`)
- `ParseError` — failed to parse CLI output
- `CancelledError` — execution cancelled
- `NoResultError` — stream ended without a result message

```python
from agentrunner import TimeoutError

try:
    await runner.run("complex task", ClaudeRunOptions(timeout=30_000))
except TimeoutError:
    print("Timed out!")
```

## Usage Examples

### Session Resume

```python
# First run — capture the session ID.
result = await runner.run("Set up the project structure")
session_id = result.session_id

# Resume the same session later.
result = await runner.run("Now add tests", ClaudeRunOptions(resume=session_id))
```

### Session Object

```python
session = runner.start("Explain this code", ClaudeRunOptions(max_turns=1, timeout=30_000))

async for msg in session.messages:
    print(f"[{msg.type}] {msg.raw[:80]}")

result = await session.result
print(f"Response: {result.text}")
```

### Streaming with Partial Messages

```python
from agentrunner.claudecode import parse

stream = await runner.run_stream("List fun facts", ClaudeRunOptions(
    include_partial_messages=True,
))
async for msg in stream:
    if msg.type == "assistant":
        parsed = parse(msg.raw)
        if parsed.type == "stream_event":
            import json
            raw = json.loads(msg.raw)
            delta = raw.get("event", {}).get("delta", {})
            if delta.get("type") == "text_delta":
                print(delta["text"], end="", flush=True)
```

## Development

```bash
cd python
pip install -e ".[dev]"   # install with dev dependencies
ruff check src/ tests/    # lint
python -m pytest           # run tests
```

Or from the repo root:

```bash
make check-python  # build + lint + test
make check         # all libraries
```
