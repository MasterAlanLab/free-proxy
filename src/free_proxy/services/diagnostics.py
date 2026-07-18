from __future__ import annotations

import asyncio
import os
import shlex
import shutil
import socket
import ssl
import sys
import urllib.parse
from pathlib import Path

from free_proxy.config import Settings
from free_proxy.domain.exceptions import NetworkOperationError
from free_proxy.domain.models import DiagnosticCheck, DnsRepairResult, SystemDiagnostics
from free_proxy.infrastructure.network.commands import SystemCommandRunner


class NetworkDiagnosticsService:
    def __init__(
        self,
        settings: Settings,
        runner: SystemCommandRunner | None = None,
        platform: str | None = None,
    ) -> None:
        self._settings = settings
        self._runner = runner or SystemCommandRunner()
        self._platform = platform or sys.platform
        self._last_provider_checks: list[DiagnosticCheck] = []

    @property
    def auto_repair_enabled(self) -> bool:
        return self._settings.dns_repair_enabled

    async def diagnose(self, *, for_startup: bool = False) -> SystemDiagnostics:
        executable = shlex.split(self._settings.openvpn_command)[0]
        provider_host = urllib.parse.urlsplit(self._settings.vpngate_api_url).hostname or ""
        checks = [
            DiagnosticCheck(
                name="platform",
                ok=self._platform.startswith("linux"),
                detail=self._platform,
            ),
            self._root_check(),
            await self._data_directory_check(),
            await self._command_check("openvpn", executable),
            await self._command_check("ip", "ip"),
            await self._command_check("sysctl", "sysctl"),
            DiagnosticCheck(
                name="tun_device",
                ok=await asyncio.to_thread(Path("/dev/net/tun").exists),
                detail="/dev/net/tun",
            ),
            await self._tun_access_check(),
            await self._default_route_check(),
            await self._sysctl_check("ipv4_forwarding", "net.ipv4.ip_forward", {"1"}),
            await self._sysctl_check("rp_filter", "net.ipv4.conf.all.rp_filter", {"0", "2"}),
            await self._dns_check(provider_host),
            await self._port_check(require_available=for_startup),
            DiagnosticCheck(
                name="proxy_bindings",
                ok=self._local_proxy_bindings(),
                detail=f"http+socks5={self._settings.proxy_host}:{self._settings.proxy_port}",
            ),
        ]
        checks.extend(self._last_provider_checks)
        return SystemDiagnostics(healthy=all(check.ok for check in checks), checks=checks)

    async def diagnose_provider_failure(self, error: Exception) -> list[DiagnosticCheck]:
        host = urllib.parse.urlsplit(self._settings.vpngate_api_url).hostname or ""
        original = DiagnosticCheck(
            name="provider_last_error",
            ok=False,
            detail=f"{type(error).__name__}: {error}",
        )
        try:
            checks = await asyncio.wait_for(
                self._provider_failure_checks(host),
                timeout=min(8.0, self._settings.request_timeout_seconds),
            )
        except Exception as exc:
            checks = [
                DiagnosticCheck(
                    name="provider_diagnostics",
                    ok=False,
                    detail=f"diagnostics failed: {type(exc).__name__}: {exc}",
                    severity="warning",
                )
            ]
        self._last_provider_checks = [original, *checks]
        return list(self._last_provider_checks)

    def clear_provider_failure(self) -> None:
        self._last_provider_checks = []

    async def _provider_failure_checks(self, host: str) -> list[DiagnosticCheck]:
        checks = [await self._dns_check(host)]
        checks.append(await self._tcp_check("provider_tcp", host, 443))
        checks.append(await self._tcp_check("external_network", "1.1.1.1", 443))
        checks.append(await self._tls_check(host))
        return checks

    async def _tcp_check(self, name: str, host: str, port: int) -> DiagnosticCheck:
        try:
            _, writer = await asyncio.wait_for(asyncio.open_connection(host, port), timeout=2)
            writer.close()
            await writer.wait_closed()
            return DiagnosticCheck(name=name, ok=True, detail=f"{host}:{port} reachable")
        except (ConnectionError, OSError, TimeoutError) as exc:
            return DiagnosticCheck(name=name, ok=False, detail=str(exc))

    async def _tls_check(self, host: str) -> DiagnosticCheck:
        context = ssl.create_default_context()
        try:
            _, writer = await asyncio.wait_for(
                asyncio.open_connection(host, 443, ssl=context, server_hostname=host),
                timeout=3,
            )
            writer.close()
            await writer.wait_closed()
            return DiagnosticCheck(name="provider_tls", ok=True, detail="TLS handshake succeeded")
        except (ConnectionError, OSError, TimeoutError, ssl.SSLError) as exc:
            return DiagnosticCheck(name="provider_tls", ok=False, detail=str(exc))

    async def startup_preflight(self) -> SystemDiagnostics:
        result = await self.diagnose(for_startup=True)
        for check in result.checks:
            if not check.ok:
                continue
            if check.name == "data_directory":
                self._settings.ensure_directories()
        return result

    async def repair_dns(self) -> DnsRepairResult:
        if not self._settings.dns_repair_enabled:
            raise NetworkOperationError("DNS repair is disabled by FREE_PROXY_DNS_REPAIR_ENABLED")
        if not self._platform.startswith("linux"):
            raise NetworkOperationError("Automatic DNS repair is only supported on Linux")
        if await asyncio.to_thread(shutil.which, "resolvectl") is None:
            raise NetworkOperationError("resolvectl is required for automatic DNS repair")

        route = await self._runner.run(["ip", "route", "show", "default"])
        interface = parse_default_interface(route.stdout)
        if route.returncode != 0 or not interface:
            raise NetworkOperationError("Unable to determine the default network interface")
        servers = self._settings.parsed_dns_repair_servers
        if not servers:
            raise NetworkOperationError("No DNS repair servers are configured")

        commands = [
            ["resolvectl", "dns", interface, *servers],
            ["resolvectl", "domain", interface, "~."],
            ["resolvectl", "flush-caches"],
        ]
        for command in commands:
            result = await self._runner.run(command)
            if result.returncode != 0:
                raise NetworkOperationError(
                    f"DNS repair command failed: {' '.join(command)}: {result.stderr.strip()}"
                )
        return DnsRepairResult(
            repaired=True,
            interface=interface,
            servers=servers,
            detail="systemd-resolved DNS configuration updated",
        )

    async def _command_check(self, name: str, executable: str) -> DiagnosticCheck:
        path = await asyncio.to_thread(shutil.which, executable)
        return DiagnosticCheck(name=name, ok=path is not None, detail=path or "not found")

    def _root_check(self) -> DiagnosticCheck:
        if not self._platform.startswith("linux"):
            return DiagnosticCheck(
                name="root",
                ok=True,
                detail="not required outside Linux",
                severity="warning",
            )
        root = os.geteuid() == 0
        return DiagnosticCheck(name="root", ok=root, detail=f"uid={os.geteuid()}")

    async def _data_directory_check(self) -> DiagnosticCheck:
        try:
            await asyncio.to_thread(self._settings.ensure_directories)
            writable = await asyncio.to_thread(
                os.access,
                self._settings.data_dir,
                os.W_OK | os.X_OK,
            )
        except OSError as exc:
            return DiagnosticCheck(name="data_directory", ok=False, detail=str(exc))
        return DiagnosticCheck(
            name="data_directory",
            ok=writable,
            detail=str(self._settings.data_dir),
        )

    async def _tun_access_check(self) -> DiagnosticCheck:
        path = Path("/dev/net/tun")
        exists = await asyncio.to_thread(path.exists)
        if not exists:
            return DiagnosticCheck(name="tun_access", ok=False, detail="/dev/net/tun is missing")
        accessible = await asyncio.to_thread(os.access, path, os.R_OK | os.W_OK)
        return DiagnosticCheck(name="tun_access", ok=accessible, detail="read/write access")

    async def _default_route_check(self) -> DiagnosticCheck:
        result = await self._runner.run(["ip", "route", "show", "default"])
        interface = parse_default_interface(result.stdout)
        return DiagnosticCheck(
            name="default_route",
            ok=result.returncode == 0 and interface is not None,
            detail=interface or result.stderr.strip() or "no default route",
        )

    async def _sysctl_check(self, name: str, key: str, allowed: set[str]) -> DiagnosticCheck:
        result = await self._runner.run(["sysctl", "-n", key])
        value = result.stdout.strip()
        return DiagnosticCheck(
            name=name,
            ok=result.returncode == 0 and value in allowed,
            detail=f"{key}={value or result.stderr.strip()}",
            recoverable=True,
        )

    async def _port_check(self, *, require_available: bool) -> DiagnosticCheck:
        def can_bind() -> tuple[bool, str]:
            family = socket.AF_INET6 if ":" in self._settings.proxy_host else socket.AF_INET
            sock = socket.socket(family, socket.SOCK_STREAM)
            try:
                sock.bind((self._settings.proxy_host, self._settings.proxy_port))
                return True, f"{self._settings.proxy_host}:{self._settings.proxy_port} available"
            except OSError as exc:
                return False, str(exc)
            finally:
                sock.close()

        ok, detail = await asyncio.to_thread(can_bind)
        if not ok and not require_available:
            try:
                _, writer = await asyncio.wait_for(
                    asyncio.open_connection(
                        self._settings.proxy_host,
                        self._settings.proxy_port,
                    ),
                    timeout=1,
                )
                writer.close()
                await writer.wait_closed()
                return DiagnosticCheck(
                    name="proxy_port",
                    ok=True,
                    detail=(
                        f"{self._settings.proxy_host}:{self._settings.proxy_port} "
                        "accepting connections"
                    ),
                )
            except (ConnectionError, OSError, TimeoutError):
                pass
        return DiagnosticCheck(name="proxy_port", ok=ok, detail=detail, recoverable=True)

    async def _dns_check(self, host: str) -> DiagnosticCheck:
        if not host:
            return DiagnosticCheck(name="provider_dns", ok=False, detail="invalid provider URL")
        try:
            addresses = await asyncio.to_thread(socket.getaddrinfo, host, 443)
        except OSError as exc:
            return DiagnosticCheck(name="provider_dns", ok=False, detail=str(exc))
        unique = sorted({str(address[4][0]) for address in addresses})
        return DiagnosticCheck(name="provider_dns", ok=bool(unique), detail=", ".join(unique))

    def _local_proxy_bindings(self) -> bool:
        local_hosts = {"127.0.0.1", "::1", "localhost"}
        return self._settings.proxy_host in local_hosts


def parse_default_interface(route_output: str) -> str | None:
    parts = route_output.split()
    try:
        return parts[parts.index("dev") + 1]
    except (ValueError, IndexError):
        return None


def is_dns_error(exc: Exception) -> bool:
    text = str(exc).lower()
    markers = (
        "dns",
        "name or service not known",
        "temporary failure in name resolution",
        "nodename nor servname provided",
        "cannot resolve",
    )
    return any(marker in text for marker in markers)
