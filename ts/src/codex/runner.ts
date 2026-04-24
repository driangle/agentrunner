import { createInterface } from "node:readline";
import { once } from "node:events";
import type { Runner, Result, Session } from "../types.js";
import type { CodexMessage } from "./accessors.js";
import {
  NotFoundError,
  NonZeroExitError,
  NoResultError,
  NotSupportedError,
} from "../errors.js";
import type { CodexRunnerConfig, CodexRunOptions, SpawnFn } from "./options.js";
import type { StreamMessage, ItemStreamMessage, TurnUsage } from "./types.js";
import { parse } from "./parser.js";
import { buildArgs } from "./args.js";
import { combinedSignal, abortError } from "../signal.js";
import { logCmd, resolveSpawn, collectErrorDetail } from "./process.js";
import { mapMessageType } from "./mapping.js";

/** Create a Codex CLI runner. */
export function createCodexRunner(
  config: CodexRunnerConfig = {},
): Runner<CodexRunOptions, CodexMessage> {
  const { spawn: spawnFn, binary } = resolveSpawn(config);
  const s = (prompt: string, opts?: CodexRunOptions) =>
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
  config: CodexRunnerConfig,
  spawnFn: SpawnFn,
  binary: string,
  prompt: string,
  options: CodexRunOptions = {},
): Session<CodexMessage> {
  const args = buildArgs(prompt, options);
  const { signal, clearTimeout: clearTO } = combinedSignal(options);
  const env = options.env ? { ...process.env, ...options.env } : undefined;
  logCmd(config, binary, args, options.workingDir);
  const child = spawnFn(binary, args, { cwd: options.workingDir, env, signal });

  if (!child.stdout) {
    const err = new NotFoundError(`failed to start ${binary}: no stdout`);
    const rejected = Promise.reject(err);
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

  let threadId = "";
  let lastText = "";
  let turnUsage: TurnUsage | undefined;
  let hadError = false;
  let errorMsg = "";

  let resolveResult: (value: Result) => void;
  let rejectResult: (reason: unknown) => void;
  const resultPromise = new Promise<Result>((resolve, reject) => {
    resolveResult = resolve;
    rejectResult = reject;
  });
  resultPromise.catch(() => {});

  async function* messageGenerator(): AsyncGenerator<CodexMessage> {
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

        const msg: CodexMessage = {
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
      if (hadError && !turnUsage) {
        resolveResult!(buildResult(errorMsg, true));
        return;
      }
      if (turnUsage) {
        resolveResult!(buildResult(lastText, hadError));
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
    if (parsed.type === "thread.started" && parsed.thread_id) {
      threadId = parsed.thread_id;
    }
    if (parsed.type === "item.completed") {
      const item = (parsed as ItemStreamMessage).item;
      if (item?.type === "agent_message") lastText = item.text ?? "";
    }
    if (parsed.type === "turn.completed") {
      const msg = parsed as StreamMessage & { usage?: TurnUsage };
      if (msg.usage) turnUsage = msg.usage;
    }
    if (parsed.type === "error") {
      hadError = true;
      errorMsg = (parsed as StreamMessage & { message?: string }).message ?? "";
    }
    if (parsed.type === "turn.failed") {
      hadError = true;
      errorMsg =
        (parsed as StreamMessage & { error?: { message: string } }).error
          ?.message ?? "";
    }
  }

  function buildResult(text: string, isError: boolean): Result {
    return {
      text,
      isError,
      exitCode: 0,
      usage: {
        inputTokens: turnUsage?.input_tokens ?? 0,
        outputTokens: turnUsage?.output_tokens ?? 0,
        cacheReadInputTokens: turnUsage?.cached_input_tokens ?? 0,
      },
      costUSD: 0,
      durationMs: 0,
      sessionId: threadId,
    };
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
