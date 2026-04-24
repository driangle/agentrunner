import type { Message } from "../types.js";
import type {
  StreamMessage,
  MessageStreamMessage,
  ToolUseStreamMessage,
  ToolResultStreamMessage,
  ErrorStreamMessage,
  ResultStreamMessage,
} from "./types.js";

/** Typed alias for a Message carrying Gemini stream data. */
export type GeminiMessage = Message<StreamMessage>;

/** Return the text content from an assistant message event, or undefined. */
export function messageText(msg: GeminiMessage): string | undefined {
  const data = msg.data;
  if (
    data.type === "message" &&
    (data as MessageStreamMessage).role === "assistant"
  ) {
    return (data as MessageStreamMessage).content;
  }
  return undefined;
}

/** Return the tool name from a tool_use event, or undefined. */
export function messageToolName(msg: GeminiMessage): string | undefined {
  if (msg.data.type === "tool_use") {
    return (msg.data as ToolUseStreamMessage).tool_name;
  }
  return undefined;
}

/** Return the tool parameters from a tool_use event, or undefined. */
export function messageToolInput(msg: GeminiMessage): unknown | undefined {
  if (msg.data.type === "tool_use") {
    return (msg.data as ToolUseStreamMessage).parameters;
  }
  return undefined;
}

/** Return the tool output from a tool_result event, or undefined. */
export function messageToolOutput(msg: GeminiMessage): string | undefined {
  if (msg.data.type === "tool_result") {
    return (msg.data as ToolResultStreamMessage).output;
  }
  return undefined;
}

/** Whether this message represents an error. */
export function messageIsError(msg: GeminiMessage): boolean {
  if (msg.data.type === "error") return true;
  if (
    msg.data.type === "result" &&
    (msg.data as ResultStreamMessage).status === "error"
  ) {
    return true;
  }
  return false;
}

/** Return the error message, or undefined. */
export function messageErrorText(msg: GeminiMessage): string | undefined {
  if (msg.data.type === "error") {
    return (msg.data as ErrorStreamMessage).message;
  }
  if (msg.data.type === "result") {
    return (msg.data as ResultStreamMessage).error?.message;
  }
  return undefined;
}
