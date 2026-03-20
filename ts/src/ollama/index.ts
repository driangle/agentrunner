export { createOllamaRunner } from "./runner.js";
export { messageText, messageThinking, messageIsResult } from "./accessors.js";
export type { OllamaMessage } from "./accessors.js";

export type { OnMessageFn, Logger } from "../types.js";

export type {
  OllamaRunnerConfig,
  OllamaRunOptions,
  FetchFn,
} from "./options.js";

export type {
  ChatRequest,
  ChatMessage,
  ChatResponse,
  ModelOptions,
} from "./types.js";
