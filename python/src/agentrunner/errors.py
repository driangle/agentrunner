"""Exception hierarchy for runner errors."""


class RunnerError(Exception):
    """Base class for all runner errors."""


class NotFoundError(RunnerError):
    """Runner binary or API endpoint is not reachable."""


class TimeoutError(RunnerError):
    """Execution exceeded the configured timeout."""


class NonZeroExitError(RunnerError):
    """CLI process exited with a non-zero code."""

    def __init__(self, exit_code: int, message: str) -> None:
        super().__init__(message)
        self.exit_code = exit_code


class ParseError(RunnerError):
    """Failed to parse runner output."""


class CancelledError(RunnerError):
    """Execution was cancelled by the caller."""


class NoResultError(RunnerError):
    """Stream ended without a result message."""

    def __init__(self) -> None:
        super().__init__("no result in output")
