export { createClaudeRunner } from "./runner.js";
export { parse } from "./parser.js";
export { buildArgs } from "./args.js";
export {
  messageText,
  messageThinking,
  messageToolName,
  messageToolInput,
  messageToolOutput,
  messageIsError,
  messageTextDelta,
  messageThinkingDelta,
  messageChannelReplyContent,
  messageChannelReplyDestination,
} from "./accessors.js";
export type { ClaudeMessage } from "./accessors.js";

export type { OnMessageFn, Logger } from "../types.js";

export type {
  ClaudeRunnerConfig,
  ClaudeRunOptions,
  SpawnFn,
} from "./options.js";

/** @experimental Channel types and utilities — subject to change. */
export type { ChannelMessage } from "./channel.js";
/** @experimental Channel constants and helpers — subject to change. */
export {
  CHANNEL_REPLY_TOOL_NAME,
  isChannelReply,
  channelReplyContent,
  channelReplyDestination,
} from "./channel.js";

export type {
  StreamMessage,
  SystemStreamMessage,
  AssistantStreamMessage,
  UserStreamMessage,
  ResultStreamMessage,
  StreamEventStreamMessage,
  RateLimitStreamMessage,
  UnknownStreamMessage,
  AssistantMessage,
  ContentBlock,
  StreamEventInner,
  MessageStartData,
  ContentBlockInfo,
  Delta,
  RateLimitInfo,
  ResultUsage,
  StreamUsage,
} from "./types.js";
