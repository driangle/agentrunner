import type { StreamMessage } from "./types.js";

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
 * Parse a single JSONL line from Gemini CLI output into a typed StreamMessage.
 * Unknown fields are preserved for forward compatibility.
 */
export function parse(line: string): StreamMessage {
  const raw: unknown = JSON.parse(line);

  if (!isStreamMessageShape(raw)) {
    throw new SyntaxError("not a valid stream message: missing type field");
  }

  return raw as unknown as StreamMessage;
}
