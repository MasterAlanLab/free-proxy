from __future__ import annotations

import asyncio
import logging
from typing import Any

from free_proxy.config import Settings
from free_proxy.domain.enums import TunnelStatus
from free_proxy.domain.exceptions import ResourceNotFoundError
from free_proxy.domain.models import GatewayStatus, TunnelStartResult
from free_proxy.infrastructure.database.repositories import ProxyNodeRepository, SettingsRepository
from free_proxy.infrastructure.network.routing import PolicyRouter
from free_proxy.infrastructure.tunnel.openvpn import OpenVpnManager
from free_proxy.proxy.gateway import ProxyGateway
from free_proxy.services.operations import NetworkOperationCoordinator
from free_proxy.services.proxy_pool import ProxyPoolService

logger = logging.getLogger(__name__)


class GatewayService:
    def __init__(
        self,
        settings: Settings,
        repository: ProxyNodeRepository,
        tunnel_manager: OpenVpnManager,
        policy_router: PolicyRouter,
        proxy_gateway: ProxyGateway,
        proxy_pool: ProxyPoolService,
        settings_repository: SettingsRepository | None = None,
        coordinator: NetworkOperationCoordinator | None = None,
    ) -> None:
        self._settings = settings
        self._repository = repository
        self._tunnel_manager = tunnel_manager
        self._policy_router = policy_router
        self._proxy_gateway = proxy_gateway
        self._proxy_pool = proxy_pool
        self._settings_repository = settings_repository
        self._coordinator = coordinator
        self._operation_lock = asyncio.Lock()
        self._last_error: str | None = None
        self._active_latency_ms = 0
        self._exit_ip: str | None = None
        self._exit_latency_ms = 0
        self._unexpected_exit_handler: Any = None
        self._connection_enabled = True

    async def start(self) -> None:
        if self._settings.proxy_enabled:
            await self._proxy_gateway.start()

    async def shutdown(self) -> None:
        await self.disconnect_only()
        await self._proxy_gateway.stop()

    async def activate(self, node_id: str) -> TunnelStartResult:
        if self._coordinator is None:
            async with self._operation_lock:
                return await self._activate(node_id)
        async with self._coordinator.acquire("activate"):
            async with self._operation_lock:
                return await self._activate(node_id)

    async def _activate(self, node_id: str) -> TunnelStartResult:
        logger.info("Activating proxy exit node: %s", node_id)
        target = await self._repository.get_target(node_id)
        if target is None:
            raise ResourceNotFoundError(f"Proxy node not found: {node_id}")
        node = await self._repository.get_node(node_id)
        if node is None:
            raise ResourceNotFoundError(f"Proxy node not found: {node_id}")
        if self._settings_repository is not None:
            await self._settings_repository.set_connection_enabled(True)
        self._connection_enabled = True
        await self._proxy_pool.validate_allowed(node)

        result = await self._tunnel_manager.connect(
            node_id=node_id,
            config_text=target.config_text,
        )
        if not result.success:
            await self._repository.mark_unavailable(node_id)
            self._last_error = result.message
            logger.error("Failed to activate node %s: %s", node_id, result.message)
            return result
        try:
            await self._policy_router.setup(self._settings.tunnel_interface)
        except Exception as exc:
            await self._tunnel_manager.disconnect()
            await self._repository.mark_unavailable(node_id)
            self._last_error = str(exc)
            raise
        self._last_error = None
        self._active_latency_ms = 0
        logger.info("Proxy exit node activated: %s", node_id)
        return result

    def set_unexpected_exit_handler(self, handler: Any) -> None:
        self._unexpected_exit_handler = handler
        self._tunnel_manager.set_exit_handler(self._handle_unexpected_exit)

    async def _handle_unexpected_exit(self, returncode: int | None) -> None:
        if self._tunnel_manager.active_node_id is None:
            return
        self._last_error = f"OpenVPN exited unexpectedly (code={returncode})"
        await self._policy_router.cleanup()
        self._active_latency_ms = 0
        self._exit_ip = None
        self._exit_latency_ms = 0
        self._tunnel_manager.clear_exited_process()
        if self._unexpected_exit_handler is not None:
            await self._unexpected_exit_handler()

    async def activate_job(self, node_id: str) -> dict[str, Any]:
        return (await self.activate(node_id)).model_dump(mode="json")

    async def disconnect(self) -> None:
        async with self._operation_lock:
            if self._settings_repository is not None:
                await self._settings_repository.set_connection_enabled(False)
            self._connection_enabled = False
            await self.disconnect_only()

    def set_connection_enabled(self, enabled: bool) -> None:
        self._connection_enabled = enabled

    async def disconnect_only(self) -> None:
        active_id = self._tunnel_manager.active_node_id
        await self._policy_router.cleanup()
        await self._tunnel_manager.disconnect()
        self._active_latency_ms = 0
        self._exit_ip = None
        self._exit_latency_ms = 0
        if active_id:
            logger.info("Proxy exit node disconnected: %s", active_id)

    def update_active_latency(self, latency_ms: int) -> None:
        self._active_latency_ms = latency_ms

    def update_health(self, *, exit_ip: str | None, latency_ms: int) -> None:
        self._exit_ip = exit_ip
        self._exit_latency_ms = latency_ms

    def status(self) -> GatewayStatus:
        if self._tunnel_manager.active_running:
            tunnel_status = TunnelStatus.CONNECTED
        elif self._tunnel_manager.active_node_id is not None:
            tunnel_status = TunnelStatus.FAILED
        else:
            tunnel_status = TunnelStatus.IDLE
        listener = self._proxy_gateway.server.listener
        return GatewayStatus(
            running=self._proxy_gateway.running,
            active_node_id=self._tunnel_manager.active_node_id,
            tunnel_status=tunnel_status,
            proxy_listener=listener,
            socks_listener=listener,
            http_listener=listener,
            last_error=self._last_error,
            active_latency_ms=self._active_latency_ms,
            exit_ip=self._exit_ip,
            exit_latency_ms=self._exit_latency_ms,
            connection_enabled=self._connection_enabled,
        )
