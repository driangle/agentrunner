"""Map Claude stream-json types to common types."""

from __future__ import annotations

import json

from ..types import Result, Usage
from .types import StreamMessage

_TYPE_MAP = {
    "system": "system",
    "assistant": "assistant",
    "result": "result",
}


def map_message_type(raw_type: str, raw_line: str) -> str:
    """Map Claude stream-json type to common MessageType.

    Uses the raw JSON line to distinguish tool_use and tool_result subtypes
    from generic assistant/user messages, matching the INTERFACE.md taxonomy.
    """
    if raw_type in _TYPE_MAP:
        return _TYPE_MAP[raw_type]

    if raw_type == "user":
        try:
            d = json.loads(raw_line)
            for block in d.get("content", []):
                if isinstance(block, dict) and block.get("type") == "tool_result":
                    return "tool_result"
        except (json.JSONDecodeError, TypeError):
            pass
        return "user"

    if raw_type == "stream_event":
        return "assistant"

    return raw_type


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
