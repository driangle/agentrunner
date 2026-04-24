import type { Result } from "../types.js";
import type { ResultStreamMessage } from "./types.js";

/** Build a common Result from tracked Gemini stream state. */
export function buildResult(
  text: string,
  isError: boolean,
  sessionId: string,
  resultMsg?: ResultStreamMessage,
): Result {
  return {
    text,
    isError,
    exitCode: 0,
    usage: {
      inputTokens: resultMsg?.stats?.input_tokens ?? 0,
      outputTokens: resultMsg?.stats?.output_tokens ?? 0,
      cacheReadInputTokens: resultMsg?.stats?.cached ?? 0,
    },
    costUSD: 0,
    durationMs: resultMsg?.stats?.duration_ms ?? 0,
    sessionId,
  };
}

/** Resolve the final text from a result message. */
export function resolveResultText(
  resultMsg: ResultStreamMessage,
  lastAssistantText: string,
): string {
  if (resultMsg.status === "error" && resultMsg.error && !lastAssistantText) {
    return resultMsg.error.message;
  }
  return lastAssistantText;
}
