"""Parse Claude Code CLI stream-json output lines."""

from __future__ import annotations

import json

from .types import (
    AssistantMessage,
    ContentBlock,
    ResultUsage,
    StreamEventInner,
    StreamMessage,
)


def _parse_content_block(raw: dict) -> ContentBlock:
    return ContentBlock(
        type=raw.get("type", ""),
        text=raw.get("text"),
        thinking=raw.get("thinking"),
        name=raw.get("name"),
        input=raw.get("input"),
        content=raw.get("content"),
    )


def _parse_assistant_message(raw: dict) -> AssistantMessage:
    content = [_parse_content_block(b) for b in raw.get("content", [])]
    return AssistantMessage(
        model=raw.get("model"),
        id=raw.get("id"),
        content=content,
        stop_reason=raw.get("stop_reason"),
    )


def _parse_result_usage(raw: dict) -> ResultUsage:
    return ResultUsage(
        input_tokens=raw.get("input_tokens", 0),
        output_tokens=raw.get("output_tokens", 0),
        cache_creation_input_tokens=raw.get("cache_creation_input_tokens", 0),
        cache_read_input_tokens=raw.get("cache_read_input_tokens", 0),
    )


def _parse_stream_event(raw: dict) -> StreamEventInner | None:
    if not isinstance(raw, dict) or not isinstance(raw.get("type"), str):
        return None
    return StreamEventInner(type=raw["type"])


def parse(line: str) -> StreamMessage:
    """Parse a single JSON line into a StreamMessage.

    For assistant-type lines, content blocks are lifted from the nested
    'message' wrapper into StreamMessage.content for convenient access.
    """
    raw = json.loads(line)

    if not isinstance(raw, dict) or not isinstance(raw.get("type"), str):
        raise ValueError("not a valid stream message: missing type field")

    msg = StreamMessage(
        type=raw["type"],
        subtype=raw.get("subtype"),
        session_id=raw.get("session_id"),
        model=raw.get("model"),
    )

    # Result fields.
    if "result" in raw:
        msg.result = raw["result"]
    if "is_error" in raw:
        msg.is_error = raw["is_error"]
    if "total_cost_usd" in raw:
        msg.total_cost_usd = raw["total_cost_usd"]
    if "duration_ms" in raw:
        msg.duration_ms = raw["duration_ms"]
    if "duration_api_ms" in raw:
        msg.duration_api_ms = raw["duration_api_ms"]
    if "num_turns" in raw:
        msg.num_turns = raw["num_turns"]
    if "usage" in raw and isinstance(raw["usage"], dict):
        msg.usage = _parse_result_usage(raw["usage"])

    # Assistant message.
    if "message" in raw and isinstance(raw["message"], dict):
        msg.message = _parse_assistant_message(raw["message"])

    # Lift content from assistant message.
    if msg.type == "assistant" and msg.message and msg.message.content:
        msg.content = msg.message.content

    # Stream event.
    if msg.type == "stream_event" and "event" in raw:
        msg.event = _parse_stream_event(raw["event"])

    return msg
