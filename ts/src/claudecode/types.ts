/**
 * Types for Claude Code CLI stream-json output.
 * Each line from `claude -p --output-format stream-json` maps to StreamMessage.
 */

/** Top-level envelope for all Claude stream-json lines. */
export interface StreamMessage {
  type: string;
  subtype?: string;

  /**
   * Content blocks lifted from the assistant message wrapper.
   * Populated by parse() for assistant-type messages.
   */
  content: ContentBlock[];

  /** Nested "message" object in assistant-type lines. */
  message?: AssistantMessage;

  /** Result fields — present when type === "result". */
  result?: string;
  is_error?: boolean;
  total_cost_usd?: number;
  duration_ms?: number;
  duration_api_ms?: number;
  num_turns?: number;
  session_id?: string;
  model?: string;
  usage?: ResultUsage;

  /** System/init fields. */
  tools?: unknown[];

  /** Rate limit info from rate_limit_event messages. */
  rate_limit_info?: RateLimitInfo;

  /** Stream event fields — present when type === "stream_event". */
  event?: StreamEventInner;
  parent_tool_use_id?: string;
}

/** Nested "message" object inside assistant-type stream lines. */
export interface AssistantMessage {
  model?: string;
  id?: string;
  content?: ContentBlock[];
  stop_reason?: string;
  usage?: StreamUsage;
}

/** One block inside an assistant message. */
export interface ContentBlock {
  type: string;
  text?: string;
  thinking?: string;
  name?: string;
  input?: unknown;
  content?: unknown;
}

/** Parsed inner event from a stream_event line. */
export interface StreamEventInner {
  type: string;
  message?: MessageStartData;
  index?: number;
  content_block?: ContentBlockInfo;
  delta?: Delta;
  usage?: StreamUsage;
}

/** Message metadata from a message_start event. */
export interface MessageStartData {
  model: string;
  id: string;
  usage?: StreamUsage;
}

/** Content block info in content_block_start events. */
export interface ContentBlockInfo {
  type: string;
  name?: string;
  id?: string;
}

/** Incremental data in delta events. */
export interface Delta {
  type?: string;
  text?: string;
  thinking?: string;
  partial_json?: string;
  stop_reason?: string;
  stop_sequence?: string;
}

/** Rate limit details from rate_limit_event messages. */
export interface RateLimitInfo {
  status: string;
  rateLimitType?: string;
  utilization?: number;
  resetsAt?: number;
  isUsingOverage?: boolean;
}

/** Token counts from the final result message (includes cache fields). */
export interface ResultUsage {
  input_tokens?: number;
  output_tokens?: number;
  cache_creation_input_tokens?: number;
  cache_read_input_tokens?: number;
}

/** Token counts from streaming events. */
export interface StreamUsage {
  input_tokens?: number;
  output_tokens?: number;
}
