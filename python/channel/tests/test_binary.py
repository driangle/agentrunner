"""Tests for channel binary resolution."""

import os

import pytest

from agentrunner_channel import binary_path


class TestBinaryPath:
    def test_env_override(self, monkeypatch):
        monkeypatch.setenv("AGENTRUNNER_CHANNEL_BIN", "/custom/agentrunner-channel")
        assert binary_path() == "/custom/agentrunner-channel"

    def test_not_found_raises(self, monkeypatch):
        monkeypatch.delenv("AGENTRUNNER_CHANNEL_BIN", raising=False)
        monkeypatch.setattr("shutil.which", lambda _: None)
        with pytest.raises(FileNotFoundError, match="agentrunner-channel binary not found"):
            binary_path()
