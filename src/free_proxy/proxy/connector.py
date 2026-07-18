from __future__ import annotations

import asyncio
import socket
from typing import Protocol

from free_proxy.proxy.dns import Resolver, SystemResolver, bind_to_device


class OutboundConnector(Protocol):
    async def connect(
        self,
        host: str,
        port: int,
    ) -> tuple[asyncio.StreamReader, asyncio.StreamWriter]: ...


class SocketConnector:
    def __init__(
        self,
        *,
        resolver: Resolver | None = None,
        interface: str | None = None,
        timeout: float = 20,
    ) -> None:
        self._resolver = resolver or SystemResolver()
        self._interface = interface
        self._timeout = timeout

    async def connect(
        self,
        host: str,
        port: int,
    ) -> tuple[asyncio.StreamReader, asyncio.StreamWriter]:
        resolved_host = await self._resolver.resolve(host)
        loop = asyncio.get_running_loop()
        addresses = await loop.getaddrinfo(
            resolved_host,
            port,
            type=socket.SOCK_STREAM,
        )
        addresses.sort(key=address_priority)
        last_error: OSError | None = None
        for family, socktype, protocol, _, address in addresses:
            sock = socket.socket(family, socktype, protocol)
            sock.setblocking(False)
            try:
                bind_to_device(sock, self._interface)
                await asyncio.wait_for(loop.sock_connect(sock, address), self._timeout)
                return await asyncio.open_connection(sock=sock)
            except OSError as exc:
                last_error = exc
                sock.close()
            except Exception:
                sock.close()
                raise
        if last_error is not None:
            raise last_error
        raise OSError(f"No address found for {host}:{port}")


def address_priority(address: tuple[object, ...]) -> int:
    return 0 if address[0] == socket.AF_INET else 1
