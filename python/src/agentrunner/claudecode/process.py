"""Process helpers: logging, error collection, spawn resolution."""

from __future__ import annotations

import asyncio
import re

from .options import ClaudeRunnerConfig, SpawnFn

_NEEDS_QUOTING = re.compile(r'[\s"\'\\]')


def log_cmd(
    config: ClaudeRunnerConfig,
    binary: str,
    args: list[str],
    cwd: str | None = None,
) -> None:
    """Log the command about to be executed."""
    if not config.logger:
        return

    parts: list[str] = []
    for a in [binary, *args]:
        if _NEEDS_QUOTING.search(a):
            parts.append(f"'{a}'")
        else:
            parts.append(a)

    config.logger.debug("executing CLI command", extra={"cmd": " ".join(parts), "dir": cwd or ""})


async def _default_spawn(
    program: str,
    *args: str,
    cwd: str | None = None,
    env: dict[str, str] | None = None,
) -> asyncio.subprocess.Process:
    return await asyncio.create_subprocess_exec(
        program,
        *args,
        cwd=cwd,
        env=env,
        stdin=asyncio.subprocess.DEVNULL,
        stdout=asyncio.subprocess.PIPE,
        stderr=asyncio.subprocess.PIPE,
    )


def resolve_spawn(config: ClaudeRunnerConfig) -> tuple[SpawnFn, str]:
    """Resolve the spawn function and binary name."""
    binary = config.binary or "claude"
    spawn_fn: SpawnFn = config.spawn or _default_spawn
    return spawn_fn, binary


def collect_error_detail(stderr: str, stdout_errors: list[str]) -> str:
    """Build a human-readable error string from stderr and unparseable stdout lines."""
    parts: list[str] = []
    trimmed = stderr.strip()
    if trimmed:
        parts.append(trimmed)
    if stdout_errors:
        parts.append("\n".join(stdout_errors))
    if not parts:
        return "unknown error (no output from CLI)"
    return "\n".join(parts)
