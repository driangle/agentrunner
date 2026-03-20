"""Configuration and option types for the Claude Code runner."""

from __future__ import annotations

import asyncio
from dataclasses import dataclass
from typing import Protocol

from ..types import RunOptions


class Logger(Protocol):
    """Logger interface for debug output. Opt-in, disabled by default."""

    def debug(self, message: str, *args: object, **kwargs: object) -> None: ...
    def error(self, message: str, *args: object, **kwargs: object) -> None: ...


class SpawnFn(Protocol):
    """Function that spawns an async subprocess. Used for dependency injection in tests."""

    async def __call__(
        self,
        program: str,
        *args: str,
        cwd: str | None = None,
        env: dict[str, str] | None = None,
    ) -> asyncio.subprocess.Process: ...


@dataclass
class ClaudeRunnerConfig:
    """Configuration for creating a Claude Code runner."""

    binary: str = "claude"
    spawn: SpawnFn | None = None
    logger: Logger | None = None


@dataclass
class ClaudeRunOptions(RunOptions):
    """Claude Code-specific options that extend the common RunOptions."""

    allowed_tools: list[str] | None = None
    disallowed_tools: list[str] | None = None
    mcp_config: str | None = None
    json_schema: str | None = None
    max_budget_usd: float | None = None
    resume: str | None = None
    continue_session: bool = False
    session_id: str | None = None
    include_partial_messages: bool = False
