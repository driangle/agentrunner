"""Resolve the agentrunner-channel binary path."""

import os
import platform
import shutil
from pathlib import Path


def binary_path() -> str:
    """Return the path to the agentrunner-channel binary.

    Resolution order:
        1. AGENTRUNNER_CHANNEL_BIN environment variable
        2. Bundled binary in this package's bin/ directory
        3. agentrunner-channel on $PATH

    Raises:
        FileNotFoundError: if the binary cannot be found.
    """
    env_path = os.environ.get("AGENTRUNNER_CHANNEL_BIN")
    if env_path:
        return env_path

    name = (
        "agentrunner-channel.exe"
        if platform.system() == "Windows"
        else "agentrunner-channel"
    )
    bundled = Path(__file__).parent / "bin" / name
    if bundled.exists():
        return str(bundled)

    found = shutil.which("agentrunner-channel")
    if found:
        return found

    raise FileNotFoundError(
        "agentrunner-channel binary not found. "
        "Install the agentrunner-channel package for your platform, "
        "add it to $PATH, or set AGENTRUNNER_CHANNEL_BIN."
    )
