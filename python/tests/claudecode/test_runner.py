"""Tests for the Claude Code runner with mock spawn."""

import asyncio
import json

import pytest

from agentrunner import (
    CancelledError,
    ClaudeRunner,
    Message,
    NonZeroExitError,
    NoResultError,
    TimeoutError,
)
from agentrunner.claudecode.options import ClaudeRunOptions


class FakeProcess:
    """A fake asyncio.subprocess.Process for testing."""

    def __init__(self, lines: list[str], exit_code: int = 0, stderr_text: str = "") -> None:
        self._exit_code = exit_code
        self._killed = False

        # Build stdout data.
        stdout_data = "\n".join(lines) + "\n" if lines else ""
        self._stdout_reader = asyncio.StreamReader()
        self._stdout_reader.feed_data(stdout_data.encode())
        self._stdout_reader.feed_eof()

        self._stderr_data = stderr_text.encode()
        self._stderr_reader = asyncio.StreamReader()
        self._stderr_reader.feed_data(self._stderr_data)
        self._stderr_reader.feed_eof()

        self.stdout = self._stdout_reader
        self.stderr = self._stderr_reader
        self.returncode: int | None = None
        self._wait_event = asyncio.Event()
        self._wait_event.set()  # Completes immediately by default.

    async def wait(self) -> int:
        await self._wait_event.wait()
        self.returncode = self._exit_code
        return self._exit_code

    def kill(self) -> None:
        self._killed = True
        self.returncode = self._exit_code
        self._wait_event.set()


class SlowFakeProcess:
    """A fake process that never completes on its own (for timeout/cancel tests)."""

    def __init__(self) -> None:
        self._stdout_reader = asyncio.StreamReader()
        self._stderr_reader = asyncio.StreamReader()
        self._stderr_reader.feed_data(b"")
        self._stderr_reader.feed_eof()
        self.stdout = self._stdout_reader
        self.stderr = self._stderr_reader
        self.returncode: int | None = None
        self._wait_event = asyncio.Event()

    async def wait(self) -> int:
        await self._wait_event.wait()
        self.returncode = -9
        return -9

    def kill(self) -> None:
        self.returncode = -9
        # Feed EOF to unblock readline.
        self._stdout_reader.feed_eof()
        self._wait_event.set()


def mock_spawn(lines: list[str], exit_code: int = 0, stderr_text: str = ""):
    """Create a mock spawn function returning a FakeProcess."""

    async def _spawn(
        program: str,
        *args: str,
        cwd: str | None = None,
        env: dict[str, str] | None = None,
    ) -> FakeProcess:
        return FakeProcess(lines, exit_code, stderr_text)

    return _spawn


def slow_spawn():
    """Create a mock spawn that returns a process that never completes."""
    proc: SlowFakeProcess | None = None

    async def _spawn(
        program: str,
        *args: str,
        cwd: str | None = None,
        env: dict[str, str] | None = None,
    ) -> SlowFakeProcess:
        nonlocal proc
        proc = SlowFakeProcess()
        return proc

    return _spawn


HAPPY_LINES = [
    json.dumps(
        {
            "type": "system",
            "subtype": "init",
            "session_id": "sess-1",
            "model": "claude-sonnet-4-6",
        }
    ),
    json.dumps(
        {
            "type": "assistant",
            "message": {
                "model": "claude-sonnet-4-6",
                "id": "msg_01",
                "content": [{"type": "text", "text": "Hello world"}],
            },
        }
    ),
    json.dumps(
        {
            "type": "result",
            "subtype": "success",
            "result": "Hello world",
            "is_error": False,
            "total_cost_usd": 0.05,
            "duration_ms": 1234,
            "duration_api_ms": 1100,
            "num_turns": 2,
            "session_id": "sess-1",
            "usage": {
                "input_tokens": 100,
                "output_tokens": 50,
                "cache_creation_input_tokens": 10,
                "cache_read_input_tokens": 20,
            },
        }
    ),
]

STREAM_MULTI_LINES = [
    json.dumps(
        {
            "type": "system",
            "subtype": "init",
            "session_id": "sess-s1",
            "model": "claude-sonnet-4-6",
        }
    ),
    json.dumps(
        {
            "type": "stream_event",
            "event": {
                "type": "content_block_delta",
                "index": 0,
                "delta": {"type": "text_delta", "text": "Hello"},
            },
            "session_id": "sess-s1",
        }
    ),
    json.dumps(
        {
            "type": "stream_event",
            "event": {
                "type": "content_block_delta",
                "index": 0,
                "delta": {"type": "text_delta", "text": " world"},
            },
            "session_id": "sess-s1",
        }
    ),
    json.dumps(
        {
            "type": "assistant",
            "message": {
                "model": "claude-sonnet-4-6",
                "id": "msg_01",
                "content": [{"type": "text", "text": "Hello world"}],
            },
        }
    ),
    json.dumps(
        {
            "type": "result",
            "subtype": "success",
            "result": "Hello world",
            "is_error": False,
            "total_cost_usd": 0.05,
            "duration_ms": 500,
            "session_id": "sess-s1",
            "usage": {"input_tokens": 100, "output_tokens": 50},
        }
    ),
]


class TestRun:
    async def test_happy_path(self):
        runner = ClaudeRunner(_spawn=mock_spawn(HAPPY_LINES))
        result = await runner.run("say hello")

        assert result.text == "Hello world"
        assert result.session_id == "sess-1"
        assert result.cost_usd == 0.05
        assert result.duration_ms == 1234
        assert result.is_error is False
        assert result.usage.input_tokens == 100
        assert result.usage.output_tokens == 50
        assert result.usage.cache_creation_input_tokens == 10
        assert result.usage.cache_read_input_tokens == 20

    async def test_error_result(self):
        lines = [
            json.dumps(
                {
                    "type": "result",
                    "subtype": "error",
                    "result": "Something failed",
                    "is_error": True,
                    "session_id": "sess-err",
                    "usage": {"input_tokens": 10, "output_tokens": 5},
                }
            ),
        ]
        runner = ClaudeRunner(_spawn=mock_spawn(lines))
        result = await runner.run("fail please")

        assert result.is_error is True
        assert result.text == "Something failed"
        assert result.session_id == "sess-err"

    async def test_no_result_raises(self):
        lines = [json.dumps({"type": "system", "subtype": "init", "session_id": "sess-x"})]
        runner = ClaudeRunner(_spawn=mock_spawn(lines))

        with pytest.raises(NoResultError):
            await runner.run("hello")

    async def test_non_zero_exit_raises(self):
        runner = ClaudeRunner(_spawn=mock_spawn([], exit_code=1, stderr_text="fatal error"))

        with pytest.raises(NonZeroExitError) as exc_info:
            await runner.run("hello")
        assert "fatal error" in str(exc_info.value)

    async def test_timeout_raises(self):
        runner = ClaudeRunner(_spawn=slow_spawn())

        with pytest.raises(TimeoutError):
            await runner.run("hello", ClaudeRunOptions(timeout=0.05))

    async def test_session_id_fallback_to_init(self):
        lines = [
            json.dumps(
                {
                    "type": "system",
                    "subtype": "init",
                    "session_id": "sess-from-init",
                    "model": "claude-sonnet-4-6",
                }
            ),
            json.dumps(
                {
                    "type": "result",
                    "subtype": "success",
                    "result": "done",
                    "is_error": False,
                    "total_cost_usd": 0.01,
                    "duration_ms": 100,
                    "usage": {"input_tokens": 10, "output_tokens": 5},
                }
            ),
        ]
        runner = ClaudeRunner(_spawn=mock_spawn(lines))
        result = await runner.run("hello")

        assert result.session_id == "sess-from-init"


class TestRunStream:
    async def test_happy_path_yields_messages(self):
        runner = ClaudeRunner(_spawn=mock_spawn(STREAM_MULTI_LINES))
        messages: list[Message] = []

        session = await runner.run_stream("say hello")
        async for msg in session:
            messages.append(msg)

        assert len(messages) == 5
        assert messages[0].type == "system"
        assert messages[-1].type == "result"
        # Middle messages should be assistant (stream_event maps to assistant).
        for msg in messages[1:-1]:
            assert msg.type == "assistant"

    async def test_message_ordering(self):
        runner = ClaudeRunner(_spawn=mock_spawn(STREAM_MULTI_LINES))
        types: list[str] = []

        session = await runner.run_stream("test ordering")
        async for msg in session:
            types.append(msg.type)

        assert types[0] == "system"
        assert types[-1] == "result"

    async def test_no_result_raises(self):
        lines = [
            json.dumps({"type": "system", "subtype": "init", "session_id": "sess-nr"}),
            json.dumps(
                {
                    "type": "assistant",
                    "message": {
                        "model": "claude-sonnet-4-6",
                        "id": "msg_01",
                        "content": [{"type": "text", "text": "partial"}],
                    },
                }
            ),
        ]
        runner = ClaudeRunner(_spawn=mock_spawn(lines))
        messages: list[Message] = []

        session = await runner.run_stream("partial")
        async for msg in session:
            messages.append(msg)

        assert len(messages) == 2
        with pytest.raises(NoResultError):
            await session.result

    async def test_timeout_raises(self):
        runner = ClaudeRunner(_spawn=slow_spawn())

        session = await runner.run_stream("hello", ClaudeRunOptions(timeout=0.05))
        async for _msg in session:
            pass

        with pytest.raises(TimeoutError):
            await session.result

    async def test_non_zero_exit_raises(self):
        runner = ClaudeRunner(
            _spawn=mock_spawn([], exit_code=1, stderr_text="fatal error")
        )

        session = await runner.run_stream("hello")
        async for _msg in session:
            pass

        with pytest.raises(NonZeroExitError):
            await session.result

    async def test_result_accessible_after_iteration(self):
        """run_stream returns a session with accessible result."""
        runner = ClaudeRunner(_spawn=mock_spawn(STREAM_MULTI_LINES))

        session = await runner.run_stream("test result")
        async for _msg in session:
            pass

        result = await session.result
        assert result.text == "Hello world"
        assert result.cost_usd == 0.05

    async def test_raw_json_populated(self):
        runner = ClaudeRunner(_spawn=mock_spawn(STREAM_MULTI_LINES))

        session = await runner.run_stream("test raw")
        async for msg in session:
            assert len(msg.raw) > 0


class TestStart:
    async def test_happy_path(self):
        runner = ClaudeRunner(_spawn=mock_spawn(HAPPY_LINES))
        session = runner.start("say hello")

        messages: list[Message] = []
        async for msg in session:
            messages.append(msg)

        result = await session.result

        assert len(messages) == 3
        assert messages[0].type == "system"
        assert messages[-1].type == "result"
        assert result.text == "Hello world"
        assert result.session_id == "sess-1"
        assert result.cost_usd == 0.05

    async def test_session_direct_iteration(self):
        """Session supports ``async for msg in session`` directly."""
        runner = ClaudeRunner(_spawn=mock_spawn(HAPPY_LINES))
        session = runner.start("say hello")

        messages: list[Message] = []
        async for msg in session:
            messages.append(msg)

        result = await session.result
        assert len(messages) == 3
        assert result.text == "Hello world"

    async def test_abort_terminates(self):
        runner = ClaudeRunner(_spawn=slow_spawn())
        session = runner.start("long task", ClaudeRunOptions(timeout=5))

        # Abort after a brief delay.
        await asyncio.sleep(0.05)
        session.abort()

        messages: list[Message] = []
        async for msg in session:
            messages.append(msg)

        with pytest.raises((CancelledError, TimeoutError)):
            await session.result

    async def test_send_raises_not_implemented(self):
        runner = ClaudeRunner(_spawn=mock_spawn(HAPPY_LINES))
        session = runner.start("hello")

        with pytest.raises(NotImplementedError, match="not yet supported"):
            session.send("test")

        # Drain to avoid resource leak.
        async for _msg in session:
            pass

    async def test_session_id_fallback(self):
        lines = [
            json.dumps(
                {
                    "type": "system",
                    "subtype": "init",
                    "session_id": "sess-from-init",
                    "model": "claude-sonnet-4-6",
                }
            ),
            json.dumps(
                {
                    "type": "result",
                    "subtype": "success",
                    "result": "done",
                    "is_error": False,
                    "total_cost_usd": 0.01,
                    "duration_ms": 100,
                    "usage": {"input_tokens": 10, "output_tokens": 5},
                }
            ),
        ]
        runner = ClaudeRunner(_spawn=mock_spawn(lines))
        session = runner.start("hello")

        async for _msg in session:
            pass

        result = await session.result
        assert result.session_id == "sess-from-init"


class TestMessageAccessors:
    """Tests for the typed property accessors on Message."""

    def test_text_from_assistant(self):
        raw = json.dumps(
            {
                "type": "assistant",
                "message": {
                    "model": "claude-sonnet-4-6",
                    "id": "msg_01",
                    "content": [{"type": "text", "text": "Hello world"}],
                },
            }
        )
        msg = Message(type="assistant", raw=raw)
        assert msg.text == "Hello world"

    def test_text_from_result(self):
        raw = json.dumps({"type": "result", "result": "The answer"})
        msg = Message(type="result", raw=raw)
        assert msg.text == "The answer"

    def test_text_from_stream_event_delta(self):
        raw = json.dumps(
            {
                "type": "stream_event",
                "event": {
                    "type": "content_block_delta",
                    "index": 0,
                    "delta": {"type": "text_delta", "text": "chunk"},
                },
            }
        )
        msg = Message(type="assistant", raw=raw)
        assert msg.text == "chunk"

    def test_thinking_from_assistant(self):
        raw = json.dumps(
            {
                "type": "assistant",
                "message": {
                    "model": "m",
                    "id": "id",
                    "content": [{"type": "thinking", "thinking": "Let me think..."}],
                },
            }
        )
        msg = Message(type="assistant", raw=raw)
        assert msg.thinking == "Let me think..."

    def test_tool_name_and_input(self):
        raw = json.dumps(
            {
                "type": "assistant",
                "message": {
                    "model": "m",
                    "id": "id",
                    "content": [{"type": "tool_use", "name": "Read", "input": {"path": "/tmp"}}],
                },
            }
        )
        msg = Message(type="assistant", raw=raw)
        assert msg.tool_name == "Read"
        assert msg.tool_input == {"path": "/tmp"}

    def test_tool_output(self):
        raw = json.dumps(
            {
                "type": "user",
                "content": [{"type": "tool_result", "content": "file contents here"}],
            }
        )
        msg = Message(type="tool_result", raw=raw)
        assert msg.tool_output == "file contents here"

    def test_no_text_returns_none(self):
        raw = json.dumps({"type": "system", "subtype": "init"})
        msg = Message(type="system", raw=raw)
        assert msg.text is None

    def test_parsed_cache(self):
        raw = json.dumps({"type": "result", "result": "cached"})
        msg = Message(type="result", raw=raw)
        # First call parses.
        assert msg.text == "cached"
        # Second call uses cache (same _parsed dict).
        assert msg._parsed is not None
        assert msg.text == "cached"

    def test_is_error_from_error_result(self):
        raw = json.dumps({"type": "result", "result": "Something failed", "is_error": True})
        msg = Message(type="result", raw=raw)
        assert msg.is_error is True

    def test_is_error_false_by_default(self):
        raw = json.dumps({"type": "assistant", "message": {"content": []}})
        msg = Message(type="assistant", raw=raw)
        assert msg.is_error is False

    def test_error_message(self):
        raw = json.dumps({"type": "result", "result": "Something failed", "is_error": True})
        msg = Message(type="result", raw=raw)
        assert msg.error_message == "Something failed"

    def test_error_message_none_when_not_error(self):
        raw = json.dumps({"type": "result", "result": "OK", "is_error": False})
        msg = Message(type="result", raw=raw)
        assert msg.error_message is None


class TestMessageTypeMapping:
    """Tests for INTERFACE.md message type taxonomy."""

    async def test_tool_result_type(self):
        """User messages containing tool_result blocks are mapped to 'tool_result'."""
        lines = [
            json.dumps(
                {
                    "type": "system",
                    "subtype": "init",
                    "session_id": "sess-t1",
                    "model": "claude-sonnet-4-6",
                }
            ),
            json.dumps(
                {
                    "type": "user",
                    "content": [{"type": "tool_result", "content": "file contents"}],
                }
            ),
            json.dumps(
                {
                    "type": "result",
                    "subtype": "success",
                    "result": "done",
                    "is_error": False,
                    "total_cost_usd": 0.01,
                    "duration_ms": 100,
                    "usage": {"input_tokens": 10, "output_tokens": 5},
                }
            ),
        ]
        runner = ClaudeRunner(_spawn=mock_spawn(lines))
        messages: list[Message] = []

        async for msg in runner.start("test"):
            messages.append(msg)

        assert messages[1].type == "tool_result"

    async def test_plain_user_type_preserved(self):
        """User messages without tool_result stay as 'user'."""
        lines = [
            json.dumps(
                {
                    "type": "system",
                    "subtype": "init",
                    "session_id": "sess-t2",
                    "model": "claude-sonnet-4-6",
                }
            ),
            json.dumps(
                {
                    "type": "user",
                    "content": [{"type": "text", "text": "hello"}],
                }
            ),
            json.dumps(
                {
                    "type": "result",
                    "subtype": "success",
                    "result": "done",
                    "is_error": False,
                    "total_cost_usd": 0.01,
                    "duration_ms": 100,
                    "usage": {"input_tokens": 10, "output_tokens": 5},
                }
            ),
        ]
        runner = ClaudeRunner(_spawn=mock_spawn(lines))
        messages: list[Message] = []

        async for msg in runner.start("test"):
            messages.append(msg)

        assert messages[1].type == "user"
