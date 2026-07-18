from __future__ import annotations

from datetime import UTC, datetime, timedelta

from free_proxy.config import Settings
from free_proxy.domain.models import IpInfo
from free_proxy.infrastructure.database.repositories import ProxyNodeRepository
from free_proxy.infrastructure.ipinfo import IpInfoClient


class IpInfoService:
    def __init__(
        self,
        settings: Settings,
        client: IpInfoClient,
        repository: ProxyNodeRepository,
    ) -> None:
        self._settings = settings
        self._client = client
        self._repository = repository

    async def enrich(self, node_id: str, ip_address: str) -> None:
        await self.enrich_many({node_id: ip_address})

    async def enrich_many(self, nodes: dict[str, str]) -> None:
        now = datetime.now(UTC)
        cutoff = now - timedelta(seconds=self._settings.ip_info_cache_seconds)
        stale_nodes: dict[str, str] = {}
        cached: dict[str, IpInfo] = {}
        for node_id, ip_address in nodes.items():
            get_cache = getattr(self._repository, "cached_ip_info", None)
            entry = await get_cache(ip_address) if get_cache is not None else None
            if entry is not None:
                info, updated_at = entry
                cached[ip_address] = info
                if updated_at > cutoff:
                    await self._repository.update_ip_info(node_id, info, updated_at=updated_at)
                    continue
            stale_nodes[node_id] = ip_address
        if not stale_nodes:
            return
        results = await self._client.lookup_many(list(dict.fromkeys(stale_nodes.values())))
        for node_id, ip_address in stale_nodes.items():
            info = results.get(ip_address)
            if info is not None:
                cache = getattr(self._repository, "cache_ip_info", None)
                if cache is not None:
                    await cache(info, updated_at=now)
                else:
                    await self._repository.update_ip_info(node_id, info, updated_at=now)
            elif ip_address in cached:
                # Keep the previous timestamp so a failed lookup is retried
                # later instead of making stale data appear fresh.
                await self._repository.update_ip_info(node_id, cached[ip_address])
