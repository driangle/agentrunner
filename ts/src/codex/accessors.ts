import type { Message } from "../types.js";
import type {
  StreamMessage,
  ItemStreamMessage,
  ErrorStreamMessage,
  TurnFailedStreamMessage,
} from "./types.js";

/** Typed alias for a Message carrying Codex stream data. */
export type CodexMessage = Message<StreamMessage>;

/** Return the text content from an item.completed agent_message, or undefined. */
export function messageText(msg: CodexMessage): string | undefined {
  const data = msg.data;
  if (
    (data.type === "item.completed" || data.type === "item.started") &&
    (data as ItemStreamMessage).item?.type === "agent_message"
  ) {
    return (data as ItemStreamMessage).item?.text;
  }
  return undefined;
}

/** Return the command being executed for command_execution items, or undefined. */
export function messageToolName(msg: CodexMessage): string | undefined {
  const data = msg.data;
  if (data.type === "item.started" || data.type === "item.completed") {
    const item = (data as ItemStreamMessage).item;
    if (item?.type === "command_execution") {
      return item.command;
    }
  }
  return undefined;
}

/** Return the aggregated output for completed command_execution items, or undefined. */
export function messageToolOutput(msg: CodexMessage): string | undefined {
  const data = msg.data;
  if (data.type === "item.completed") {
    const item = (data as ItemStreamMessage).item;
    if (item?.type === "command_execution" && item.status === "completed") {
      return item.aggregated_output;
    }
  }
  return undefined;
}

/** Whether this message represents an error. */
export function messageIsError(msg: CodexMessage): boolean {
  return msg.data.type === "error" || msg.data.type === "turn.failed";
}

/** Return the error message, or undefined. */
export function messageErrorText(msg: CodexMessage): string | undefined {
  if (msg.data.type === "error") {
    return (msg.data as ErrorStreamMessage).message;
  }
  if (msg.data.type === "turn.failed") {
    return (msg.data as TurnFailedStreamMessage).error?.message;
  }
  return undefined;
}
