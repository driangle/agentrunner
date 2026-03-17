import type { StreamMessage, StreamEventInner } from "./types.js";

/**
 * Type guard that validates a parsed JSON value has the minimum shape
 * of a StreamMessage (an object with a string `type` field).
 */
function isStreamMessageShape(
  value: unknown,
): value is Record<string, unknown> & { type: string } {
  return (
    typeof value === "object" &&
    value !== null &&
    "type" in value &&
    typeof (value as Record<string, unknown>).type === "string"
  );
}

/**
 * Parse a single JSON line from Claude Code's stream-json output
 * into a typed StreamMessage. Unknown fields are silently ignored for
 * forward compatibility.
 *
 * For assistant-type lines, content blocks are lifted from the nested
 * "message" wrapper into StreamMessage.content for convenient access.
 * For stream_event lines, the inner event is parsed into StreamMessage.event.
 */
export function parse(line: string): StreamMessage {
  const raw: unknown = JSON.parse(line);

  if (!isStreamMessageShape(raw)) {
    throw new SyntaxError("not a valid stream message: missing type field");
  }

  const msg: StreamMessage = {
    type: raw.type,
    content: [],
    ...(raw as object),
  };

  switch (msg.type) {
    case "assistant":
      if (msg.message?.content) {
        msg.content = msg.message.content;
      }
      break;

    case "stream_event":
      if (raw.event != null && typeof raw.event === "object") {
        // Parse inner event — gracefully skip if it doesn't have a type.
        const inner = raw.event as Record<string, unknown>;
        if (typeof inner.type === "string") {
          msg.event = inner as unknown as StreamEventInner;
        }
      }
      break;
  }

  return msg;
}
