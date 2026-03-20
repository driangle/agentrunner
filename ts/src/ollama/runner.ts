import { createInterface } from "node:readline";
import { Readable } from "node:stream";
import type { Runner, Result, Session, Usage } from "../types.js";
import type { OllamaMessage } from "./accessors.js";
import {
  NotFoundError,
  ParseError,
  NoResultError,
  HttpError,
  NotSupportedError,
} from "../errors.js";
import { combinedSignal, abortError } from "../signal.js";
import type {
  OllamaRunnerConfig,
  OllamaRunOptions,
  FetchFn,
} from "./options.js";
import type { ChatResponse } from "./types.js";
import { buildRequestBody, mapFetchError, logRequest } from "./request.js";

const DEFAULT_BASE_URL = "http://localhost:11434";

/** Create an Ollama runner. */
export function createOllamaRunner(
  config: OllamaRunnerConfig = {},
): Runner<OllamaRunOptions, OllamaMessage> {
  const baseURL = config.baseURL ?? DEFAULT_BASE_URL;
  const fetchFn: FetchFn = config.fetch ?? fetch;

  return {
    start: (prompt, options) =>
      start(config, fetchFn, baseURL, prompt, options),
    run: (prompt, options) => run(config, fetchFn, baseURL, prompt, options),
    runStream: (prompt, options) =>
      runStream(config, fetchFn, baseURL, prompt, options),
  };
}

function start(
  config: OllamaRunnerConfig,
  fetchFn: FetchFn,
  baseURL: string,
  prompt: string,
  options: OllamaRunOptions = {},
): Session<OllamaMessage> {
  const { signal, clearTimeout: clearTO } = combinedSignal(options);
  const abortController = new AbortController();
  signal.addEventListener("abort", () => abortController.abort(signal.reason), {
    once: true,
  });

  let resolveResult: (value: Result) => void;
  let rejectResult: (reason: unknown) => void;
  const resultPromise = new Promise<Result>((resolve, reject) => {
    resolveResult = resolve;
    rejectResult = reject;
  });
  resultPromise.catch(() => {}); // Prevent unhandled rejection on abandon.

  async function* messageGenerator(): AsyncGenerator<OllamaMessage> {
    try {
      const body = buildRequestBody(prompt, options);
      logRequest(config, baseURL);
      let resp: Response;
      try {
        resp = await fetchFn(`${baseURL}/api/chat`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
          signal: abortController.signal,
        });
      } catch (err) {
        if (signal.aborted) {
          throw abortError(signal);
        }
        throw mapFetchError(err);
      }

      if (resp.status === 404)
        throw new NotFoundError("model not found (HTTP 404)");
      if (!resp.ok) throw new HttpError(resp.status, `HTTP ${resp.status}`);
      if (!resp.body) throw new NoResultError();

      const nodeStream = Readable.fromWeb(
        resp.body as import("node:stream/web").ReadableStream,
      );
      const rl = createInterface({ input: nodeStream });
      const textParts: string[] = [];
      let finalResp: ChatResponse | undefined;

      for await (const line of rl) {
        if (abortController.signal.aborted) break;
        if (!line) continue;

        let chunk: ChatResponse;
        try {
          chunk = JSON.parse(line) as ChatResponse;
        } catch {
          throw new ParseError(`invalid JSON: ${line}`);
        }

        if (chunk.message.content) textParts.push(chunk.message.content);
        if (chunk.done) finalResp = chunk;

        const msg: OllamaMessage = {
          type: chunk.done ? "result" : "assistant",
          raw: line,
          data: chunk,
        };

        if (options.onMessage) options.onMessage(msg);
        yield msg;
      }

      clearTO();

      if (abortController.signal.aborted) {
        const err = abortError(signal);
        rejectResult!(err);
        return;
      }

      if (!finalResp) {
        rejectResult!(new NoResultError());
        return;
      }

      const usage: Usage = {
        inputTokens: finalResp.prompt_eval_count ?? 0,
        outputTokens: finalResp.eval_count ?? 0,
      };

      resolveResult!({
        text: textParts.join(""),
        isError: false,
        exitCode: 0,
        usage,
        costUSD: 0,
        durationMs: finalResp.total_duration
          ? Math.floor(finalResp.total_duration / 1e6)
          : 0,
        sessionId: "",
      });
    } catch (err) {
      clearTO();
      rejectResult!(mapFetchError(err));
    }
  }

  const messages = messageGenerator();

  return {
    messages,
    result: resultPromise,
    abort: () => {
      abortController.abort("cancelled");
    },
    send: () => {
      throw new NotSupportedError("send is not yet supported");
    },
  };
}

async function run(
  config: OllamaRunnerConfig,
  fetchFn: FetchFn,
  baseURL: string,
  prompt: string,
  options: OllamaRunOptions = {},
): Promise<Result> {
  const session = start(config, fetchFn, baseURL, prompt, options);

  for await (const _msg of session.messages) {
    // consumed
  }

  return session.result;
}

async function* runStream(
  config: OllamaRunnerConfig,
  fetchFn: FetchFn,
  baseURL: string,
  prompt: string,
  options: OllamaRunOptions = {},
): AsyncGenerator<OllamaMessage> {
  const session = start(config, fetchFn, baseURL, prompt, options);
  yield* session.messages;

  // Propagates timeout/cancel/parse errors to the stream consumer.
  await session.result;
}
