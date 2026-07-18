from __future__ import annotations

from free_proxy.config import Settings
from free_proxy.domain.enums import NodeStatus, ProxyPolicyMode
from free_proxy.domain.models import ProxyNodeRead, TunnelStartResult
from free_proxy.infrastructure.database.repositories import (
    ProxyNodeRepository,
    SettingsRepository,
)
from free_proxy.services.gateway import GatewayService
from free_proxy.services.proxy_pool import ProxyPoolService


class AutoSwitchService:
    def __init__(
        self,
        app_settings: Settings,
        nodes: ProxyNodeRepository,
        settings: SettingsRepository,
        pool: ProxyPoolService,
        gateway: GatewayService,
    ) -> None:
        self._app_settings = app_settings
        self._nodes = nodes
        self._settings = settings
        self._pool = pool
        self._gateway = gateway

    async def switch(self) -> TunnelStartResult | None:
        settings = await self._settings.get()
        if not settings.connection_enabled:
            return None
        if settings.routing_mode is ProxyPolicyMode.FIXED:
            if not settings.fixed_node_id:
                return None
            return await self._gateway.activate(settings.fixed_node_id)

        excluded: set[str] = set()
        active_id = self._gateway.status().active_node_id
        if active_id:
            excluded.add(active_id)
        for _ in range(3):
            candidate = await self._select_excluding(excluded)
            if candidate is None:
                await self._gateway.disconnect_only()
                return None
            result = await self._gateway.activate(candidate.id)
            if result.success:
                return result
            excluded.add(candidate.id)
            await self._nodes.blacklist(
                candidate.id,
                result.message,
                self._app_settings.invalid_backoff_seconds,
            )
        return None

    async def handle_unexpected_exit(self) -> None:
        settings = await self._settings.get()
        if not settings.connection_enabled:
            return
        if settings.routing_mode is ProxyPolicyMode.FIXED and settings.fixed_node_id:
            await self._gateway.activate(settings.fixed_node_id)
        else:
            await self.switch()

    async def _select_excluding(self, excluded: set[str]) -> ProxyNodeRead | None:
        try:
            candidates = await self._nodes.list_nodes(
                limit=1000,
                status=NodeStatus.READY,
                current_only=True,
            )
        except TypeError:
            candidates = await self._nodes.list_nodes(limit=1000, status=NodeStatus.READY)
        settings = await self._settings.get()
        candidates = self._pool.apply_filters(candidates, settings)
        candidates = [node for node in candidates if node.id not in excluded]
        candidates.sort(key=lambda node: self._pool.sort_key_for(node, settings))
        return candidates[0] if candidates else None
