import type { MessageType } from "../types.js";
import type { StreamMessage } from "./types.js";

/** Map Gemini stream message types to common MessageType. */
export function mapMessageType(msg: StreamMessage): MessageType {
  switch (msg.type) {
    case "init":
      return "system";
    case "message":
      if ("role" in msg && msg.role === "user") return "user";
      return "assistant";
    case "tool_use":
      return "tool_use";
    case "tool_result":
      return "tool_result";
    case "error":
      return "error";
    case "result":
      return "result";
    default:
      return msg.type;
  }
}
