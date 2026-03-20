import { TimeoutError, CancelledError } from "./errors.js";
import type { RunOptions } from "./types.js";

/** Combine timeout and user-provided signal into a single AbortSignal. */
export function combinedSignal(options: RunOptions): {
  signal: AbortSignal;
  clearTimeout: () => void;
} {
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
export function abortError(signal: AbortSignal): TimeoutError | CancelledError {
  if (signal.reason === "timeout") {
    return new TimeoutError("execution timed out");
  }
  return new CancelledError("execution cancelled");
}
