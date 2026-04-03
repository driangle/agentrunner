import { createInterface } from "node:readline";
import { once } from "node:events";
import type { Runner, Result, Session } from "../types.js";
import type { ClaudeMessage } from "./accessors.js";
import {
  NotFoundError,
  NonZeroExitError,
  NoResultError,
  NotSupportedError,
} from "../errors.js";
import {
  setupChannel,
  sendMessage,
  isChannelReply,
  type ChannelSetup,
  type ChannelMessage,
} from "./channel.js";
import type {
  ClaudeRunnerConfig,
  ClaudeRunOptions,
  SpawnFn,
} from "./options.js";
import type {
  StreamMessage,
  ResultStreamMessage,
  AssistantStreamMessage,
} from "./types.js";
import { parse } from "./parser.js";
import { buildArgs } from "./args.js";
import { mapResult } from "./mapping.js";
import { combinedSignal, abortError } from "../signal.js";
import { logCmd, resolveSpawn, collectErrorDetail } from "./process.js";

/** Create a Claude Code runner. */
export function createClaudeRunner(
  config: ClaudeRunnerConfig = {},
): Runner<ClaudeRunOptions, ClaudeMessage> {
  const { spawn: spawnFn, binary } = resolveSpawn(config);
  const s = (prompt: string, opts?: ClaudeRunOptions) =>
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
  config: ClaudeRunnerConfig,
  spawnFn: SpawnFn,
  binary: string,
  prompt: string,
  options: ClaudeRunOptions = {},
): Session<ClaudeMessage> {
  // Channel setup: create socket, MCP config, resolve binary.
  let chSetup: ChannelSetup | undefined;
  if (options.channelEnabled) {
    chSetup = setupChannel({
      mcpConfig: options.mcpConfig,
      logFile: options.channelLogFile,
      logLevel: options.channelLogLevel,
    });
    options = { ...options, mcpConfig: chSetup.mcpConfigPath };
  }

  const args = buildArgs(prompt, options);
  const { signal, clearTimeout: clearTO } = combinedSignal(options);
  const env = options.env ? { ...process.env, ...options.env } : undefined;
  logCmd(config, binary, args, options.workingDir);
  const child = spawnFn(binary, args, { cwd: options.workingDir, env, signal });

  if (!child.stdout) {
    chSetup?.cleanup();
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

  let initSessionId = "";
  let resultMsg: ResultStreamMessage | undefined;
  let resolveResult: (value: Result) => void;
  let rejectResult: (reason: unknown) => void;
  const resultPromise = new Promise<Result>((resolve, reject) => {
    resolveResult = resolve;
    rejectResult = reject;
  });
  resultPromise.catch(() => {});

  async function* messageGenerator(): AsyncGenerator<ClaudeMessage> {
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

        if (
          parsed.type === "system" &&
          parsed.subtype === "init" &&
          parsed.session_id
        ) {
          initSessionId = parsed.session_id;
        }
        if (parsed.type === "result") {
          resultMsg = parsed as ResultStreamMessage;
        }

        // Detect channel reply tool calls and remap message type.
        let msgType: string = parsed.type;
        if (
          parsed.type === "assistant" &&
          isChannelReply(parsed as AssistantStreamMessage)
        ) {
          msgType = "channel_reply";
        }

        const msg: ClaudeMessage = { type: msgType, raw: line, data: parsed };
        if (options.onMessage) options.onMessage(msg);
        yield msg;
      }

      const [exitCode] = await closePromise;
      clearTO();
      if (signal.aborted) {
        rejectResult!(abortError(signal));
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
        rejectResult!(
          new NonZeroExitError(exitCode, `exit ${exitCode}: ${detail}`),
        );
        return;
      }
      rejectResult!(new NoResultError());
    } finally {
      if (child.exitCode === null) child.kill();
      clearTO();
      chSetup?.cleanup();
    }
  }

  const sendFn = chSetup
    ? (input: unknown) => sendMessage(chSetup.sockPath, input as ChannelMessage)
    : () =>
        Promise.reject(new NotSupportedError("send requires channelEnabled"));

  return {
    messages: messageGenerator(),
    result: resultPromise,
    abort: () => {
      if (child.exitCode === null) child.kill();
    },
    send: sendFn,
  };
}
