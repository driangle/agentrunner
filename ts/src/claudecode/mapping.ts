import type { Result, MessageType, Usage } from "../types.js";
import type { StreamMessage } from "./types.js";

/** Map Claude stream-json type to common MessageType. */
export function mapMessageType(type: string): MessageType {
  switch (type) {
    case "system":
      return "system";
    case "assistant":
      return "assistant";
    case "user":
      return "user";
    case "result":
      return "result";
    case "stream_event":
      // Stream events carry assistant content deltas.
      return "assistant";
    default:
      return type;
  }
}

/** Map a StreamMessage result to a common Result. */
export function mapResult(
  msg: StreamMessage,
  fallbackSessionId: string,
): Result {
  const usage: Usage = {
    inputTokens: msg.usage?.input_tokens ?? 0,
    outputTokens: msg.usage?.output_tokens ?? 0,
    cacheCreationInputTokens: msg.usage?.cache_creation_input_tokens ?? 0,
    cacheReadInputTokens: msg.usage?.cache_read_input_tokens ?? 0,
  };

  return {
    text: msg.result ?? "",
    isError: msg.is_error ?? false,
    exitCode: 0,
    usage,
    costUSD: msg.total_cost_usd ?? 0,
    durationMs: msg.duration_ms ?? 0,
    sessionId: msg.session_id || fallbackSessionId,
  };
}
