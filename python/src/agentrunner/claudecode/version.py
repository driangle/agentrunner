"""CLI version detection and compatibility check."""

from __future__ import annotations

import asyncio
import re

from ..errors import NotFoundError

# Supported Claude Code CLI version range.
MIN_VERSION = "1.0.12"

_VERSION_RE = re.compile(r"(\d+\.\d+\.\d+)")


def _parse_version(version_str: str) -> tuple[int, ...]:
    """Parse a semver string into a comparable tuple."""
    return tuple(int(p) for p in version_str.split("."))


async def check_version(binary: str) -> str:
    """Run ``<binary> --version`` and verify it meets the minimum requirement.

    Returns the detected version string.
    Raises ``NotFoundError`` if the binary is missing or the version is too old.
    """
    try:
        proc = await asyncio.create_subprocess_exec(
            binary,
            "--version",
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        stdout, _ = await proc.communicate()
    except FileNotFoundError:
        raise NotFoundError(f"{binary}: command not found")

    output = stdout.decode("utf-8", errors="replace").strip()
    match = _VERSION_RE.search(output)
    if not match:
        raise NotFoundError(f"could not parse version from `{binary} --version`: {output!r}")

    version = match.group(1)
    if _parse_version(version) < _parse_version(MIN_VERSION):
        raise NotFoundError(f"{binary} version {version} is below minimum supported {MIN_VERSION}")

    return version
