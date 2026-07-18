import asyncio
import socket

import pytest

from free_proxy.proxy.connector import SocketConnector
from free_proxy.proxy.http import HttpProxyServer
from free_proxy.proxy.socks5 import Socks5Server
from free_proxy.proxy.unified import UnifiedProxyServer


async def echo_handler(
    reader: asyncio.StreamReader,
    writer: asyncio.StreamWriter,
) -> None:
    try:
        while data := await reader.read(4096):
            writer.write(data)
            await writer.drain()
    finally:
        writer.close()
        await writer.wait_closed()


@pytest.mark.asyncio
async def test_socks5_connect_relays_data() -> None:
    target = await asyncio.start_server(echo_handler, "127.0.0.1", 0)
    target_port = int(target.sockets[0].getsockname()[1])
    proxy = Socks5Server(
        host="127.0.0.1",
        port=0,
        connector=SocketConnector(interface=None),
        idle_timeout=5,
    )
    await proxy.start()
    reader, writer = await asyncio.open_connection("127.0.0.1", proxy.bound_port)
    try:
        writer.write(b"\x05\x01\x00")
        await writer.drain()
        assert await reader.readexactly(2) == b"\x05\x00"

        request = (
            b"\x05\x01\x00\x01" + socket.inet_aton("127.0.0.1") + target_port.to_bytes(2, "big")
        )
        writer.write(request)
        await writer.drain()
        response = await reader.readexactly(10)
        assert response[:2] == b"\x05\x00"

        writer.write(b"free-proxy")
        await writer.drain()
        assert await reader.readexactly(10) == b"free-proxy"
    finally:
        writer.close()
        await writer.wait_closed()
        await proxy.stop()
        target.close()
        await target.wait_closed()


@pytest.mark.asyncio
async def test_http_connect_relays_data() -> None:
    target = await asyncio.start_server(echo_handler, "127.0.0.1", 0)
    target_port = int(target.sockets[0].getsockname()[1])
    proxy = HttpProxyServer(
        host="127.0.0.1",
        port=0,
        connector=SocketConnector(interface=None),
        idle_timeout=5,
    )
    await proxy.start()
    reader, writer = await asyncio.open_connection("127.0.0.1", proxy.bound_port)
    try:
        writer.write(
            f"CONNECT 127.0.0.1:{target_port} HTTP/1.1\r\n"
            f"Host: 127.0.0.1:{target_port}\r\n\r\n".encode()
        )
        await writer.drain()
        response = await reader.readuntil(b"\r\n\r\n")
        assert response.startswith(b"HTTP/1.1 200")

        writer.write(b"gateway")
        await writer.drain()
        assert await reader.readexactly(7) == b"gateway"
    finally:
        writer.close()
        await writer.wait_closed()
        await proxy.stop()
        target.close()
        await target.wait_closed()


@pytest.mark.asyncio
async def test_socks5_rejects_missing_username_password_authentication() -> None:
    proxy = Socks5Server(
        host="127.0.0.1",
        port=0,
        connector=SocketConnector(interface=None),
        username="proxy-user",
        password="proxy-pass",
    )
    await proxy.start()
    reader, writer = await asyncio.open_connection("127.0.0.1", proxy.bound_port)
    try:
        writer.write(b"\x05\x01\x00")
        await writer.drain()
        assert await reader.readexactly(2) == b"\x05\xff"
        assert await reader.read() == b""
    finally:
        writer.close()
        await writer.wait_closed()
        await proxy.stop()


@pytest.mark.asyncio
async def test_unified_listener_dispatches_socks5_and_http() -> None:
    target = await asyncio.start_server(echo_handler, "127.0.0.1", 0)
    target_port = int(target.sockets[0].getsockname()[1])
    connector = SocketConnector(interface=None)
    socks = Socks5Server(host="127.0.0.1", port=0, connector=connector)
    http = HttpProxyServer(host="127.0.0.1", port=0, connector=connector)
    proxy = UnifiedProxyServer(
        host="127.0.0.1",
        port=0,
        socks=socks,
        http=http,
        max_connections=10,
    )
    await proxy.start()
    try:
        reader, writer = await asyncio.open_connection("127.0.0.1", proxy.bound_port)
        writer.write(b"\x05\x01\x00")
        await writer.drain()
        assert await reader.readexactly(2) == b"\x05\x00"
        writer.close()
        await writer.wait_closed()

        reader, writer = await asyncio.open_connection("127.0.0.1", proxy.bound_port)
        writer.write(
            f"CONNECT 127.0.0.1:{target_port} HTTP/1.1\r\n\r\n".encode()
        )
        await writer.drain()
        assert (await reader.readuntil(b"\r\n\r\n")).startswith(b"HTTP/1.1 200")
        writer.close()
        await writer.wait_closed()
    finally:
        await proxy.stop()
        target.close()
        await target.wait_closed()
