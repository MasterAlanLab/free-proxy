from __future__ import annotations

import asyncio
import logging
import os
import signal
from collections import deque
from collections.abc import Awaitable, Callable
from contextlib import suppress
from pathlib import Path

from free_proxy.domain.enums import TunnelFailureCode, TunnelStatus
from free_proxy.domain.models import TunnelStartResult
from free_proxy.infrastructure.tunnel.log_parser import OpenVpnLogParser

logger = logging.getLogger("free_proxy.openvpn")


class ManagedOpenVpnProcess:
    def __init__(
        self,
        process: asyncio.subprocess.Process,
        config_path: Path,
        device: str,
        log_tail: deque[str],
        reader_task: asyncio.Task[None] | None,
        on_unexpected_exit: Callable[[int | None], Awaitable[None]] | None = None,
    ) -> None:
        self.process = process
        self.config_path = config_path
        self.device = device
        self.log_tail = log_tail
        self.reader_task = reader_task
        self._on_unexpected_exit = on_unexpected_exit
        self._intentional_stop = False
        self._watcher_task = asyncio.create_task(self._watch_exit(), name=f"openvpn-watch:{device}")

    @property
    def running(self) -> bool:
        return self.process.returncode is None

    async def stop(self) -> None:
        self._intentional_stop = True
        self._watcher_task.cancel()
        with suppress(asyncio.CancelledError):
            await self._watcher_task
        if self.reader_task is not None:
            self.reader_task.cancel()
        await terminate_process(self.process)
        if self.reader_task is not None:
            try:
                await self.reader_task
            except asyncio.CancelledError:
                pass
        self.config_path.unlink(missing_ok=True)

    def set_exit_handler(self, handler: Callable[[int | None], Awaitable[None]] | None) -> None:
        self._on_unexpected_exit = handler

    async def _watch_exit(self) -> None:
        try:
            code = await self.process.wait()
            if not self._intentional_stop and self._on_unexpected_exit is not None:
                await self._on_unexpected_exit(code)
        except asyncio.CancelledError:
            raise
        except Exception:
            logger.exception("OpenVPN process exit watcher failed")


class OpenVpnProcessRunner:
    async def start(
        self,
        *,
        command: list[str],
        config_path: Path,
        device: str,
        startup_timeout: float,
        keep_alive: bool,
    ) -> tuple[TunnelStartResult, ManagedOpenVpnProcess | None]:
        started = asyncio.get_running_loop().time()
        try:
            process = await asyncio.create_subprocess_exec(
                *command,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.STDOUT,
                start_new_session=os.name != "nt",
            )
        except FileNotFoundError:
            return (
                TunnelStartResult(
                    success=False,
                    status=TunnelStatus.FAILED,
                    message="OpenVPN executable was not found",
                    failure_code=TunnelFailureCode.COMMAND_NOT_FOUND,
                ),
                None,
            )
        except OSError as exc:
            return (
                TunnelStartResult(
                    success=False,
                    status=TunnelStatus.FAILED,
                    message=f"OpenVPN could not be started: {exc}",
                    failure_code=TunnelFailureCode.START_FAILED,
                ),
                None,
            )

        if process.stdout is None:
            await terminate_process(process)
            return (
                TunnelStartResult(
                    success=False,
                    status=TunnelStatus.FAILED,
                    message="OpenVPN stdout pipe was not created",
                    failure_code=TunnelFailureCode.START_FAILED,
                ),
                None,
            )

        log_tail: deque[str] = deque(maxlen=50)
        try:
            while True:
                elapsed = asyncio.get_running_loop().time() - started
                remaining = startup_timeout - elapsed
                if remaining <= 0:
                    raise TimeoutError
                raw_line = await asyncio.wait_for(process.stdout.readline(), timeout=remaining)
                if not raw_line:
                    break
                line = raw_line.decode("utf-8", errors="replace").rstrip()
                log_tail.append(line)
                logger.info("%s", line)
                if OpenVpnLogParser.is_ready(line):
                    startup_time_ms = int((asyncio.get_running_loop().time() - started) * 1000)
                    if not keep_alive:
                        await terminate_process(process)
                        return (
                            TunnelStartResult(
                                success=True,
                                status=TunnelStatus.CONNECTED,
                                message="Tunnel handshake completed",
                                startup_time_ms=startup_time_ms,
                                log_tail=list(log_tail),
                                handshake_stage="connected",
                            ),
                            None,
                        )
                    reader_task = asyncio.create_task(
                        drain_output(process, log_tail),
                        name=f"openvpn-log:{device}",
                    )
                    managed = ManagedOpenVpnProcess(
                        process,
                        config_path,
                        device,
                        log_tail,
                        reader_task,
                    )
                    return (
                        TunnelStartResult(
                            success=True,
                            status=TunnelStatus.CONNECTED,
                            message="Tunnel connected",
                            startup_time_ms=startup_time_ms,
                            log_tail=list(log_tail),
                            handshake_stage="connected",
                        ),
                        managed,
                    )
                if OpenVpnLogParser.is_terminal_failure(line):
                    break
        except TimeoutError:
            log_tail.append(f"OpenVPN timeout after {startup_timeout:.1f}s")

        await terminate_process(process)
        lines = list(log_tail)
        return (
            TunnelStartResult(
                success=False,
                status=TunnelStatus.FAILED,
                message=OpenVpnLogParser.failure_message(lines),
                failure_code=OpenVpnLogParser.failure_code(lines),
                startup_time_ms=int((asyncio.get_running_loop().time() - started) * 1000),
                log_tail=lines,
                handshake_stage=OpenVpnLogParser.handshake_stage(lines),
            ),
            None,
        )


async def drain_output(
    process: asyncio.subprocess.Process,
    log_tail: deque[str],
) -> None:
    if process.stdout is None:
        return
    while True:
        raw_line = await process.stdout.readline()
        if not raw_line:
            return
        log_tail.append(raw_line.decode("utf-8", errors="replace").rstrip())
        logger.info("%s", log_tail[-1])


async def terminate_process(process: asyncio.subprocess.Process) -> None:
    if process.returncode is not None:
        return
    try:
        if os.name != "nt":
            os.killpg(process.pid, signal.SIGTERM)
        else:
            process.terminate()
        await asyncio.wait_for(process.wait(), timeout=8)
    except (ProcessLookupError, PermissionError):
        return
    except TimeoutError:
        if process.returncode is None:
            if os.name != "nt":
                try:
                    os.killpg(process.pid, signal.SIGKILL)
                except ProcessLookupError:
                    pass
            else:
                process.kill()
            await process.wait()
