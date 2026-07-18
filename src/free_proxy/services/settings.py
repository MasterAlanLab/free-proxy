from __future__ import annotations

from free_proxy.domain.enums import ProxyPolicyMode
from free_proxy.domain.exceptions import ResourceNotFoundError
from free_proxy.domain.models import ProxySettings, ProxySettingsUpdate
from free_proxy.infrastructure.database.repositories import (
    ProxyNodeRepository,
    SettingsRepository,
)
from free_proxy.services.auto_switch import AutoSwitchService
from free_proxy.services.gateway import GatewayService
from free_proxy.services.proxy_pool import ProxyPoolService


class SettingsService:
    def __init__(
        self,
        nodes: ProxyNodeRepository,
        settings: SettingsRepository,
        pool: ProxyPoolService,
        gateway: GatewayService,
        auto_switch: AutoSwitchService,
    ) -> None:
        self._nodes = nodes
        self._settings = settings
        self._pool = pool
        self._gateway = gateway
        self._auto_switch = auto_switch

    async def get(self) -> ProxySettings:
        return await self._settings.get()

    async def update(self, payload: ProxySettingsUpdate) -> ProxySettings:
        updated = await self._settings.update(payload)
        self._gateway.set_connection_enabled(updated.connection_enabled)
        if not updated.connection_enabled:
            await self._gateway.disconnect_only()
            return updated
        await self._enforce_active_node(updated)
        if updated.connection_enabled and self._gateway.status().active_node_id is None:
            if updated.routing_mode is ProxyPolicyMode.FIXED and updated.fixed_node_id:
                await self._gateway.activate(updated.fixed_node_id)
            else:
                await self._auto_switch.switch()
        return updated

    async def toggle_favorite(self, node_id: str) -> ProxySettings:
        if await self._nodes.get_node(node_id) is None:
            raise ResourceNotFoundError(f"Proxy node not found: {node_id}")
        updated = await self._settings.toggle_favorite(node_id)
        if updated.routing_mode is ProxyPolicyMode.FAVORITES:
            await self._enforce_active_node(updated)
        return updated

    async def _enforce_active_node(self, settings: ProxySettings) -> None:
        active_id = self._gateway.status().active_node_id
        if not active_id:
            return
        active_node = await self._nodes.get_node(active_id)
        allowed = active_node is not None and bool(
            self._pool.apply_filters([active_node], settings)
        )
        if allowed:
            return
        await self._gateway.disconnect_only()
        if settings.connection_enabled and settings.routing_mode is not ProxyPolicyMode.FIXED:
            await self._auto_switch.switch()
