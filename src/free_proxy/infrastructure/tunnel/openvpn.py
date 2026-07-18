from __future__ import annotations

import asyncio
import os
import re
import signal
from collections.abc import Awaitable, Callable
from pathlib import Path
from uuid import uuid4

from free_proxy.config import Settings
from free_proxy.domain.models import TunnelStartResult
from free_proxy.infrastructure.network.upstream import UpstreamProxy, get_upstream_proxy
from free_proxy.infrastructure.tunnel.command import OpenVpnCommandBuilder
from free_proxy.infrastructure.tunnel.process import ManagedOpenVpnProcess, OpenVpnProcessRunner


class OpenVpnManager:
    def __init__(
        self,
        settings: Settings,
        runner: OpenVpnProcessRunner | None = None,
    ) -> None:
        self._settings = settings
        self._runner = runner or OpenVpnProcessRunner()
        self._auth_file = settings.data_dir / "openvpn-auth.txt"
        self._upstream_auth_file = settings.data_dir / "upstream-proxy-auth.txt"
        self._builder = OpenVpnCommandBuilder(settings, self._auth_file)
        self._active: ManagedOpenVpnProcess | None = None
        self._active_node_id: str | None = None
        self._lock = asyncio.Lock()
        self._version: tuple[int, int] | None = None
        self._exit_handler: Callable[[int | None], Awaitable[None]] | None = None

    @property
    def active_node_id(self) -> str | None:
        return self._active_node_id

    @property
    def active_running(self) -> bool:
        return self._active is not None and self._active.running

    async def probe(self, config_text: str, device: str) -> TunnelStartResult:
        self._ensure_auth_file()
        config_path = self._write_config(config_text, prefix="probe")
        try:
            version = await self._detect_version()
            upstream, upstream_auth = self._upstream_options(config_text)
            command = self._builder.build(
                config_file=config_path,
                device=device,
                route_nopull=True,
                version=version,
                upstream_proxy=upstream,
                upstream_auth_file=upstream_auth,
            )
            result, _ = await self._runner.start(
                command=command,
                config_path=config_path,
                device=device,
                startup_timeout=self._settings.openvpn_test_timeout_seconds,
                keep_alive=False,
            )
            return result
        finally:
            config_path.unlink(missing_ok=True)

    async def connect(
        self,
        *,
        node_id: str,
        config_text: str,
    ) -> TunnelStartResult:
        async with self._lock:
            await self._disconnect_unlocked()
            self._ensure_auth_file()
            config_path = self._write_config(config_text, prefix="active")
            version = await self._detect_version()
            upstream, upstream_auth = self._upstream_options(config_text)
            command = self._builder.build(
                config_file=config_path,
                device=self._settings.tunnel_interface,
                route_nopull=True,
                version=version,
                upstream_proxy=upstream,
                upstream_auth_file=upstream_auth,
            )
            result, managed = await self._runner.start(
                command=command,
                config_path=config_path,
                device=self._settings.tunnel_interface,
                startup_timeout=self._settings.openvpn_connect_timeout_seconds,
                keep_alive=True,
            )
            if result.success and managed is not None:
                managed.set_exit_handler(self._exit_handler)
                self._active = managed
                self._active_node_id = node_id
            else:
                config_path.unlink(missing_ok=True)
            return result

    async def disconnect(self) -> None:
        async with self._lock:
            await self._disconnect_unlocked()

    def set_exit_handler(self, handler: Callable[[int | None], Awaitable[None]] | None) -> None:
        self._exit_handler = handler
        if self._active is not None:
            self._active.set_exit_handler(handler)

    def clear_exited_process(self) -> None:
        if self._active is not None and not self._active.running:
            self._active = None
            self._active_node_id = None

    async def cleanup_stale_processes(self) -> list[int]:
        terminated = await asyncio.to_thread(self._terminate_stale_processes)
        if terminated:
            await asyncio.sleep(0.5)
        return terminated

    def _terminate_stale_processes(self, proc_root: Path | None = None) -> list[int]:
        proc_root = proc_root or Path("/proc")
        if not proc_root.exists():
            return []
        markers = (
            str(self._settings.data_dir),
            str(self._auth_file),
            str(self._upstream_auth_file),
        )
        terminated: list[int] = []
        for proc_dir in proc_root.iterdir():
            if not proc_dir.name.isdigit():
                continue
            pid = int(proc_dir.name)
            if pid == os.getpid():
                continue
            try:
                raw = (proc_dir / "cmdline").read_bytes()
            except OSError:
                continue
            command = " ".join(
                part.decode("utf-8", errors="replace")
                for part in raw.split(b"\0")
                if part
            )
            if "openvpn" not in command.lower() or not any(
                marker in command for marker in markers
            ):
                continue
            try:
                os.kill(pid, signal.SIGTERM)
                terminated.append(pid)
            except (ProcessLookupError, PermissionError):
                continue
        return terminated

    async def _disconnect_unlocked(self) -> None:
        if self._active is not None:
            await self._active.stop()
        self._active = None
        self._active_node_id = None

    def _ensure_auth_file(self) -> None:
        self._settings.ensure_directories()
        self._auth_file.write_text(
            f"{self._settings.openvpn_username}\n{self._settings.openvpn_password}\n",
            encoding="utf-8",
        )
        try:
            self._auth_file.chmod(0o600)
        except OSError:
            pass

    def _write_config(self, config_text: str, *, prefix: str) -> Path:
        config_path = self._settings.data_dir / "configs" / f"{prefix}-{uuid4().hex}.ovpn"
        config_path.write_text(config_text, encoding="utf-8")
        return config_path

    def _upstream_options(
        self,
        config_text: str,
    ) -> tuple[UpstreamProxy | None, Path | None]:
        if not is_tcp_config(config_text):
            return None, None
        upstream = get_upstream_proxy(self._settings)
        if upstream is None:
            return None, None
        auth_file = self._write_upstream_auth(upstream)
        return upstream, auth_file

    def _write_upstream_auth(self, upstream: UpstreamProxy) -> Path | None:
        if upstream.username is None:
            return None
        self._upstream_auth_file.write_text(
            f"{upstream.username}\n{upstream.password or ''}\n",
            encoding="utf-8",
        )
        try:
            self._upstream_auth_file.chmod(0o600)
        except OSError:
            pass
        return self._upstream_auth_file

    async def _detect_version(self) -> tuple[int, int]:
        if self._version is not None:
            return self._version
        process: asyncio.subprocess.Process | None = None
        try:
            process = await asyncio.create_subprocess_exec(
                *self._builder.executable,
                "--version",
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.STDOUT,
                start_new_session=os.name != "nt",
            )
            stdout, _ = await asyncio.wait_for(process.communicate(), timeout=3)
            match = re.search(rb"OpenVPN\s+(\d+)\.(\d+)", stdout)
            if match:
                self._version = (int(match.group(1)), int(match.group(2)))
                return self._version
        except (FileNotFoundError, OSError, TimeoutError):
            if process is not None and process.returncode is None:
                process.kill()
                await process.wait()
        self._version = (2, 4)
        return self._version


def is_tcp_config(config_text: str) -> bool:
    for raw_line in config_text.splitlines():
        line = raw_line.strip().lower()
        if line.startswith("proto ") and "tcp" in line.split()[1]:
            return True
        if line.startswith("remote ") and " tcp" in f" {line}":
            return True
    return False
