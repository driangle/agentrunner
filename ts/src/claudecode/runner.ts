import { createInterface } from "node:readline";
import { once } from "node:events";
import type { Runner, Result, Message, Session } from "../types.js";
import { NotFoundError, NonZeroExitError, NoResultError } from "../errors.js";
import type { ClaudeRunnerConfig, ClaudeRunOptions, SpawnFn } from "./options.js";
import type { StreamMessage } from "./types.js";
import { parse } from "./parser.js";
import { buildArgs } from "./args.js";
import { mapMessageType, mapResult } from "./mapping.js";
import {
  combinedSignal,
  abortError,
  logCmd,
  resolveSpawn,
  collectErrorDetail,
} from "./process.js";

/** Create a Claude Code runner. */
export function createClaudeRunner(config: ClaudeRunnerConfig = {}): Runner {
  const { spawn: spawnFn, binary } = resolveSpawn(config);

  return {
    start: (prompt, options) =>
      start(config, spawnFn, binary, prompt, options as ClaudeRunOptions),
    run: (prompt, options) =>
      run(config, spawnFn, binary, prompt, options as ClaudeRunOptions),
    runStream: (prompt, options) =>
      runStream(config, spawnFn, binary, prompt, options as ClaudeRunOptions),
  };
}

function start(
  config: ClaudeRunnerConfig,
  spawnFn: SpawnFn,
  binary: string,
  prompt: string,
  options: ClaudeRunOptions = {},
): Session {
  const args = buildArgs(prompt, options);
  const { signal, clearTimeout: clearTO } = combinedSignal(options);

  const env = options.env ? { ...process.env, ...options.env } : undefined;

  logCmd(config, binary, args, options.workingDir);

  const child = spawnFn(binary, args, {
    cwd: options.workingDir,
    env,
    signal,
  });

  if (!child.stdout) {
    const err = new NotFoundError(`failed to start ${binary}: no stdout`);
    return {
      messages: (async function* () {})(),
      result: Promise.reject(err),
      abort: () => {},
      send: () => {
        throw new Error("not yet supported");
      },
    };
  }

  // Capture close promise before consuming stdout to avoid missing the event.
  const closePromise = once(child, "close") as Promise<[number | null]>;

  const rl = createInterface({ input: child.stdout });
  const stderrChunks: Buffer[] = [];
  child.stderr?.on("data", (chunk: Buffer) => stderrChunks.push(chunk));
  const stdoutErrors: string[] = [];

  let initSessionId = "";
  let resultMsg: StreamMessage | undefined;

  // Result promise that resolves when the process completes.
  let resolveResult: (value: Result) => void;
  let rejectResult: (reason: unknown) => void;
  const resultPromise = new Promise<Result>((resolve, reject) => {
    resolveResult = resolve;
    rejectResult = reject;
  });

  async function* messageGenerator(): AsyncGenerator<Message> {
    try {
      for await (const line of rl) {
        if (signal.aborted) {
          break;
        }
        if (!line) continue;

        let parsed: StreamMessage;
        try {
          parsed = parse(line);
        } catch {
          stdoutErrors.push(line);
          continue;
        }

        if (parsed.type === "system" && parsed.subtype === "init" && parsed.session_id) {
          initSessionId = parsed.session_id;
        }
        if (parsed.type === "result") {
          resultMsg = parsed;
        }

        const msg: Message = {
          type: mapMessageType(parsed.type),
          raw: line,
        };

        if (options.onMessage) {
          options.onMessage(msg);
        }

        yield msg;
      }

      const [exitCode] = await closePromise;

      clearTO();

      if (signal.aborted) {
        const err = abortError(signal);
        rejectResult!(err);
        return;
      }

      if (resultMsg) {
        resolveResult!(mapResult(resultMsg, initSessionId));
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
        rejectResult!(new NonZeroExitError(exitCode, `exit ${exitCode}: ${detail}`));
        return;
      }

      rejectResult!(new NoResultError());
    } finally {
      if (child.exitCode === null) {
        child.kill();
      }
      clearTO();
    }
  }

  const messages = messageGenerator();

  return {
    messages,
    result: resultPromise,
    abort: () => {
      if (child.exitCode === null) {
        child.kill();
      }
    },
    send: () => {
      throw new Error("not yet supported");
    },
  };
}

async function run(
  config: ClaudeRunnerConfig,
  spawnFn: SpawnFn,
  binary: string,
  prompt: string,
  options: ClaudeRunOptions = {},
): Promise<Result> {
  const session = start(config, spawnFn, binary, prompt, options);

  // Drain all messages.
  for await (const _msg of session.messages) {
    // consumed
  }

  return session.result;
}

async function* runStream(
  config: ClaudeRunnerConfig,
  spawnFn: SpawnFn,
  binary: string,
  prompt: string,
  options: ClaudeRunOptions = {},
): AsyncGenerator<Message> {
  const session = start(config, spawnFn, binary, prompt, options);
  yield* session.messages;

  // After messages are drained, check for errors on the result.
  // This propagates timeout/cancel/exit errors to the stream consumer.
  await session.result;
}
