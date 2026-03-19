"""Map Claude stream-json types to common types."""

from __future__ import annotations

from ..types import Result, Usage
from .types import StreamMessage

_TYPE_MAP = {
    "system": "system",
    "assistant": "assistant",
    "user": "user",
    "result": "result",
    "stream_event": "assistant",
}


def map_message_type(type_: str) -> str:
    """Map Claude stream-json type to common MessageType."""
    return _TYPE_MAP.get(type_, type_)


def map_result(msg: StreamMessage, fallback_session_id: str) -> Result:
    """Map a StreamMessage result to a common Result."""
    usage = Usage(
        input_tokens=msg.usage.input_tokens if msg.usage else 0,
        output_tokens=msg.usage.output_tokens if msg.usage else 0,
        cache_creation_input_tokens=msg.usage.cache_creation_input_tokens if msg.usage else 0,
        cache_read_input_tokens=msg.usage.cache_read_input_tokens if msg.usage else 0,
    )

    return Result(
        text=msg.result or "",
        is_error=msg.is_error or False,
        exit_code=0,
        usage=usage,
        cost_usd=msg.total_cost_usd or 0,
        duration_ms=msg.duration_ms or 0,
        session_id=msg.session_id or fallback_session_id,
    )
