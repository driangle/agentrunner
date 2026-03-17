/** Base class for all runner errors. */
export class RunnerError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "RunnerError";
  }
}

/** Runner binary or API endpoint is not reachable. */
export class NotFoundError extends RunnerError {
  constructor(message: string) {
    super(message);
    this.name = "NotFoundError";
  }
}

/** Execution exceeded the configured timeout. */
export class TimeoutError extends RunnerError {
  constructor(message: string) {
    super(message);
    this.name = "TimeoutError";
  }
}

/** CLI process exited with a non-zero code. */
export class NonZeroExitError extends RunnerError {
  exitCode: number;

  constructor(exitCode: number, message: string) {
    super(message);
    this.name = "NonZeroExitError";
    this.exitCode = exitCode;
  }
}

/** Failed to parse runner output. */
export class ParseError extends RunnerError {
  constructor(message: string) {
    super(message);
    this.name = "ParseError";
  }
}

/** Execution was cancelled by the caller. */
export class CancelledError extends RunnerError {
  constructor(message: string) {
    super(message);
    this.name = "CancelledError";
  }
}

/** Stream ended without a result message. */
export class NoResultError extends RunnerError {
  constructor() {
    super("no result in output");
    this.name = "NoResultError";
  }
}
