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
    # Exclusive whitelist of built-in tools (--tools). Unlike allowed_tools,
    # which only pre-approves tools and leaves unlisted built-ins available,
    # --tools replaces the built-in tool set, so every unlisted built-in is
    # denied at registration. Use [""] to disable all tools, ["default"] for
    # the full set, or explicit names. None leaves the CLI default.
    tools: list[str] | None = None
    # Permission mode for the run (--permission-mode). Known modes: "default",
    # "acceptEdits", "auto", "plan", "dontAsk", "bypassPermissions". Passed
    # through as a plain string so new CLI modes work without a library change.
    # When set, takes precedence over dangerously_skip_permissions.
    permission_mode: str | None = None
    mcp_config: str | None = None
    json_schema: str | None = None
    # Additional settings for the run (--settings). May be a path to a settings
    # JSON file or a raw JSON string; passed through verbatim, the CLI
    # disambiguates. In non-interactive mode the CLI silently ignores settings
    # that fail validation.
    settings: str | None = None
    max_budget_usd: float | None = None
    resume: str | None = None
    continue_session: bool = False
    session_id: str | None = None
    include_partial_messages: bool = False
