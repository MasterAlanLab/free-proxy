from __future__ import annotations

from free_proxy.config import Settings
from free_proxy.proxy.connector import OutboundConnector, SocketConnector
from free_proxy.proxy.dns import TunDnsResolver
from free_proxy.proxy.http import HttpProxyServer
from free_proxy.proxy.socks5 import Socks5Server
from free_proxy.proxy.unified import UnifiedProxyServer


class ProxyGateway:
    def __init__(
        self,
        settings: Settings,
        connector: OutboundConnector | None = None,
    ) -> None:
        self._settings = settings
        outbound_connector = connector or SocketConnector(
            resolver=TunDnsResolver(
                interface=settings.tunnel_interface,
                dns_server=settings.proxy_dns_server,
            ),
            interface=settings.tunnel_interface,
            timeout=settings.proxy_connect_timeout_seconds,
        )
        self.socks = Socks5Server(
            host=settings.proxy_host,
            port=settings.proxy_port,
            connector=outbound_connector,
            username=settings.proxy_username,
            password=settings.proxy_password,
            max_connections=settings.proxy_max_connections,
            idle_timeout=settings.proxy_idle_timeout_seconds,
        )
        self.http = HttpProxyServer(
            host=settings.proxy_host,
            port=settings.proxy_port,
            connector=outbound_connector,
            username=settings.proxy_username,
            password=settings.proxy_password,
            max_connections=settings.proxy_max_connections,
            idle_timeout=settings.proxy_idle_timeout_seconds,
        )
        self.server = UnifiedProxyServer(
            host=settings.proxy_host,
            port=settings.proxy_port,
            socks=self.socks,
            http=self.http,
            max_connections=settings.proxy_max_connections,
        )

    @property
    def running(self) -> bool:
        return self.server.running

    async def start(self) -> None:
        await self.server.start()

    async def stop(self) -> None:
        await self.server.stop()
