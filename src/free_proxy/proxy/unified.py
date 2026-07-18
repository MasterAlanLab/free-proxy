from __future__ import annotations

import asyncio
from typing import Any, cast

from free_proxy.proxy.http import HttpProxyServer
from free_proxy.proxy.relay import close_writer
from free_proxy.proxy.socks5 import Socks5Server, format_listener


class UnifiedProxyServer:
    """Serve SOCKS5 and HTTP proxy clients on the same TCP listener."""

    def __init__(
        self,
        *,
        host: str,
        port: int,
        socks: Socks5Server,
        http: HttpProxyServer,
        max_connections: int,
    ) -> None:
        self._host = host
        self._port = port
        self._socks = socks
        self._http = http
        self._semaphore = asyncio.Semaphore(max_connections)
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

    async def _accept(self, reader: asyncio.StreamReader, writer: asyncio.StreamWriter) -> None:
        if self._semaphore.locked():
            await close_writer(writer)
            return
        async with self._semaphore:
            try:
                first_byte = await asyncio.wait_for(reader.readexactly(1), timeout=15)
                buffer = cast(Any, reader)._buffer
                buffer[:0] = first_byte  # Restore the sniffed byte ahead of buffered data.
                if first_byte == b"\x05":
                    await self._socks.handle(reader, writer)
                elif first_byte.isalpha():
                    await self._http.handle(reader, writer)
                else:
                    await close_writer(writer)
            except (asyncio.IncompleteReadError, ConnectionError, OSError, TimeoutError):
                await close_writer(writer)
