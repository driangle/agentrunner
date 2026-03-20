"""Claude Code runner implementation."""

from __future__ import annotations

import asyncio
import os
from collections.abc import AsyncIterator
from typing import Any

from ..errors import (
    CancelledError,
    NonZeroExitError,
    NoResultError,
    NotFoundError,
    TimeoutError,
)
from ..types import Message, Result
from .args import build_args
from .mapping import map_message_type, map_result
from .options import ClaudeRunOptions, Logger
from .parser import parse
from .process import SpawnFn, collect_error_detail, default_spawn, log_cmd
from .types import StreamMessage
from .version import check_version


class ClaudeSession:
    """Session encapsulates a running Claude Code agent process.

    Supports ``async for msg in session`` to iterate messages,
    and ``await session.result`` to get the final result.
    """

    def __init__(
        self,
        logger: Logger | None,
        spawn_fn: SpawnFn,
        binary: str,
        prompt: str,
        options: ClaudeRunOptions,
    ) -> None:
        self._logger = logger
        self._spawn_fn = spawn_fn
        self._binary = binary
        self._prompt = prompt
        self._options = options

        self._loop = asyncio.get_running_loop()
        self._queue: asyncio.Queue[Message | None] = asyncio.Queue()
        self._result_future: asyncio.Future[Result] = self._loop.create_future()
        self._process: asyncio.subprocess.Process | None = None
        self._aborted = False
        self._timed_out = False
        self._task: asyncio.Task[None] | None = None

        # Launch the background task.
        self._task = asyncio.ensure_future(self._run_process())

    async def _run_process(self) -> None:
        args = build_args(self._prompt, self._options)
        env = {**os.environ, **self._options.env} if self._options.env else None

        log_cmd(self._logger, self._binary, args, self._options.working_dir)

        try:
            self._process = await self._spawn_fn(
                self._binary,
                *args,
                cwd=self._options.working_dir,
                env=env,
            )
        except FileNotFoundError:
            err = NotFoundError(f"failed to start {self._binary}: command not found")
            self._result_future.set_exception(err)
            await self._queue.put(None)
            return

        if self._process.stdout is None:
            err = NotFoundError(f"failed to start {self._binary}: no stdout")
            self._result_future.set_exception(err)
            await self._queue.put(None)
            return

        # Set up timeout (options.timeout is in seconds).
        timeout_handle: asyncio.TimerHandle | None = None
        if self._options.timeout is not None and self._options.timeout > 0:
            timeout_handle = self._loop.call_later(self._options.timeout, self._on_timeout)

        init_session_id = ""
        result_msg: StreamMessage | None = None
        stdout_errors: list[str] = []

        try:
            while True:
                raw_line = await self._process.stdout.readline()
                if not raw_line:
                    break

                line = raw_line.decode("utf-8", errors="replace").rstrip("\n").rstrip("\r")
                if not line:
                    continue

                if self._aborted or self._timed_out:
                    break

                try:
                    parsed = parse(line)
                except Exception:
                    stdout_errors.append(line)
                    continue

                if parsed.type == "system" and parsed.subtype == "init" and parsed.session_id:
                    init_session_id = parsed.session_id
                if parsed.type == "result":
                    result_msg = parsed

                msg = Message(type=map_message_type(parsed.type, line), raw=line)

                await self._queue.put(msg)

            # Wait for process to finish.
            await self._process.wait()

            if timeout_handle:
                timeout_handle.cancel()

            if self._timed_out:
                self._result_future.set_exception(TimeoutError("execution timed out"))
                return

            if self._aborted:
                self._result_future.set_exception(CancelledError("execution cancelled"))
                return

            if result_msg:
                self._result_future.set_result(map_result(result_msg, init_session_id))
                return

            exit_code = self._process.returncode
            if exit_code is not None and exit_code != 0:
                stderr_bytes = await self._process.stderr.read() if self._process.stderr else b""
                stderr = stderr_bytes.decode("utf-8", errors="replace")
                detail = collect_error_detail(stderr, stdout_errors)
                if self._logger:
                    self._logger.error(
                        "CLI command failed",
                        extra={
                            "exit_code": exit_code,
                            "stderr": stderr.strip(),
                            "stdout_errors": stdout_errors,
                        },
                    )
                self._result_future.set_exception(
                    NonZeroExitError(exit_code, f"exit {exit_code}: {detail}")
                )
                return

            self._result_future.set_exception(NoResultError())
        finally:
            await self._queue.put(None)
            if self._process.returncode is None:
                try:
                    self._process.kill()
                except ProcessLookupError:
                    pass
            if timeout_handle:
                timeout_handle.cancel()

    def _on_timeout(self) -> None:
        self._timed_out = True
        if self._process and self._process.returncode is None:
            try:
                self._process.kill()
            except ProcessLookupError:
                pass

    def __aiter__(self) -> AsyncIterator[Message]:
        return self._message_iter()

    async def _message_iter(self) -> AsyncIterator[Message]:
        while True:
            msg = await self._queue.get()
            if msg is None:
                break
            yield msg

    @property
    def result(self) -> asyncio.Future[Result]:
        return self._result_future

    def abort(self) -> None:
        self._aborted = True
        if self._process and self._process.returncode is None:
            try:
                self._process.kill()
            except ProcessLookupError:
                pass

    def send(self, input: Any) -> None:
        raise NotImplementedError("send is not yet supported")


class ClaudeRunner:
    """Claude Code runner.

    Construct directly with keyword arguments::

        runner = ClaudeRunner(binary="/usr/local/bin/claude", logger=my_logger)

    For testing, pass ``_spawn`` to inject a fake subprocess spawner::

        runner = ClaudeRunner(_spawn=my_fake_spawn)
    """

    def __init__(
        self,
        *,
        binary: str = "claude",
        logger: Logger | None = None,
        _spawn: SpawnFn | None = None,
    ) -> None:
        self._binary = binary
        self._logger = logger
        self._spawn_fn: SpawnFn = _spawn or default_spawn
        self._has_custom_spawn = _spawn is not None
        self._version_checked = False

    async def _ensure_version(self) -> None:
        """Check the CLI version once per runner instance."""
        if self._version_checked or self._has_custom_spawn:
            return
        await check_version(self._binary)
        self._version_checked = True

    def start(
        self,
        prompt: str,
        options: ClaudeRunOptions | None = None,
    ) -> ClaudeSession:
        return ClaudeSession(
            self._logger,
            self._spawn_fn,
            self._binary,
            prompt,
            options or ClaudeRunOptions(),
        )

    async def run(
        self,
        prompt: str,
        options: ClaudeRunOptions | None = None,
    ) -> Result:
        await self._ensure_version()
        session = self.start(prompt, options)
        async for _msg in session:
            pass
        return await session.result

    async def run_stream(
        self,
        prompt: str,
        options: ClaudeRunOptions | None = None,
    ) -> ClaudeSession:
        """Start a session and return it for streaming.

        The returned session is async-iterable and also provides
        ``session.result`` and ``session.abort()``.
        """
        await self._ensure_version()
        return self.start(prompt, options)
