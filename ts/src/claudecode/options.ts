import type { ChildProcess } from "node:child_process";
import type { RunOptions, OnMessageFn, Logger } from "../types.js";

/**
 * Function that spawns a child process. Used for dependency injection in tests.
 * Must return a ChildProcess with stdout as a readable stream.
 */
export type SpawnFn = (
  command: string,
  args: readonly string[],
  options: {
    cwd?: string;
    env?: NodeJS.ProcessEnv;
    signal?: AbortSignal;
  },
) => ChildProcess;

/** Configuration for creating a Claude Code runner. */
export interface ClaudeRunnerConfig {
  /** Override the CLI binary name (default: "claude"). */
  binary?: string;

  /** Inject a custom spawn function for testing. */
  spawn?: SpawnFn;

  /** Structured logger for debug output (nil by default). */
  logger?: Logger;
}

/** Claude Code-specific options that extend the common RunOptions. */
export interface ClaudeRunOptions extends RunOptions {
  /** Enable two-way channel communication via session.send(). */
  channelEnabled?: boolean;

  /** File path for channel MCP server logs. Only used when channelEnabled is true. */
  channelLogFile?: string;

  /** Log level for channel MCP server ("debug" | "info" | "warn" | "error"). Defaults to "info". */
  channelLogLevel?: "debug" | "info" | "warn" | "error";

  /** Bypass interactive permission prompts. */
  skipPermissions?: boolean;

  /** Tools the agent may use. */
  allowedTools?: string[];

  /** Tools the agent may not use. */
  disallowedTools?: string[];

  /** Path to MCP server configuration file. */
  mcpConfig?: string;

  /** JSON Schema for structured output. */
  jsonSchema?: string;

  /** Cost limit in USD. */
  maxBudgetUSD?: number;

  /** Session ID to resume. */
  resume?: string;

  /** Continue the most recent session. */
  continueSession?: boolean;

  /** Specific session ID for the conversation. */
  sessionId?: string;

  /** Enable streaming of partial/incremental messages. */
  includePartialMessages?: boolean;

  /** Callback invoked for each streaming message. */
  onMessage?: OnMessageFn;
}
