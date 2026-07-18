from typing import Protocol

from free_proxy.domain.models import DiscoveredNode


class ProxyProvider(Protocol):
    name: str

    async def discover(self) -> list[DiscoveredNode]: ...

    async def close(self) -> None: ...
