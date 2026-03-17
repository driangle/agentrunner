import { createInterface } from "node:readline";
import { once } from "node:events";
import type { Runner, Result, Message } from "../types.js";
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
    run: (prompt, options) =>
      run(config, spawnFn, binary, prompt, options as ClaudeRunOptions),
    runStream: (prompt, options) =>
      runStream(config, spawnFn, binary, prompt, options as ClaudeRunOptions),
  };
}

async function run(
  config: ClaudeRunnerConfig,
  spawnFn: SpawnFn,
  binary: string,
  prompt: string,
  options: ClaudeRunOptions = {},
): Promise<Result> {
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
    throw new NotFoundError(`failed to start ${binary}: no stdout`);
  }

  // Capture close promise before consuming stdout to avoid missing the event.
  const closePromise = once(child, "close") as Promise<[number | null]>;

  const rl = createInterface({ input: child.stdout });
  const stderrChunks: Buffer[] = [];
  child.stderr?.on("data", (chunk: Buffer) => stderrChunks.push(chunk));

  let resultMsg: StreamMessage | undefined;
  let initSessionId = "";
  const stdoutErrors: string[] = [];

  for await (const line of rl) {
    if (!line) continue;

    try {
      const msg = parse(line);
      if (msg.type === "system" && msg.subtype === "init" && msg.session_id) {
        initSessionId = msg.session_id;
      }
      if (msg.type === "result") {
        resultMsg = msg;
      }
    } catch {
      stdoutErrors.push(line);
    }
  }

  const [exitCode] = await closePromise;

  clearTO();

  if (signal.aborted) {
    throw abortError(signal);
  }

  if (resultMsg) {
    return mapResult(resultMsg, initSessionId);
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
    throw new NonZeroExitError(exitCode, `exit ${exitCode}: ${detail}`);
  }

  throw new NoResultError();
}

async function* runStream(
  config: ClaudeRunnerConfig,
  spawnFn: SpawnFn,
  binary: string,
  prompt: string,
  options: ClaudeRunOptions = {},
): AsyncGenerator<Message> {
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
    throw new NotFoundError(`failed to start ${binary}: no stdout`);
  }

  // Capture close promise before consuming stdout to avoid missing the event.
  const closePromise = once(child, "close") as Promise<[number | null]>;

  const rl = createInterface({ input: child.stdout });
  const stderrChunks: Buffer[] = [];
  child.stderr?.on("data", (chunk: Buffer) => stderrChunks.push(chunk));
  const stdoutErrors: string[] = [];

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
      throw abortError(signal);
    }

    if (exitCode != null && exitCode !== 0) {
      const stderr = Buffer.concat(stderrChunks).toString("utf-8");
      const detail = collectErrorDetail(stderr, stdoutErrors);
      if (config.logger) {
        config.logger.error("CLI command failed", {
          exit_code: exitCode,
          stderr: stderr.trim(),
          stdout_errors: stdoutErrors,
        });
      }
      throw new NonZeroExitError(exitCode, `exit ${exitCode}: ${detail}`);
    }
  } finally {
    if (child.exitCode === null) {
      child.kill();
    }
    clearTO();
  }
}
