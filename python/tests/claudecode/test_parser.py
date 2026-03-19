"""Tests for stream-json parser."""

import json

import pytest

from agentrunner.claudecode.parser import parse


class TestParse:
    def test_system_init(self):
        line = json.dumps({"type": "system", "subtype": "init", "session_id": "sess-1"})
        msg = parse(line)
        assert msg.type == "system"
        assert msg.subtype == "init"
        assert msg.session_id == "sess-1"

    def test_assistant_lifts_content(self):
        line = json.dumps({
            "type": "assistant",
            "message": {
                "model": "claude-sonnet-4-6",
                "id": "msg_01",
                "content": [{"type": "text", "text": "Hello world"}],
            },
        })
        msg = parse(line)
        assert msg.type == "assistant"
        assert len(msg.content) == 1
        assert msg.content[0].type == "text"
        assert msg.content[0].text == "Hello world"

    def test_result(self):
        line = json.dumps({
            "type": "result",
            "subtype": "success",
            "result": "Hello world",
            "is_error": False,
            "total_cost_usd": 0.05,
            "duration_ms": 1234,
            "session_id": "sess-1",
            "usage": {
                "input_tokens": 100,
                "output_tokens": 50,
                "cache_creation_input_tokens": 10,
                "cache_read_input_tokens": 20,
            },
        })
        msg = parse(line)
        assert msg.type == "result"
        assert msg.result == "Hello world"
        assert msg.is_error is False
        assert msg.total_cost_usd == 0.05
        assert msg.duration_ms == 1234
        assert msg.session_id == "sess-1"
        assert msg.usage is not None
        assert msg.usage.input_tokens == 100
        assert msg.usage.output_tokens == 50
        assert msg.usage.cache_creation_input_tokens == 10
        assert msg.usage.cache_read_input_tokens == 20

    def test_stream_event(self):
        line = json.dumps({
            "type": "stream_event",
            "event": {
                "type": "content_block_delta",
                "index": 0,
                "delta": {"type": "text_delta", "text": "Hello"},
            },
        })
        msg = parse(line)
        assert msg.type == "stream_event"
        assert msg.event is not None
        assert msg.event.type == "content_block_delta"

    def test_missing_type_raises(self):
        with pytest.raises(ValueError, match="missing type field"):
            parse(json.dumps({"foo": "bar"}))

    def test_invalid_json_raises(self):
        with pytest.raises(json.JSONDecodeError):
            parse("not json at all")

    def test_empty_content_for_non_assistant(self):
        line = json.dumps({"type": "system", "subtype": "init"})
        msg = parse(line)
        assert msg.content == []

    def test_error_result(self):
        line = json.dumps({
            "type": "result",
            "subtype": "error",
            "result": "Something failed",
            "is_error": True,
        })
        msg = parse(line)
        assert msg.is_error is True
        assert msg.result == "Something failed"

    def test_unknown_fields_ignored(self):
        line = json.dumps({"type": "assistant", "unknown_field": "value", "message": None})
        msg = parse(line)
        assert msg.type == "assistant"
