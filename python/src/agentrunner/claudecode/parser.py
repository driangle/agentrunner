"""Parse Claude Code CLI stream-json output lines."""

from __future__ import annotations

import json

from .types import (
    AssistantMessage,
    ContentBlock,
    ContentBlockInfo,
    Delta,
    MessageStartData,
    ResultUsage,
    StreamEventInner,
    StreamMessage,
    StreamUsage,
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


def _parse_delta(raw: dict) -> Delta:
    return Delta(
        type=raw.get("type"),
        text=raw.get("text"),
        thinking=raw.get("thinking"),
        partial_json=raw.get("partial_json"),
        stop_reason=raw.get("stop_reason"),
        stop_sequence=raw.get("stop_sequence"),
    )


def _parse_content_block_info(raw: dict) -> ContentBlockInfo:
    return ContentBlockInfo(
        type=raw.get("type", ""),
        name=raw.get("name"),
        id=raw.get("id"),
    )


def _parse_stream_usage(raw: dict) -> StreamUsage:
    return StreamUsage(
        input_tokens=raw.get("input_tokens", 0),
        output_tokens=raw.get("output_tokens", 0),
    )


def _parse_message_start_data(raw: dict) -> MessageStartData:
    usage = None
    if "usage" in raw and isinstance(raw["usage"], dict):
        usage = _parse_stream_usage(raw["usage"])
    return MessageStartData(
        model=raw.get("model", ""),
        id=raw.get("id", ""),
        usage=usage,
    )


def _parse_stream_event(raw: dict) -> StreamEventInner | None:
    if not isinstance(raw, dict) or not isinstance(raw.get("type"), str):
        return None

    event = StreamEventInner(type=raw["type"])
    event.index = raw.get("index")

    if "delta" in raw and isinstance(raw["delta"], dict):
        event.delta = _parse_delta(raw["delta"])
    if "content_block" in raw and isinstance(raw["content_block"], dict):
        event.content_block = _parse_content_block_info(raw["content_block"])
    if "message" in raw and isinstance(raw["message"], dict):
        event.message = _parse_message_start_data(raw["message"])
    if "usage" in raw and isinstance(raw["usage"], dict):
        event.usage = _parse_stream_usage(raw["usage"])

    return event


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
