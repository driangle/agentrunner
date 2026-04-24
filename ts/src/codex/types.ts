/**
 * Types for Codex CLI JSONL output.
 * Each line from `codex exec --json` maps to StreamMessage.
 *
 * StreamMessage is a discriminated union on the `type` field.
 */

/** A Codex event item (agent_message or command_execution). */
export interface Item {
  id: string;
  type: string;
  text?: string;
  command?: string;
  aggregated_output?: string;
  exit_code?: number;
  status?: string;
}

/** Token counts from a turn.completed event. */
export interface TurnUsage {
  input_tokens?: number;
  cached_input_tokens?: number;
  output_tokens?: number;
}

/** Error details from a turn.failed event. */
export interface TurnError {
  message: string;
}

/** Base fields shared by all Codex stream message types. */
interface BaseStreamMessage {
  type: string;
  thread_id?: string;
}

/** Thread lifecycle events (thread.started, turn.started). */
export interface SystemStreamMessage extends BaseStreamMessage {
  type: "thread.started" | "turn.started";
}

/** Item events (item.started, item.completed). */
export interface ItemStreamMessage extends BaseStreamMessage {
  type: "item.started" | "item.completed";
  item?: Item;
}

/** Turn completed with usage stats. */
export interface TurnCompletedStreamMessage extends BaseStreamMessage {
  type: "turn.completed";
  usage?: TurnUsage;
}

/** Error event. */
export interface ErrorStreamMessage extends BaseStreamMessage {
  type: "error";
  message?: string;
}

/** Turn failed event. */
export interface TurnFailedStreamMessage extends BaseStreamMessage {
  type: "turn.failed";
  error?: TurnError;
}

/** Catch-all for unknown/future message types (forward compatibility). */
export interface UnknownStreamMessage extends BaseStreamMessage {
  type: string;
}

/** Discriminated union of all Codex stream message types. */
export type StreamMessage =
  | SystemStreamMessage
  | ItemStreamMessage
  | TurnCompletedStreamMessage
  | ErrorStreamMessage
  | TurnFailedStreamMessage
  | UnknownStreamMessage;
