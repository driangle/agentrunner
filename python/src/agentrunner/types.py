"""Common types shared across all runners."""

from __future__ import annotations

import asyncio
from collections.abc import AsyncIterable, Callable
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
    """The unit of streaming output from run_stream."""

    type: str
    raw: str


@dataclass
class RunOptions:
    """Common options for a runner invocation."""

    model: str | None = None
    system_prompt: str | None = None
    append_system_prompt: str | None = None
    working_dir: str | None = None
    env: dict[str, str] | None = None
    max_turns: int | None = None
    timeout: float | None = None
    skip_permissions: bool = False


class Session(Protocol):
    """Session encapsulates a running agent process."""

    @property
    def messages(self) -> AsyncIterable[Message]: ...

    @property
    def result(self) -> asyncio.Future[Result]: ...

    def abort(self) -> None: ...

    def send(self, input: Any) -> None: ...


OnMessageFn = Callable[[Message], None]


class Runner(Protocol):
    """Runner executes prompts against an AI coding agent."""

    def start(self, prompt: str, options: RunOptions | None = None) -> Session: ...

    async def run(self, prompt: str, options: RunOptions | None = None) -> Result: ...

    def run_stream(
        self, prompt: str, options: RunOptions | None = None
    ) -> AsyncIterable[Message]: ...
