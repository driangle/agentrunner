/**
 * Types for Gemini CLI JSONL output.
 * Each line from `gemini --output-format stream-json` maps to StreamMessage.
 *
 * StreamMessage is a discriminated union on the `type` field.
 */

/** Token counts and timing from a result event. */
export interface StreamStats {
  total_tokens?: number;
  input_tokens?: number;
  output_tokens?: number;
  cached?: number;
  input?: number;
  duration_ms?: number;
  tool_calls?: number;
  models?: Record<string, ModelStreamStats>;
}

/** Per-model token counts. */
export interface ModelStreamStats {
  total_tokens?: number;
  input_tokens?: number;
  output_tokens?: number;
  cached?: number;
  input?: number;
}

/** Error details from result or error events. */
export interface EventError {
  type?: string;
  message: string;
}

/** Base fields shared by all Gemini stream message types. */
interface BaseStreamMessage {
  type: string;
  timestamp?: string;
}

/** Session initialization event. */
export interface InitStreamMessage extends BaseStreamMessage {
  type: "init";
  session_id?: string;
  model?: string;
}

/** Text message event (user or assistant). */
export interface MessageStreamMessage extends BaseStreamMessage {
  type: "message";
  role?: string;
  content?: string;
  delta?: boolean;
}

/** Tool invocation event. */
export interface ToolUseStreamMessage extends BaseStreamMessage {
  type: "tool_use";
  tool_name?: string;
  tool_id?: string;
  parameters?: unknown;
}

/** Tool execution result event. */
export interface ToolResultStreamMessage extends BaseStreamMessage {
  type: "tool_result";
  tool_id?: string;
  status?: string;
  output?: string;
}

/** Standalone error event. */
export interface ErrorStreamMessage extends BaseStreamMessage {
  type: "error";
  severity?: string;
  message?: string;
}

/** Final result event with stats. */
export interface ResultStreamMessage extends BaseStreamMessage {
  type: "result";
  status?: string;
  error?: EventError;
  stats?: StreamStats;
}

/** Catch-all for unknown/future message types (forward compatibility). */
export interface UnknownStreamMessage extends BaseStreamMessage {
  type: string;
}

/** Discriminated union of all Gemini stream message types. */
export type StreamMessage =
  | InitStreamMessage
  | MessageStreamMessage
  | ToolUseStreamMessage
  | ToolResultStreamMessage
  | ErrorStreamMessage
  | ResultStreamMessage
  | UnknownStreamMessage;
