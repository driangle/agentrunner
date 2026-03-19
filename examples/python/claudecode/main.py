#!/usr/bin/env python3
"""Example: using the agentrunner Python library to invoke Claude Code CLI.

Covers basic run, streaming with partial messages, session resume,
and the Session object pattern.

Prerequisites:
  - Claude Code CLI installed (>= 1.0.12): https://docs.anthropic.com/en/docs/claude-code
  - Authenticated with `claude login`

Run:
  cd examples/python/claudecode
  pip install -e .
  python main.py
  python main.py --binary /path/to/claude
"""

import argparse
import asyncio
import logging
import sys

from agentrunner.claudecode import (
    ClaudeRunner,
    ClaudeRunnerConfig,
    ClaudeRunOptions,
    create_claude_runner,
    parse,
)


async def example_simple_run(runner: ClaudeRunner) -> None:
    """Send a single prompt and print the result."""
    prompt = "What is 2+2? Reply with just the number."
    print(f"Prompt:   {prompt}")

    result = await runner.run(prompt, ClaudeRunOptions(max_turns=1, timeout=30_000))

    print(f"Response: {result.text}")
    print(f"Cost:     ${result.cost_usd:.4f}")
    print(f"Tokens:   {result.usage.input_tokens} in / {result.usage.output_tokens} out")
    print(f"Duration: {result.duration_ms}ms")
    print(f"Session:  {result.session_id}")
    print(f"Error:    {result.is_error}")
    print(f"Exit:     {result.exit_code}")


async def example_streaming(runner: ClaudeRunner, verbose: bool) -> None:
    """Use run_stream to print messages as they arrive."""
    prompt = "List 3 fun facts about Python. Be brief."
    print(f"Prompt: {prompt}")
    print("---")

    model = "unknown"

    stream = await runner.run_stream(
        prompt, ClaudeRunOptions(max_turns=1, timeout=30_000, include_partial_messages=True)
    )
    async for msg in stream:
        if msg.type == "system":
            if verbose:
                print(f"[system] {msg.raw}")
            parsed = parse(msg.raw)
            if parsed.model:
                model = parsed.model

        elif msg.type == "assistant":
            # With --include-partial-messages, the CLI emits two kinds of
            # messages mapped to "assistant":
            #   1. stream_event with content_block_delta — real-time text deltas
            #   2. assistant — full accumulated message (arrives at the end)
            # Print deltas for real-time streaming; skip the final assistant
            # message to avoid duplicating the output.
            parsed = parse(msg.raw)
            if parsed.type == "stream_event" and parsed.event:
                # Access delta from the raw JSON for text deltas.
                import json

                raw = json.loads(msg.raw)
                event = raw.get("event", {})
                delta = event.get("delta", {})
                if delta.get("type") == "text_delta" and delta.get("text"):
                    sys.stdout.write(delta["text"])
                    sys.stdout.flush()

        elif msg.type == "result":
            parsed = parse(msg.raw)
            print("\n---")
            print(f"Cost:     ${parsed.total_cost_usd or 0:.4f}")
            print(f"Duration: {parsed.duration_ms or 0}ms")
            print(f"Turns:    {parsed.num_turns or 0}")
            print(f"Model:    {parsed.model or model}")
            print(f"Session:  {parsed.session_id or ''}")
            print(f"Error:    {parsed.is_error or False}")


async def example_session_resume(runner: ClaudeRunner) -> None:
    """Demonstrate multi-turn conversations using session IDs."""
    # First turn: ask Claude to remember something.
    prompt1 = "Remember this number: 42. Just confirm you've noted it."
    print(f"Prompt 1: {prompt1}")

    first = await runner.run(prompt1, ClaudeRunOptions(max_turns=1, timeout=30_000))

    print(f"Response: {first.text}")
    print(f"Session:  {first.session_id}")

    if not first.session_id:
        raise RuntimeError("No session ID returned — cannot demonstrate resume")

    # Second turn: resume the session and reference the earlier context.
    prompt2 = "What number did I ask you to remember?"
    print(f"\nPrompt 2: {prompt2} (resume: {first.session_id})")

    second = await runner.run(
        prompt2,
        ClaudeRunOptions(max_turns=1, timeout=30_000, resume=first.session_id),
    )

    print(f"Response: {second.text}")


async def example_session(runner: ClaudeRunner) -> None:
    """Demonstrate the Session object pattern with full lifecycle control."""
    prompt = "What is the capital of France? Reply with just the city name."
    print(f"Prompt: {prompt}")

    session = runner.start(prompt, ClaudeRunOptions(max_turns=1, timeout=30_000))

    # Iterate messages as they arrive.
    async for msg in session.messages:
        preview = msg.raw[:80] + "..." if len(msg.raw) > 80 else msg.raw
        print(f"[{msg.type}] {preview}")

    # Get the final result.
    result = await session.result

    print(f"Response: {result.text}")
    print(f"Cost:     ${result.cost_usd:.4f}")
    print(f"Session:  {result.session_id}")


async def main() -> None:
    parser = argparse.ArgumentParser(description="agentrunner Claude Code example")
    parser.add_argument("--binary", default="claude", help="path to Claude Code CLI binary")
    parser.add_argument("--verbose", action="store_true", help="enable debug logging")
    args = parser.parse_args()

    logger = None
    if args.verbose:
        logging.basicConfig(level=logging.DEBUG, stream=sys.stderr)
        logger = logging.getLogger("agentrunner.claudecode")

    runner = create_claude_runner(ClaudeRunnerConfig(binary=args.binary, logger=logger))

    print("=== Example 1: Simple Run ===")
    await example_simple_run(runner)

    print("\n=== Example 2: Streaming ===")
    await example_streaming(runner, args.verbose)

    print("\n=== Example 3: Session Resume ===")
    await example_session_resume(runner)

    print("\n=== Example 4: Session Object ===")
    await example_session(runner)


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        pass
    except Exception as e:
        print(f"error: {e}", file=sys.stderr)
        sys.exit(1)
