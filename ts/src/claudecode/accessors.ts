import type { Message } from "../types.js";
import type {
  StreamMessage,
  AssistantStreamMessage,
  ResultStreamMessage,
  StreamEventStreamMessage,
} from "./types.js";

/** Typed alias for a Message carrying Claude stream data. */
export type ClaudeMessage = Message<StreamMessage>;

/** Return the text content from an assistant or result message, or undefined. */
export function messageText(msg: ClaudeMessage): string | undefined {
  const data = msg.data;
  if (data.type === "assistant") {
    const asst = data as AssistantStreamMessage;
    const block = asst.content.find((b) => b.type === "text");
    return block?.text;
  }
  if (data.type === "result") {
    return (data as ResultStreamMessage).result;
  }
  return undefined;
}

/** Return the thinking content from an assistant message, or undefined. */
export function messageThinking(msg: ClaudeMessage): string | undefined {
  if (msg.data.type !== "assistant") return undefined;
  const asst = msg.data as AssistantStreamMessage;
  const block = asst.content.find((b) => b.type === "thinking");
  return block?.thinking;
}

/** Return the tool name from an assistant tool_use content block, or undefined. */
export function messageToolName(msg: ClaudeMessage): string | undefined {
  if (msg.data.type !== "assistant") return undefined;
  const asst = msg.data as AssistantStreamMessage;
  const block = asst.content.find((b) => b.type === "tool_use");
  return block?.name;
}

/** Return the tool input from an assistant tool_use content block, or undefined. */
export function messageToolInput(msg: ClaudeMessage): unknown | undefined {
  if (msg.data.type !== "assistant") return undefined;
  const asst = msg.data as AssistantStreamMessage;
  const block = asst.content.find((b) => b.type === "tool_use");
  return block?.input;
}

/** Return the tool output from a user (tool_result) message, or undefined. */
export function messageToolOutput(msg: ClaudeMessage): unknown | undefined {
  if (msg.data.type !== "user") return undefined;
  // User messages contain tool results in the nested message.content.
  const data = msg.data as StreamMessage & {
    message?: { content?: Array<{ content?: unknown }> };
  };
  const block = data.message?.content?.[0];
  return block?.content;
}

/** Whether this message represents an error. */
export function messageIsError(msg: ClaudeMessage): boolean {
  if (msg.data.type === "result") {
    return (msg.data as ResultStreamMessage).is_error ?? false;
  }
  return false;
}

/** Return the text delta from a stream_event, or undefined. */
export function messageTextDelta(msg: ClaudeMessage): string | undefined {
  if (msg.data.type !== "stream_event") return undefined;
  const se = msg.data as StreamEventStreamMessage;
  const delta = se.event?.delta;
  if (delta?.type === "text_delta") return delta.text;
  return undefined;
}

/** Return the thinking delta from a stream_event, or undefined. */
export function messageThinkingDelta(msg: ClaudeMessage): string | undefined {
  if (msg.data.type !== "stream_event") return undefined;
  const se = msg.data as StreamEventStreamMessage;
  const delta = se.event?.delta;
  if (delta?.type === "thinking_delta") return delta.thinking;
  return undefined;
}
