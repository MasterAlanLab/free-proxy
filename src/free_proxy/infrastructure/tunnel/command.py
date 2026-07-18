from __future__ import annotations

import shlex
from pathlib import Path

from free_proxy.config import Settings
from free_proxy.infrastructure.network.upstream import UpstreamProxy


class OpenVpnCommandBuilder:
    def __init__(self, settings: Settings, auth_file: Path) -> None:
        self._settings = settings
        self._auth_file = auth_file

    @property
    def executable(self) -> list[str]:
        command = shlex.split(self._settings.openvpn_command)
        if not command:
            raise ValueError("FREE_PROXY_OPENVPN_COMMAND cannot be empty")
        return command

    def build(
        self,
        *,
        config_file: Path,
        device: str,
        route_nopull: bool,
        version: tuple[int, int] = (2, 5),
        upstream_proxy: UpstreamProxy | None = None,
        upstream_auth_file: Path | None = None,
    ) -> list[str]:
        command = [
            *self.executable,
            "--config",
            str(config_file),
            "--dev",
            device,
            "--dev-type",
            "tun",
            "--pull-filter",
            "ignore",
            "route-ipv6",
            "--pull-filter",
            "ignore",
            "ifconfig-ipv6",
            "--route-delay",
            "2",
            "--connect-retry-max",
            "1",
            "--connect-timeout",
            "15",
            "--auth-user-pass",
            str(self._auth_file),
            "--auth-nocache",
            "--verb",
            "3",
        ]
        if version >= (2, 5):
            command.extend(
                ["--data-ciphers", "AES-128-CBC:AES-256-GCM:AES-128-GCM:CHACHA20-POLY1305"]
            )
        else:
            command.extend(
                ["--ncp-ciphers", "AES-128-CBC:AES-256-GCM:AES-128-GCM:CHACHA20-POLY1305"]
            )
        if Path("/etc/ssl/certs").exists():
            command.extend(["--capath", "/etc/ssl/certs"])
        if route_nopull:
            command.append("--route-nopull")
        if upstream_proxy is not None:
            option = "--socks-proxy" if upstream_proxy.kind == "socks" else "--http-proxy"
            command.extend([option, upstream_proxy.host, str(upstream_proxy.port)])
            if upstream_auth_file is not None:
                command.append(str(upstream_auth_file))
        return command
