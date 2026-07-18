import signal
import sys
from pathlib import Path

import pytest

from free_proxy.config import Settings
from free_proxy.domain.enums import TunnelFailureCode
from free_proxy.infrastructure.tunnel.command import OpenVpnCommandBuilder
from free_proxy.infrastructure.tunnel.log_parser import OpenVpnLogParser
from free_proxy.infrastructure.tunnel.openvpn import OpenVpnManager
from free_proxy.infrastructure.tunnel.process import OpenVpnProcessRunner


def test_command_builder_selects_cipher_option_by_version(tmp_path: Path) -> None:
    settings = Settings(data_dir=tmp_path, openvpn_command="openvpn")
    builder = OpenVpnCommandBuilder(settings, tmp_path / "auth.txt")
    config = tmp_path / "node.ovpn"

    modern = builder.build(
        config_file=config,
        device="tun2",
        route_nopull=True,
        version=(2, 6),
    )
    legacy = builder.build(
        config_file=config,
        device="tun2",
        route_nopull=True,
        version=(2, 4),
    )

    assert "--data-ciphers" in modern
    assert "--ncp-ciphers" in legacy
    assert "--route-nopull" in modern
    assert modern[modern.index("--dev") + 1] == "tun2"


def test_log_parser_classifies_common_failures() -> None:
    assert OpenVpnLogParser.is_ready("Initialization Sequence Completed")
    assert OpenVpnLogParser.failure_code(["AUTH_FAILED"]) is TunnelFailureCode.AUTH_FAILED
    assert (
        OpenVpnLogParser.failure_code(["Cannot open TUN/TAP dev /dev/net/tun"])
        is TunnelFailureCode.TUN_UNAVAILABLE
    )


@pytest.mark.asyncio
async def test_process_runner_detects_successful_handshake(tmp_path: Path) -> None:
    config = tmp_path / "probe.ovpn"
    config.write_text("client\n")
    runner = OpenVpnProcessRunner()

    result, process = await runner.start(
        command=[
            sys.executable,
            "-c",
            "print('Initialization Sequence Completed', flush=True)",
        ],
        config_path=config,
        device="tun2",
        startup_timeout=2,
        keep_alive=False,
    )

    assert result.success is True
    assert process is None


@pytest.mark.asyncio
async def test_process_runner_reports_auth_failure(tmp_path: Path) -> None:
    config = tmp_path / "probe.ovpn"
    config.write_text("client\n")
    runner = OpenVpnProcessRunner()

    result, process = await runner.start(
        command=[sys.executable, "-c", "print('AUTH_FAILED', flush=True)"],
        config_path=config,
        device="tun2",
        startup_timeout=2,
        keep_alive=False,
    )

    assert result.success is False
    assert result.failure_code is TunnelFailureCode.AUTH_FAILED
    assert process is None


def test_tcp_openvpn_uses_authenticated_upstream_proxy(tmp_path: Path) -> None:
    settings = Settings(
        data_dir=tmp_path,
        upstream_proxy_url="socks5://proxy-user:proxy-pass@127.0.0.1:1080",
    )
    manager = OpenVpnManager(settings)
    upstream, auth_file = manager._upstream_options("client\nproto tcp-client\n")

    assert upstream is not None
    assert auth_file is not None
    assert auth_file.read_text(encoding="utf-8") == "proxy-user\nproxy-pass\n"
    command = manager._builder.build(
        config_file=tmp_path / "node.ovpn",
        device="tun0",
        route_nopull=True,
        upstream_proxy=upstream,
        upstream_auth_file=auth_file,
    )
    assert command[command.index("--socks-proxy") + 1 :] == [
        "127.0.0.1",
        "1080",
        str(auth_file),
    ]
    assert manager._upstream_options("client\nproto udp\n") == (None, None)


def test_stale_cleanup_only_terminates_project_openvpn_processes(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    settings = Settings(data_dir=tmp_path / "data")
    manager = OpenVpnManager(settings)
    proc_root = tmp_path / "proc"
    project_process = proc_root / "123"
    unrelated_process = proc_root / "124"
    project_process.mkdir(parents=True)
    unrelated_process.mkdir(parents=True)
    (project_process / "cmdline").write_bytes(
        b"openvpn\0--config\0" + str(settings.data_dir / "configs/node.ovpn").encode() + b"\0"
    )
    (unrelated_process / "cmdline").write_bytes(b"openvpn\0--config\0/tmp/other.ovpn\0")
    killed: list[tuple[int, int]] = []
    monkeypatch.setattr(
        "free_proxy.infrastructure.tunnel.openvpn.os.kill",
        lambda pid, sig: killed.append((pid, sig)),
    )

    terminated = manager._terminate_stale_processes(proc_root)

    assert terminated == [123]
    assert killed == [(123, signal.SIGTERM)]
