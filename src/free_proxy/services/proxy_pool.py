from __future__ import annotations

from free_proxy.domain.countries import normalize_country
from free_proxy.domain.enums import IpType, NodeStatus, ProxyPolicyMode, RoutingIpType
from free_proxy.domain.exceptions import FreeProxyError
from free_proxy.domain.models import PoolStatistics, ProxyNodeRead, ProxySettings
from free_proxy.infrastructure.database.repositories import (
    ProxyNodeRepository,
    SettingsRepository,
)


class ProxyPoolService:
    def __init__(
        self,
        nodes: ProxyNodeRepository,
        settings: SettingsRepository,
    ) -> None:
        self._nodes = nodes
        self._settings = settings

    async def select_best(self, *, exclude_node_id: str | None = None) -> ProxyNodeRead | None:
        settings = await self._settings.get()
        if not settings.connection_enabled:
            return None
        candidates = await self._nodes.list_nodes(
            limit=1000,
            status=NodeStatus.READY,
            current_only=True,
        )
        candidates = self.apply_filters(candidates, settings)
        if exclude_node_id:
            candidates = [node for node in candidates if node.id != exclude_node_id]
        candidates.sort(key=lambda node: self.sort_key_for(node, settings))
        return candidates[0] if candidates else None

    async def validate_allowed(self, node: ProxyNodeRead) -> None:
        settings = await self._settings.get()
        if not settings.connection_enabled:
            raise FreeProxyError("Proxy connections are disabled")
        if node not in self.apply_filters([node], settings):
            raise FreeProxyError("The selected node does not match the current routing settings")

    async def statistics(self) -> PoolStatistics:
        nodes = await self._nodes.list_nodes(limit=5000)
        settings = await self._settings.get()
        blacklisted = await self._nodes.active_blacklist_ids()
        return PoolStatistics(
            total=len(nodes),
            ready=sum(node.status is NodeStatus.READY for node in nodes),
            discovered=sum(node.status is NodeStatus.DISCOVERED for node in nodes),
            unavailable=sum(node.status is NodeStatus.UNAVAILABLE for node in nodes),
            cooldown=sum(node.status is NodeStatus.COOLDOWN for node in nodes),
            residential=sum(node.ip_type is IpType.RESIDENTIAL for node in nodes),
            mobile=sum(node.ip_type is IpType.MOBILE for node in nodes),
            hosting=sum(node.ip_type is IpType.HOSTING for node in nodes),
            unknown=sum(node.ip_type is IpType.UNKNOWN for node in nodes),
            favorites=len(settings.favorite_node_ids),
            blacklisted=len(blacklisted),
            countries=len(
                {
                    country
                    for node in nodes
                    if (country := node.country_code or node.country)
                }
            ),
        )

    @staticmethod
    def apply_filters(
        nodes: list[ProxyNodeRead],
        settings: ProxySettings,
        *,
        include_unknown_ip_type: bool = False,
    ) -> list[ProxyNodeRead]:
        candidates = list(nodes)
        if settings.routing_mode is ProxyPolicyMode.FIXED:
            candidates = [node for node in candidates if node.id == settings.fixed_node_id]
        elif settings.routing_mode is ProxyPolicyMode.COUNTRY and settings.force_country:
            target = normalize_country(settings.force_country)
            candidates = [node for node in candidates if normalize_country(node.country) == target]
        elif settings.routing_mode is ProxyPolicyMode.FAVORITES:
            favorite_ids = set(settings.favorite_node_ids)
            candidates = [node for node in candidates if node.id in favorite_ids]

        if settings.routing_mode is ProxyPolicyMode.FIXED:
            # A fixed node is only eligible when explicitly selected; it is never
            # treated as an automatic fallback candidate.
            candidates = [node for node in candidates if node.id == settings.fixed_node_id]

        if settings.routing_ip_type is RoutingIpType.RESIDENTIAL:
            candidates = [
                node
                for node in candidates
                if node.ip_type in (IpType.RESIDENTIAL, IpType.MOBILE)
                or (include_unknown_ip_type and node.ip_type is IpType.UNKNOWN)
            ]
        elif settings.routing_ip_type is RoutingIpType.HOSTING:
            candidates = [
                node
                for node in candidates
                if node.ip_type is IpType.HOSTING
                or (include_unknown_ip_type and node.ip_type is IpType.UNKNOWN)
            ]
        return candidates

    @staticmethod
    def sort_key(node: ProxyNodeRead) -> tuple[int, int, int]:
        residential_rank = 0 if node.ip_type in (IpType.RESIDENTIAL, IpType.MOBILE) else 1
        latency = node.latency_ms or 999_999
        return residential_rank, latency, -node.source_score

    @staticmethod
    def sort_key_for(node: ProxyNodeRead, settings: ProxySettings) -> tuple[int, int, int, int]:
        residential_rank = (
            0
            if node.ip_type in (IpType.RESIDENTIAL, IpType.MOBILE)
            else 1
        )
        latency = node.latency_ms or 999_999
        if settings.routing_mode is ProxyPolicyMode.RESIDENTIAL_FIRST:
            return residential_rank, latency, -node.source_score, -node.source_speed_bps
        return latency, -node.source_score, -node.source_speed_bps, residential_rank
