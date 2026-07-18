from __future__ import annotations

import asyncio
import ipaddress
import secrets
import socket

from free_proxy.domain.exceptions import FreeProxyError, NetworkOperationError
from free_proxy.proxy.connector import OutboundConnector
from free_proxy.proxy.relay import close_writer, relay_bidirectional


class SocksProtocolError(ValueError):
    def __init__(self, message: str, *, reply_sent: bool = False) -> None:
        super().__init__(message)
        self.reply_sent = reply_sent


class Socks5Server:
    def __init__(
        self,
        *,
        host: str,
        port: int,
        connector: OutboundConnector,
        username: str | None = None,
        password: str | None = None,
        max_connections: int = 256,
        idle_timeout: float = 120,
    ) -> None:
        self._host = host
        self._port = port
        self._connector = connector
        self._username = username
        self._password = password
        self._semaphore = asyncio.Semaphore(max_connections)
        self._idle_timeout = idle_timeout
        self._server: asyncio.Server | None = None

    @property
    def running(self) -> bool:
        return self._server is not None and self._server.is_serving()

    @property
    def listener(self) -> str:
        if self._server is not None and self._server.sockets:
            address = self._server.sockets[0].getsockname()
            return format_listener(str(address[0]), int(address[1]))
        return format_listener(self._host, self._port)

    @property
    def bound_port(self) -> int:
        if self._server is not None and self._server.sockets:
            return int(self._server.sockets[0].getsockname()[1])
        return self._port

    async def start(self) -> None:
        if self._server is None:
            self._server = await asyncio.start_server(self._accept, self._host, self._port)

    async def stop(self) -> None:
        if self._server is None:
            return
        self._server.close()
        await self._server.wait_closed()
        self._server = None

    async def _accept(
        self,
        reader: asyncio.StreamReader,
        writer: asyncio.StreamWriter,
    ) -> None:
        if self._semaphore.locked():
            await close_writer(writer)
            return
        async with self._semaphore:
            await self._handle(reader, writer)

    async def handle(self, reader: asyncio.StreamReader, writer: asyncio.StreamWriter) -> None:
        await self._handle(reader, writer)

    async def _handle(
        self,
        reader: asyncio.StreamReader,
        writer: asyncio.StreamWriter,
    ) -> None:
        upstream_writer: asyncio.StreamWriter | None = None
        connected = False
        try:
            await self._negotiate_auth(reader, writer)
            host, port = await self._read_connect_request(reader, writer)
            upstream_reader, upstream_writer = await self._connector.connect(host, port)
            writer.write(b"\x05\x00\x00\x01\x00\x00\x00\x00\x00\x00")
            await writer.drain()
            connected = True
            await relay_bidirectional(
                reader,
                writer,
                upstream_reader,
                upstream_writer,
                idle_timeout=self._idle_timeout,
            )
        except (
            asyncio.IncompleteReadError,
            ConnectionError,
            OSError,
            TimeoutError,
            ValueError,
            FreeProxyError,
        ) as exc:
            reply_sent = isinstance(exc, SocksProtocolError) and exc.reply_sent
            if not connected and not reply_sent and not writer.is_closing():
                try:
                    writer.write(
                        bytes([5, socks_failure_code(exc)])
                        + b"\x00\x01\x00\x00\x00\x00\x00\x00"
                    )
                    await writer.drain()
                except (ConnectionError, OSError):
                    pass
        finally:
            await close_writer(upstream_writer)
            await close_writer(writer)

    async def _negotiate_auth(
        self,
        reader: asyncio.StreamReader,
        writer: asyncio.StreamWriter,
    ) -> None:
        version, method_count = await reader.readexactly(2)
        if version != 5:
            raise SocksProtocolError("Invalid SOCKS version")
        methods = await reader.readexactly(method_count)
        auth_enabled = self._username is not None or self._password is not None
        selected_method = 2 if auth_enabled else 0
        if selected_method not in methods:
            writer.write(b"\x05\xff")
            await writer.drain()
            raise SocksProtocolError(
                "No supported SOCKS authentication method",
                reply_sent=True,
            )
        writer.write(bytes([5, selected_method]))
        await writer.drain()
        if auth_enabled:
            await self._authenticate(reader, writer)

    async def _authenticate(
        self,
        reader: asyncio.StreamReader,
        writer: asyncio.StreamWriter,
    ) -> None:
        version = (await reader.readexactly(1))[0]
        if version != 1:
            raise SocksProtocolError("Invalid SOCKS username/password version")
        username_length = (await reader.readexactly(1))[0]
        username = (await reader.readexactly(username_length)).decode("utf-8", errors="replace")
        password_length = (await reader.readexactly(1))[0]
        password = (await reader.readexactly(password_length)).decode("utf-8", errors="replace")
        valid = secrets.compare_digest(username, self._username or "") and secrets.compare_digest(
            password, self._password or ""
        )
        writer.write(b"\x01\x00" if valid else b"\x01\x01")
        await writer.drain()
        if not valid:
            raise SocksProtocolError("SOCKS authentication failed", reply_sent=True)

    async def _read_connect_request(
        self,
        reader: asyncio.StreamReader,
        writer: asyncio.StreamWriter,
    ) -> tuple[str, int]:
        version, command, _, address_type = await reader.readexactly(4)
        if version != 5 or command != 1:
            writer.write(b"\x05\x07\x00\x01\x00\x00\x00\x00\x00\x00")
            await writer.drain()
            raise SocksProtocolError("Only SOCKS CONNECT is supported", reply_sent=True)
        if address_type == 1:
            host = str(ipaddress.ip_address(await reader.readexactly(4)))
        elif address_type == 3:
            length = (await reader.readexactly(1))[0]
            host = (await reader.readexactly(length)).decode("idna")
        elif address_type == 4:
            host = socket.inet_ntop(socket.AF_INET6, await reader.readexactly(16))
        else:
            raise SocksProtocolError("Unsupported SOCKS address type")
        port = int.from_bytes(await reader.readexactly(2), "big")
        return host, port


def format_listener(host: str, port: int) -> str:
    return f"[{host}]:{port}" if ":" in host else f"{host}:{port}"


def socks_failure_code(exc: Exception) -> int:
    text = str(exc).lower()
    if isinstance(exc, NetworkOperationError) or "network" in text or "resolve" in text:
        return 3
    if "host" in text or "dns" in text:
        return 4
    if "refused" in text:
        return 5
    return 1
