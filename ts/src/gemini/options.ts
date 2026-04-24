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

/** Configuration for creating a Gemini runner. */
export interface GeminiRunnerConfig {
  /** Override the CLI binary name (default: "gemini"). */
  binary?: string;

  /** Inject a custom spawn function for testing. */
  spawn?: SpawnFn;

  /** Structured logger for debug output (nil by default). */
  logger?: Logger;
}

/** Gemini-specific options that extend the common RunOptions. */
export interface GeminiRunOptions extends RunOptions {
  /** Approval behavior: "default", "auto_edit", "yolo", or "plan". */
  approvalMode?: string;

  /** Enable sandbox mode. */
  sandbox?: boolean;

  /** Extensions to use. */
  extensions?: string[];

  /** Tools allowed without confirmation. */
  allowedTools?: string[];

  /** Session ID or "latest" to resume. */
  resume?: string;

  /** Additional directories to add to workspace context. */
  includeDirs?: string[];

  /** Disable output sanitization. */
  rawOutput?: boolean;

  /** Bypass interactive permission prompts (maps to --yolo). */
  dangerouslySkipPermissions?: boolean;

  /** Callback invoked for each streaming message. */
  onMessage?: OnMessageFn;
}
