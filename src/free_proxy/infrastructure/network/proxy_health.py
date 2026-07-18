from __future__ import annotations

import asyncio
import ipaddress
import logging

from free_proxy.config import Settings
from free_proxy.domain.models import ProxyHealthResult

logger = logging.getLogger(__name__)


class SocksProxyHealthChecker:
    def __init__(self, settings: Settings) -> None:
        self._settings = settings

    async def check(self) -> ProxyHealthResult:
        last_error = "Proxy exit check failed"
        for host in ("ip.sb", "api.ipify.org"):
            try:
                return await self._request_ip(host)
            except (
                asyncio.IncompleteReadError,
                EOFError,
                ConnectionError,
                OSError,
                TimeoutError,
                ValueError,
                IndexError,
            ) as exc:
                last_error = str(exc)
                logger.warning("Proxy health endpoint %s failed: %s", host, exc)
            except Exception as exc:
                last_error = f"{type(exc).__name__}: {exc}"
                logger.exception("Unexpected proxy health endpoint failure for %s", host)
        return ProxyHealthResult(ok=False, error=last_error)

    async def _request_ip(self, host: str) -> ProxyHealthResult:
        loop = asyncio.get_running_loop()
        started = loop.time()
        reader, writer = await asyncio.wait_for(
            asyncio.open_connection(self._settings.proxy_host, self._settings.proxy_port),
            timeout=self._settings.proxy_connect_timeout_seconds,
        )
        try:
            methods = b"\x00\x02" if self._settings.proxy_auth_enabled else b"\x00"
            writer.write(bytes([5, len(methods)]) + methods)
            await writer.drain()
            version, method = await reader.readexactly(2)
            if version != 5 or method == 255:
                raise ConnectionError("SOCKS5 proxy rejected health check authentication")
            if method == 2:
                await self._authenticate(reader, writer)

            host_bytes = host.encode("idna")
            writer.write(
                b"\x05\x01\x00\x03"
                + bytes([len(host_bytes)])
                + host_bytes
                + (80).to_bytes(2, "big")
            )
            await writer.drain()
            response = await reader.readexactly(4)
            if response[1] != 0:
                raise ConnectionError(f"SOCKS5 health check connect failed: {response[1]}")
            await consume_socks_address(reader, response[3])

            writer.write(f"GET / HTTP/1.1\r\nHost: {host}\r\nConnection: close\r\n\r\n".encode())
            await writer.drain()
            raw_response = await asyncio.wait_for(
                reader.read(),
                timeout=self._settings.proxy_connect_timeout_seconds,
            )
            if len(raw_response) > 64 * 1024:
                raise ConnectionError("Proxy exit IP response was too large")
            header, separator, body = raw_response.partition(b"\r\n\r\n")
            if not separator or b" 200 " not in header.split(b"\r\n", 1)[0]:
                raise ConnectionError("Proxy exit IP endpoint did not return HTTP 200")
            exit_ip = body.decode("utf-8", errors="replace").strip().splitlines()[0]
            ipaddress.ip_address(exit_ip)
            return ProxyHealthResult(
                ok=True,
                exit_ip=exit_ip,
                latency_ms=max(1, int((loop.time() - started) * 1000)),
            )
        finally:
            writer.close()
            try:
                await writer.wait_closed()
            except (ConnectionError, OSError):
                pass

    async def _authenticate(
        self,
        reader: asyncio.StreamReader,
        writer: asyncio.StreamWriter,
    ) -> None:
        username = (self._settings.proxy_username or "").encode()
        password = (self._settings.proxy_password or "").encode()
        writer.write(
            b"\x01" + bytes([len(username)]) + username + bytes([len(password)]) + password
        )
        await writer.drain()
        if await reader.readexactly(2) != b"\x01\x00":
            raise ConnectionError("SOCKS5 health check authentication failed")


async def consume_socks_address(reader: asyncio.StreamReader, address_type: int) -> None:
    if address_type == 1:
        await reader.readexactly(4)
    elif address_type == 3:
        length = (await reader.readexactly(1))[0]
        await reader.readexactly(length)
    elif address_type == 4:
        await reader.readexactly(16)
    else:
        raise ValueError("Invalid SOCKS5 response address type")
    await reader.readexactly(2)
