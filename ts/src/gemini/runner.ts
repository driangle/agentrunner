import { createInterface } from "node:readline";
import { once } from "node:events";
import type { Runner, Result, Session } from "../types.js";
import type { GeminiMessage } from "./accessors.js";
import {
  NotFoundError,
  NonZeroExitError,
  NoResultError,
  NotSupportedError,
} from "../errors.js";
import type {
  GeminiRunnerConfig,
  GeminiRunOptions,
  SpawnFn,
} from "./options.js";
import type {
  StreamMessage,
  MessageStreamMessage,
  ResultStreamMessage,
  ErrorStreamMessage,
} from "./types.js";
import { parse } from "./parser.js";
import { buildArgs } from "./args.js";
import { combinedSignal, abortError } from "../signal.js";
import { logCmd, resolveSpawn, collectErrorDetail } from "./process.js";
import { mapMessageType } from "./mapping.js";
import { buildResult, resolveResultText } from "./result.js";

/** Create a Gemini CLI runner. */
export function createGeminiRunner(
  config: GeminiRunnerConfig = {},
): Runner<GeminiRunOptions, GeminiMessage> {
  const { spawn: spawnFn, binary } = resolveSpawn(config);
  const s = (prompt: string, opts?: GeminiRunOptions) =>
    start(config, spawnFn, binary, prompt, opts);

  return {
    start: s,
    async run(prompt, options) {
      const session = s(prompt, options);
      for await (const _msg of session.messages) {
      } // drain
      return session.result;
    },
    async *runStream(prompt, options) {
      const session = s(prompt, options);
      yield* session.messages;
      await session.result; // Propagates timeout/cancel/exit errors.
    },
  };
}

function start(
  config: GeminiRunnerConfig,
  spawnFn: SpawnFn,
  binary: string,
  prompt: string,
  options: GeminiRunOptions = {},
): Session<GeminiMessage> {
  const args = buildArgs(prompt, options);
  const { signal, clearTimeout: clearTO } = combinedSignal(options);
  const env = options.env ? { ...process.env, ...options.env } : undefined;
  logCmd(config, binary, args, options.workingDir);
  const child = spawnFn(binary, args, { cwd: options.workingDir, env, signal });

  if (!child.stdout) {
    const rejected = Promise.reject(
      new NotFoundError(`failed to start ${binary}: no stdout`),
    );
    rejected.catch(() => {});
    return {
      messages: (async function* () {})(),
      result: rejected,
      abort: () => {},
      send: () =>
        Promise.reject(new NotSupportedError("send is not supported")),
    };
  }

  const closePromise = once(child, "close") as Promise<[number | null]>;
  const rl = createInterface({ input: child.stdout });
  const stderrChunks: Buffer[] = [];
  child.stderr?.on("data", (chunk: Buffer) => stderrChunks.push(chunk));
  const stdoutErrors: string[] = [];

  let sessionId = "";
  let lastAssistantText = "";
  let resultMsg: ResultStreamMessage | undefined;
  let hadError = false;
  let errorMsg = "";

  let resolveResult: (value: Result) => void;
  let rejectResult: (reason: unknown) => void;
  const resultPromise = new Promise<Result>((resolve, reject) => {
    resolveResult = resolve;
    rejectResult = reject;
  });
  resultPromise.catch(() => {});

  async function* messageGenerator(): AsyncGenerator<GeminiMessage> {
    try {
      for await (const line of rl) {
        if (signal.aborted) break;
        if (!line) continue;

        let parsed: StreamMessage;
        try {
          parsed = parse(line);
        } catch {
          stdoutErrors.push(line);
          continue;
        }

        trackState(parsed);

        const msg: GeminiMessage = {
          type: mapMessageType(parsed),
          raw: line,
          data: parsed,
        };
        if (options.onMessage) options.onMessage(msg);
        yield msg;
      }

      const [exitCode] = await closePromise;
      clearTO();

      if (signal.aborted) {
        rejectResult!(abortError(signal));
        return;
      }
      if (hadError && !resultMsg) {
        resolveResult!(buildResult(errorMsg, true, sessionId));
        return;
      }
      if (resultMsg) {
        const text = resolveResultText(resultMsg, lastAssistantText);
        resolveResult!(
          buildResult(
            text,
            resultMsg.status === "error" || hadError,
            sessionId,
            resultMsg,
          ),
        );
        return;
      }

      const stderr = Buffer.concat(stderrChunks).toString("utf-8");
      if (exitCode != null && exitCode !== 0) {
        const detail = collectErrorDetail(stderr, stdoutErrors);
        if (config.logger) {
          config.logger.error("CLI command failed", {
            exit_code: exitCode,
            stderr: stderr.trim(),
            stdout_errors: stdoutErrors,
          });
        }
        rejectResult!(
          new NonZeroExitError(exitCode, `exit ${exitCode}: ${detail}`),
        );
        return;
      }
      rejectResult!(new NoResultError());
    } finally {
      if (child.exitCode === null) child.kill();
      clearTO();
    }
  }

  function trackState(parsed: StreamMessage): void {
    if (parsed.type === "init" && "session_id" in parsed && parsed.session_id) {
      sessionId = parsed.session_id;
    }
    if (parsed.type === "message") {
      const m = parsed as MessageStreamMessage;
      if (m.role === "assistant" && !m.delta) {
        lastAssistantText = m.content ?? "";
      }
    }
    if (parsed.type === "result") {
      resultMsg = parsed as ResultStreamMessage;
    }
    if (parsed.type === "error") {
      hadError = true;
      errorMsg = (parsed as ErrorStreamMessage).message ?? "";
    }
  }

  return {
    messages: messageGenerator(),
    result: resultPromise,
    abort: () => {
      if (child.exitCode === null) child.kill();
    },
    send: () => Promise.reject(new NotSupportedError("send is not supported")),
  };
}
