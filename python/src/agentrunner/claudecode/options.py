"""Configuration and option types for the Claude Code runner."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Protocol

from ..types import RunOptions


class Logger(Protocol):
    """Logger interface for debug output. Opt-in, disabled by default.

    Compatible with ``logging.getLogger()``.
    """

    def debug(self, message: str, *args: object, **kwargs: object) -> None: ...
    def error(self, message: str, *args: object, **kwargs: object) -> None: ...


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
