"""Tests for CLI version detection."""

import pytest

from agentrunner.claudecode.version import MIN_VERSION, _parse_version, check_version
from agentrunner.errors import NotFoundError


class TestParseVersion:
    def test_basic(self):
        assert _parse_version("1.0.12") == (1, 0, 12)

    def test_comparison(self):
        assert _parse_version("1.0.12") < _parse_version("1.1.0")
        assert _parse_version("2.0.0") > _parse_version("1.99.99")

    def test_min_version_is_valid(self):
        parsed = _parse_version(MIN_VERSION)
        assert len(parsed) == 3


class TestCheckVersion:
    async def test_missing_binary_raises(self):
        with pytest.raises(NotFoundError, match="command not found"):
            await check_version("nonexistent-binary-xxxxx")
