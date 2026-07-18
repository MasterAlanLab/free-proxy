from __future__ import annotations

import asyncio
import logging
from typing import Any

from free_proxy.config import Settings
from free_proxy.domain.enums import NodeStatus, ProxyPolicyMode
from free_proxy.domain.models import MaintenanceResult, ProxyNodeRead
from free_proxy.infrastructure.database.repositories import (
    ProxyNodeRepository,
    SettingsRepository,
)
from free_proxy.services.auto_switch import AutoSwitchService
from free_proxy.services.discovery import DiscoveryService
from free_proxy.services.gateway import GatewayService
from free_proxy.services.health import MonitorState
from free_proxy.services.operations import NetworkOperationCoordinator
from free_proxy.services.probe import ProbeService
from free_proxy.services.proxy_pool import ProxyPoolService

logger = logging.getLogger(__name__)


class MaintenanceService:
    def __init__(
        self,
        app_settings: Settings,
        nodes: ProxyNodeRepository,
        settings: SettingsRepository,
        discovery: DiscoveryService,
        probe: ProbeService,
        pool: ProxyPoolService,
        gateway: GatewayService,
        auto_switch: AutoSwitchService,
        coordinator: NetworkOperationCoordinator | None = None,
    ) -> None:
        self._app_settings = app_settings
        self._nodes = nodes
        self._settings = settings
        self._discovery = discovery
        self._probe = probe
        self._pool = pool
        self._gateway = gateway
        self._auto_switch = auto_switch
        self._coordinator = coordinator
        self._lock = asyncio.Lock()

    @property
    def running(self) -> bool:
        return self._lock.locked()

    async def run(self) -> MaintenanceResult:
        if self._coordinator is not None:
            async with self._coordinator.acquire("maintenance"):
                return await self._run()
        return await self._run()

    async def _run(self) -> MaintenanceResult:
        async with self._lock:
            logger.info("Starting periodic proxy node maintenance")
            await self._nodes.clear_expired_blacklist()
            discovery_result = await self._discovery.discover()
            purge_stale = getattr(self._nodes, "purge_stale_nodes", None)
            if purge_stale is not None:
                await purge_stale(self._app_settings.stale_node_grace_seconds)
            settings = await self._settings.get()
            initial_tested: set[str] = set()
            probed_count = 0

            if (
                settings.connection_enabled
                and settings.routing_mode is not ProxyPolicyMode.FIXED
                and not self._gateway.status().active_node_id
            ):
                fast_candidates = await self._candidate_nodes()
                fast_candidates = self._pool.apply_filters(
                    fast_candidates,
                    settings,
                    include_unknown_ip_type=True,
                )
                fast_candidates.sort(key=probe_priority_key)
                fast_ids = [
                    node.id
                    for node in fast_candidates[: self._app_settings.initial_connect_test_limit]
                ]
                if fast_ids:
                    initial_tested.update(fast_ids)
                    results = await self._probe.probe_many(fast_ids)
                    probed_count += len(results)
                    if any(result.available for result in results):
                        await self._auto_switch.switch()
                        if self._gateway.status().active_node_id:
                            result = await self._result(
                                discovery_result.discovered, probed_count
                            )
                            logger.info("Maintenance completed after fast connection: %s", result)
                            return result

            remaining = [
                node.id
                for node in await self._candidate_nodes(include_unavailable=True)
                if node.id not in initial_tested
                and node.id != self._gateway.status().active_node_id
            ]
            if remaining:
                results = await self._probe.probe_many(remaining)
                probed_count += len(results)

            if settings.connection_enabled and not self._gateway.status().active_node_id:
                if settings.routing_mode is ProxyPolicyMode.FIXED and settings.fixed_node_id:
                    await self._gateway.activate(settings.fixed_node_id)
                else:
                    await self._auto_switch.switch()
            result = await self._result(discovery_result.discovered, probed_count)
            logger.info("Periodic proxy node maintenance completed: %s", result)
            return result

    async def run_job(self) -> dict[str, Any]:
        return (await self.run()).model_dump(mode="json")

    async def _candidate_nodes(self, *, include_unavailable: bool = False) -> list[ProxyNodeRead]:
        try:
            nodes = await self._nodes.list_nodes(limit=1000, current_only=True)
        except TypeError:
            nodes = await self._nodes.list_nodes(limit=1000)
        allowed_statuses = {NodeStatus.DISCOVERED, NodeStatus.READY}
        if include_unavailable:
            allowed_statuses.add(NodeStatus.UNAVAILABLE)
        return [node for node in nodes if node.status in allowed_statuses]

    async def _result(self, discovered: int, probed: int) -> MaintenanceResult:
        available = await self._nodes.count_nodes(status=NodeStatus.READY)
        return MaintenanceResult(
            discovered=discovered,
            probed=probed,
            available=available,
            connected_node_id=self._gateway.status().active_node_id,
        )


class MaintenanceMonitor:
    def __init__(
        self,
        settings: Settings,
        maintenance: MaintenanceService,
        gateway: GatewayService,
    ) -> None:
        self._settings = settings
        self._maintenance = maintenance
        self._gateway = gateway
        self._task: asyncio.Task[None] | None = None
        self.state = MonitorState()

    def start(self) -> None:
        if self._task is None:
            self._task = asyncio.create_task(self._run(), name="node-maintenance-monitor")

    @property
    def running(self) -> bool:
        return self._task is not None and not self._task.done()

    async def stop(self) -> None:
        if self._task is None:
            return
        self._task.cancel()
        try:
            await self._task
        except asyncio.CancelledError:
            pass
        self._task = None

    async def _run(self) -> None:
        while True:
            success = False
            try:
                await self._maintenance.run()
                success = True
                self.state.heartbeat(success=True)
            except asyncio.CancelledError:
                raise
            except Exception as exc:
                self.state.heartbeat(success=False, error=f"{type(exc).__name__}: {exc}")
                logger.exception("Periodic proxy node maintenance failed: %s", exc)
            if not success and not self._gateway.status().active_node_id:
                delay = self._settings.disconnected_retry_seconds
            else:
                delay = self._settings.maintenance_interval_seconds
            await asyncio.sleep(delay)


def probe_priority_key(node: ProxyNodeRead) -> tuple[int, int, int, int]:
    return (
        node.source_ping_ms or 999_999,
        -node.source_score,
        -node.source_speed_bps,
        node.source_sessions,
    )
