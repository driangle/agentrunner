"""Types for Claude Code CLI stream-json output."""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any


@dataclass
class ContentBlock:
    """One block inside an assistant message."""

    type: str = ""
    text: str | None = None
    thinking: str | None = None
    name: str | None = None
    input: Any = None
    content: Any = None


@dataclass
class StreamUsage:
    """Token counts from streaming events."""

    input_tokens: int = 0
    output_tokens: int = 0


@dataclass
class ResultUsage:
    """Token counts from the final result message (includes cache fields)."""

    input_tokens: int = 0
    output_tokens: int = 0
    cache_creation_input_tokens: int = 0
    cache_read_input_tokens: int = 0


@dataclass
class Delta:
    """Incremental data in delta events."""

    type: str | None = None
    text: str | None = None
    thinking: str | None = None
    partial_json: str | None = None
    stop_reason: str | None = None
    stop_sequence: str | None = None


@dataclass
class ContentBlockInfo:
    """Content block info in content_block_start events."""

    type: str = ""
    name: str | None = None
    id: str | None = None


@dataclass
class MessageStartData:
    """Message metadata from a message_start event."""

    model: str = ""
    id: str = ""
    usage: StreamUsage | None = None


@dataclass
class StreamEventInner:
    """Parsed inner event from a stream_event line."""

    type: str = ""
    message: MessageStartData | None = None
    index: int | None = None
    content_block: ContentBlockInfo | None = None
    delta: Delta | None = None
    usage: StreamUsage | None = None


@dataclass
class RateLimitInfo:
    """Rate limit details from rate_limit_event messages."""

    status: str = ""
    rate_limit_type: str | None = None
    utilization: float | None = None
    resets_at: float | None = None
    is_using_overage: bool | None = None


@dataclass
class AssistantMessage:
    """Nested 'message' object inside assistant-type stream lines."""

    model: str | None = None
    id: str | None = None
    content: list[ContentBlock] = field(default_factory=list)
    stop_reason: str | None = None
    usage: StreamUsage | None = None


@dataclass
class StreamMessage:
    """Top-level envelope for all Claude stream-json lines."""

    type: str = ""
    subtype: str | None = None
    content: list[ContentBlock] = field(default_factory=list)
    message: AssistantMessage | None = None

    # Result fields.
    result: str | None = None
    is_error: bool | None = None
    total_cost_usd: float | None = None
    duration_ms: float | None = None
    duration_api_ms: float | None = None
    num_turns: int | None = None
    session_id: str | None = None
    model: str | None = None
    usage: ResultUsage | None = None

    # System/init fields.
    tools: list[Any] | None = None

    # Rate limit info.
    rate_limit_info: RateLimitInfo | None = None

    # Stream event fields.
    event: StreamEventInner | None = None
    parent_tool_use_id: str | None = None
