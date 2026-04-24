import type { MessageType } from "../types.js";
import type { StreamMessage, ItemStreamMessage } from "./types.js";

/** Map Codex stream message types to common MessageType. */
export function mapMessageType(msg: StreamMessage): MessageType {
  switch (msg.type) {
    case "thread.started":
    case "turn.started":
      return "system";
    case "item.started": {
      const item = (msg as ItemStreamMessage).item;
      if (item?.type === "command_execution") return "tool_use";
      return "assistant";
    }
    case "item.completed": {
      const item = (msg as ItemStreamMessage).item;
      if (item?.type === "agent_message") return "assistant";
      if (item?.type === "command_execution") return "tool_result";
      return "assistant";
    }
    case "turn.completed":
      return "result";
    case "error":
    case "turn.failed":
      return "error";
    default:
      return msg.type;
  }
}
