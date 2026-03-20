"""Common types shared across all runners."""

from __future__ import annotations

import asyncio
import json
from collections.abc import AsyncIterator, Callable
from dataclasses import dataclass, field
from typing import Any, Protocol


@dataclass
class Usage:
    """Token consumption counts."""

    input_tokens: int = 0
    output_tokens: int = 0
    cache_creation_input_tokens: int = 0
    cache_read_input_tokens: int = 0


@dataclass
class Result:
    """Final output from a runner invocation."""

    text: str = ""
    is_error: bool = False
    exit_code: int = 0
    usage: Usage = field(default_factory=Usage)
    cost_usd: float = 0.0
    duration_ms: float = 0.0
    session_id: str = ""


@dataclass
class Message:
    """The unit of streaming output from run_stream.

    Provides typed accessors for common fields so callers don't need to
    re-parse ``raw`` JSON for typical use cases.
    """

    type: str
    raw: str

    _parsed: dict[str, Any] | None = field(default=None, repr=False, compare=False)

    def _raw_dict(self) -> dict[str, Any]:
        if self._parsed is None:
            self._parsed = json.loads(self.raw)
        return self._parsed

    def text(self) -> str | None:
        """Extract text content from assistant or result messages."""
        d = self._raw_dict()
        if d.get("type") == "result":
            return d.get("result")
        msg = d.get("message")
        if isinstance(msg, dict):
            for block in msg.get("content", []):
                if isinstance(block, dict) and block.get("type") == "text":
                    return block.get("text")
        # Stream event text delta.
        event = d.get("event")
        if isinstance(event, dict):
            delta = event.get("delta")
            if isinstance(delta, dict) and delta.get("type") == "text_delta":
                return delta.get("text")
        return None

    def thinking(self) -> str | None:
        """Extract thinking content from assistant messages."""
        d = self._raw_dict()
        msg = d.get("message")
        if isinstance(msg, dict):
            for block in msg.get("content", []):
                if isinstance(block, dict) and block.get("type") == "thinking":
                    return block.get("thinking")
        # Stream event thinking delta.
        event = d.get("event")
        if isinstance(event, dict):
            delta = event.get("delta")
            if isinstance(delta, dict) and delta.get("type") == "thinking_delta":
                return delta.get("thinking")
        return None

    def tool_name(self) -> str | None:
        """Extract tool name from tool_use content blocks."""
        d = self._raw_dict()
        msg = d.get("message")
        if isinstance(msg, dict):
            for block in msg.get("content", []):
                if isinstance(block, dict) and block.get("type") == "tool_use":
                    return block.get("name")
        return None

    def tool_input(self) -> Any | None:
        """Extract tool input from tool_use content blocks."""
        d = self._raw_dict()
        msg = d.get("message")
        if isinstance(msg, dict):
            for block in msg.get("content", []):
                if isinstance(block, dict) and block.get("type") == "tool_use":
                    return block.get("input")
        return None

    def tool_output(self) -> str | None:
        """Extract tool output from user/tool_result messages."""
        d = self._raw_dict()
        if d.get("type") == "user":
            for block in d.get("content", []):
                if isinstance(block, dict) and block.get("type") == "tool_result":
                    return block.get("content")
        return None


@dataclass
class RunOptions:
    """Common options for a runner invocation.

    ``timeout`` is in seconds, following Python convention
    (e.g. ``asyncio.wait_for``, ``socket.settimeout``).
    """

    model: str | None = None
    system_prompt: str | None = None
    append_system_prompt: str | None = None
    working_dir: str | None = None
    env: dict[str, str] | None = None
    max_turns: int | None = None
    timeout: float | None = None
    skip_permissions: bool = False


OnMessageFn = Callable[[Message], None]


class Session(Protocol):
    """Session encapsulates a running agent process."""

    def __aiter__(self) -> AsyncIterator[Message]: ...

    @property
    def result(self) -> asyncio.Future[Result]: ...

    def abort(self) -> None: ...

    def send(self, input: Any) -> None: ...


class Runner(Protocol):
    """Runner executes prompts against an AI coding agent."""

    def start(
        self,
        prompt: str,
        options: RunOptions | None = None,
        on_message: OnMessageFn | None = None,
    ) -> Session: ...

    async def run(
        self,
        prompt: str,
        options: RunOptions | None = None,
        on_message: OnMessageFn | None = None,
    ) -> Result: ...

    def run_stream(
        self,
        prompt: str,
        options: RunOptions | None = None,
        on_message: OnMessageFn | None = None,
    ) -> AsyncIterator[Message]: ...
