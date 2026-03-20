"""Process helpers: logging, error collection, spawn resolution."""

from __future__ import annotations

import asyncio
import shlex
from typing import Protocol

from .options import Logger


class SpawnFn(Protocol):
    """Function that spawns an async subprocess.

    Internal protocol used for dependency injection in tests.
    """

    async def __call__(
        self,
        program: str,
        *args: str,
        cwd: str | None = None,
        env: dict[str, str] | None = None,
    ) -> asyncio.subprocess.Process: ...


def log_cmd(
    logger: Logger | None,
    binary: str,
    args: list[str],
    cwd: str | None = None,
) -> None:
    """Log the command about to be executed."""
    if not logger:
        return

    cmd = " ".join(shlex.quote(a) for a in [binary, *args])
    logger.debug("executing CLI command", extra={"cmd": cmd, "dir": cwd or ""})


async def default_spawn(
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
