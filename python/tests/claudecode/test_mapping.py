"""Tests for message type mapping."""

import json

from agentrunner.claudecode.mapping import map_message_type


class TestMapMessageType:
    def test_system(self):
        assert map_message_type("system", "{}") == "system"

    def test_assistant(self):
        assert map_message_type("assistant", "{}") == "assistant"

    def test_result(self):
        assert map_message_type("result", "{}") == "result"

    def test_stream_event_maps_to_assistant(self):
        assert map_message_type("stream_event", "{}") == "assistant"

    def test_user_with_tool_result_maps_to_tool_result(self):
        raw = json.dumps(
            {"type": "user", "content": [{"type": "tool_result", "content": "output"}]}
        )
        assert map_message_type("user", raw) == "tool_result"

    def test_plain_user_stays_user(self):
        raw = json.dumps({"type": "user", "content": [{"type": "text", "text": "hello"}]})
        assert map_message_type("user", raw) == "user"

    def test_unknown_type_passed_through(self):
        assert map_message_type("rate_limit_event", "{}") == "rate_limit_event"
