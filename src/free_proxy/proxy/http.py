from __future__ import annotations

import asyncio
import base64
import secrets
import urllib.parse

from free_proxy.domain.exceptions import FreeProxyError
from free_proxy.proxy.connector import OutboundConnector
from free_proxy.proxy.relay import close_writer, relay_bidirectional
from free_proxy.proxy.socks5 import format_listener


class HttpProxyServer:
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
        response_started = False
        try:
            header = await asyncio.wait_for(
                reader.readuntil(b"\r\n\r\n"),
                timeout=15,
            )
            lines = header[:-4].decode("iso-8859-1").split("\r\n")
            method, target, version = lines[0].split(" ", 2)
            if not self._authorized(lines[1:]):
                writer.write(
                    b"HTTP/1.1 407 Proxy Authentication Required\r\n"
                    b'Proxy-Authenticate: Basic realm="Free Proxy"\r\n'
                    b"Content-Length: 0\r\n\r\n"
                )
                await writer.drain()
                return

            if method.upper() == "CONNECT":
                host, port = parse_authority(target, 443)
                upstream_reader, upstream_writer = await self._connector.connect(host, port)
                writer.write(b"HTTP/1.1 200 Connection Established\r\n\r\n")
                await writer.drain()
                response_started = True
            else:
                host, port, request = build_forward_request(method, target, version, lines[1:])
                upstream_reader, upstream_writer = await self._connector.connect(host, port)
                upstream_writer.write(request)
                await upstream_writer.drain()
                response_started = True

            await relay_bidirectional(
                reader,
                writer,
                upstream_reader,
                upstream_writer,
                idle_timeout=self._idle_timeout,
            )
        except (
            asyncio.IncompleteReadError,
            asyncio.LimitOverrunError,
            ConnectionError,
            OSError,
            TimeoutError,
            ValueError,
            FreeProxyError,
        ):
            if not response_started and not writer.is_closing():
                try:
                    writer.write(b"HTTP/1.1 502 Bad Gateway\r\nContent-Length: 0\r\n\r\n")
                    await writer.drain()
                except (ConnectionError, OSError):
                    pass
        finally:
            await close_writer(upstream_writer)
            await close_writer(writer)

    def _authorized(self, headers: list[str]) -> bool:
        if self._username is None and self._password is None:
            return True
        for header in headers:
            name, separator, value = header.partition(":")
            if not separator or name.lower() != "proxy-authorization":
                continue
            scheme, _, token = value.strip().partition(" ")
            if scheme.lower() != "basic":
                return False
            try:
                decoded = base64.b64decode(token, validate=True).decode("utf-8")
            except (ValueError, UnicodeDecodeError):
                return False
            username, separator, password = decoded.partition(":")
            return (
                bool(separator)
                and secrets.compare_digest(username, self._username or "")
                and secrets.compare_digest(password, self._password or "")
            )
        return False


def parse_authority(authority: str, default_port: int) -> tuple[str, int]:
    authority = authority.strip()
    if authority.startswith("["):
        host, separator, suffix = authority[1:].partition("]")
        if not separator:
            raise ValueError("Invalid IPv6 authority")
        return host, int(suffix[1:]) if suffix.startswith(":") else default_port
    if authority.count(":") == 1:
        host, port_text = authority.rsplit(":", 1)
        return host, int(port_text)
    return authority, default_port


def build_forward_request(
    method: str,
    target: str,
    version: str,
    headers: list[str],
) -> tuple[str, int, bytes]:
    parsed = urllib.parse.urlsplit(target)
    host = parsed.hostname
    port = parsed.port
    if host is None:
        host_header = next(
            (
                header.split(":", 1)[1].strip()
                for header in headers
                if header.lower().startswith("host:")
            ),
            "",
        )
        host, port = parse_authority(host_header, 80)
    if not host:
        raise ValueError("HTTP proxy request has no target host")
    port = port or (443 if parsed.scheme == "https" else 80)
    path = urllib.parse.urlunsplit(("", "", parsed.path or "/", parsed.query, ""))
    forwarded_headers = [
        header
        for header in headers
        if not header.lower().startswith(
            ("proxy-authorization:", "proxy-connection:", "connection:")
        )
    ]
    request = (
        f"{method} {path} {version}\r\n"
        + "\r\n".join(forwarded_headers)
        + "\r\nConnection: close\r\n\r\n"
    )
    return host, port, request.encode("iso-8859-1")
