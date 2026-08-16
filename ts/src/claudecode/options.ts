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
  /** [Experimental] Enable two-way channel communication via session.send(). */
  channelEnabled?: boolean;

  /** File path for channel MCP server logs. Only used when channelEnabled is true. */
  channelLogFile?: string;

  /** Log level for channel MCP server ("debug" | "info" | "warn" | "error"). Defaults to "info". */
  channelLogLevel?: "debug" | "info" | "warn" | "error";

  /** Bypass interactive permission prompts. */
  dangerouslySkipPermissions?: boolean;

  /**
   * Permission mode for the run (--permission-mode). Known modes:
   * "default", "acceptEdits", "auto", "plan", "dontAsk", "bypassPermissions".
   * Passed through as a plain string so new CLI modes work without a library
   * change.
   *
   * When set, this takes precedence over `dangerouslySkipPermissions` — only
   * `--permission-mode` is passed.
   */
  permissionMode?: string;

  /** Tools the agent may use. */
  allowedTools?: string[];

  /** Tools the agent may not use. */
  disallowedTools?: string[];

  /**
   * Exclusive whitelist of built-in tools (--tools). Unlike `allowedTools`,
   * which only pre-approves tools and leaves unlisted built-ins available,
   * `--tools` replaces the built-in tool set, so every unlisted built-in is
   * denied at registration — including built-ins added by future CLI versions.
   *
   * Use `[""]` to disable all tools, `["default"]` to use all tools, or
   * explicit names (e.g. `["Read", "Grep"]`). Omit to leave the CLI default.
   */
  tools?: string[];

  /** Path to MCP server configuration file. */
  mcpConfig?: string;

  /** JSON Schema for structured output. */
  jsonSchema?: string;

  /**
   * Additional settings for the run (--settings). May be a path to a settings
   * JSON file or a raw JSON string — the value is passed through verbatim and
   * the CLI disambiguates.
   *
   * Note: in non-interactive mode the CLI silently ignores settings that fail
   * validation, so a malformed value will not surface an error.
   */
  settings?: string;

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

  /** Path to a file where the CLI writes debug logs (--debug-file). */
  debugFile?: string;

  /** Callback invoked for each streaming message. */
  onMessage?: OnMessageFn;
}
