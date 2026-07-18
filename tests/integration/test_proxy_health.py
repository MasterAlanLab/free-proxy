import asyncio
from pathlib import Path

import pytest

from free_proxy.config import Settings
from free_proxy.infrastructure.network.proxy_health import SocksProxyHealthChecker


async def fake_socks_ip_server(
    reader: asyncio.StreamReader,
    writer: asyncio.StreamWriter,
) -> None:
    try:
        version, method_count = await reader.readexactly(2)
        assert version == 5
        await reader.readexactly(method_count)
        writer.write(b"\x05\x00")
        await writer.drain()

        _, command, _, address_type = await reader.readexactly(4)
        assert command == 1
        assert address_type == 3
        host_length = (await reader.readexactly(1))[0]
        await reader.readexactly(host_length)
        await reader.readexactly(2)
        writer.write(b"\x05\x00\x00\x01\x00\x00\x00\x00\x00\x00")
        await writer.drain()

        await reader.readuntil(b"\r\n\r\n")
        writer.write(
            b"HTTP/1.1 200 OK\r\nContent-Length: 12\r\nConnection: close\r\n\r\n203.0.113.8\n"
        )
        await writer.drain()
    finally:
        writer.close()
        await writer.wait_closed()


@pytest.mark.asyncio
async def test_health_checker_reads_exit_ip_through_local_socks(tmp_path: Path) -> None:
    server = await asyncio.start_server(fake_socks_ip_server, "127.0.0.1", 0)
    port = int(server.sockets[0].getsockname()[1])
    settings = Settings(
        data_dir=tmp_path,
        proxy_port=port,
        proxy_connect_timeout_seconds=2,
    )
    try:
        result = await SocksProxyHealthChecker(settings).check()
    finally:
        server.close()
        await server.wait_closed()

    assert result.ok is True
    assert result.exit_ip == "203.0.113.8"
    assert result.latency_ms > 0
