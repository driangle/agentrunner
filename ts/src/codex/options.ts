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

/** Configuration for creating a Codex runner. */
export interface CodexRunnerConfig {
  /** Override the CLI binary name (default: "codex"). */
  binary?: string;

  /** Inject a custom spawn function for testing. */
  spawn?: SpawnFn;

  /** Structured logger for debug output (nil by default). */
  logger?: Logger;
}

/** Codex-specific options that extend the common RunOptions. */
export interface CodexRunOptions extends RunOptions {
  /** Sandbox policy: "read-only", "workspace-write", or "danger-full-access". */
  sandbox?: string;

  /** Approval policy: "untrusted", "on-request", or "never". */
  approval?: string;

  /** Path to a JSON Schema file for structured output validation. */
  outputSchema?: string;

  /** Image file paths for multimodal input. */
  images?: string[];

  /** Named config profile from config.toml. */
  profile?: string;

  /** Session ID to resume. */
  resume?: string;

  /** Enable live web search. */
  search?: boolean;

  /** Enable automatic execution with workspace-write sandbox. */
  fullAuto?: boolean;

  /** Run without persisting session files to disk. */
  ephemeral?: boolean;

  /** Additional directories that should be writable. */
  addDirs?: string[];

  /** Bypass interactive permission prompts. */
  dangerouslySkipPermissions?: boolean;

  /** Callback invoked for each streaming message. */
  onMessage?: OnMessageFn;
}
