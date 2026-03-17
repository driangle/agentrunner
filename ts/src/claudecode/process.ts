import { spawn as nodeSpawn } from "node:child_process";
import { TimeoutError, CancelledError } from "../errors.js";
import type { ClaudeRunnerConfig, ClaudeRunOptions, SpawnFn } from "./options.js";

/** Combine timeout and user-provided signal into a single AbortSignal. */
export function combinedSignal(
  options: ClaudeRunOptions,
): { signal: AbortSignal; clearTimeout: () => void } {
  const signals: AbortSignal[] = [];
  let timeoutId: ReturnType<typeof setTimeout> | undefined;

  if (options.signal) {
    signals.push(options.signal);
  }

  if (options.timeout != null && options.timeout > 0) {
    const controller = new AbortController();
    timeoutId = setTimeout(() => controller.abort("timeout"), options.timeout);
    signals.push(controller.signal);
  }

  if (signals.length === 0) {
    const controller = new AbortController();
    return { signal: controller.signal, clearTimeout: () => {} };
  }

  if (signals.length === 1) {
    return {
      signal: signals[0],
      clearTimeout: () => {
        if (timeoutId != null) clearTimeout(timeoutId);
      },
    };
  }

  const combined = AbortSignal.any(signals);
  return {
    signal: combined,
    clearTimeout: () => {
      if (timeoutId != null) clearTimeout(timeoutId);
    },
  };
}

/** Determine the abort error type from the signal's reason. */
export function abortError(
  signal: AbortSignal,
): TimeoutError | CancelledError {
  if (signal.reason === "timeout") {
    return new TimeoutError("execution timed out");
  }
  return new CancelledError("execution cancelled");
}

/** Log the command about to be executed. */
export function logCmd(
  config: ClaudeRunnerConfig,
  binary: string,
  args: string[],
  cwd?: string,
): void {
  if (!config.logger) return;

  const quoted = [binary, ...args].map((a) => {
    if (/[\s"'\\]/.test(a)) {
      return `'${a.replace(/'/g, "'\\''")}'`;
    }
    return a;
  });
  config.logger.debug("executing CLI command", {
    cmd: quoted.join(" "),
    dir: cwd ?? "",
  });
}

/** Resolve the spawn function, using the injected one or the default. */
export function resolveSpawn(
  config: ClaudeRunnerConfig,
): { spawn: SpawnFn; binary: string } {
  const binary = config.binary ?? "claude";

  if (config.spawn) {
    return { spawn: config.spawn, binary };
  }

  const defaultSpawn: SpawnFn = (command, args, options) => {
    return nodeSpawn(command, args, {
      cwd: options.cwd,
      env: options.env,
      signal: options.signal,
      stdio: ["ignore", "pipe", "pipe"],
    });
  };

  return { spawn: defaultSpawn, binary };
}

/**
 * Build a human-readable error string from stderr and unparseable stdout lines.
 */
export function collectErrorDetail(
  stderr: string,
  stdoutErrors: string[],
): string {
  const parts: string[] = [];
  const trimmed = stderr.trim();
  if (trimmed) {
    parts.push(trimmed);
  }
  if (stdoutErrors.length > 0) {
    parts.push(stdoutErrors.join("\n"));
  }
  if (parts.length === 0) {
    return "unknown error (no output from CLI)";
  }
  return parts.join("\n");
}
