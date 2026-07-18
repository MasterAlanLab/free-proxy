from __future__ import annotations

import asyncio
import logging
from collections.abc import Awaitable, Callable
from typing import Any

from free_proxy.config import Settings
from free_proxy.domain.exceptions import ResourceNotFoundError
from free_proxy.domain.models import ProbeResult, TunnelStartResult, utc_now
from free_proxy.infrastructure.database.repositories import (
    ProbeResultRepository,
    ProxyNodeRepository,
)
from free_proxy.infrastructure.network.latency import measure_node_latency
from free_proxy.infrastructure.network.tun import TunAllocator
from free_proxy.infrastructure.tunnel.openvpn import OpenVpnManager
from free_proxy.services.ipinfo import IpInfoService
from free_proxy.services.operations import NetworkOperationCoordinator

logger = logging.getLogger(__name__)

LatencyProbe = Callable[[str, int, float], Awaitable[int]]


class ProbeService:
    def __init__(
        self,
        settings: Settings,
        repository: ProxyNodeRepository,
        tunnel_manager: OpenVpnManager,
        tun_allocator: TunAllocator,
        latency_probe: LatencyProbe = measure_node_latency,
        ip_info: IpInfoService | None = None,
        history: ProbeResultRepository | None = None,
        coordinator: NetworkOperationCoordinator | None = None,
    ) -> None:
        self._settings = settings
        self._repository = repository
        self._tunnel_manager = tunnel_manager
        self._tun_allocator = tun_allocator
        self._latency_probe = latency_probe
        self._ip_info = ip_info
        self._history = history
        self._coordinator = coordinator
        self._semaphore = asyncio.Semaphore(settings.max_probe_concurrency)

    async def probe(self, node_id: str, *, enrich: bool = True) -> ProbeResult:
        logger.info("Starting node probe: %s", node_id)
        target = await self._repository.get_target(node_id)
        if target is None:
            raise ResourceNotFoundError(f"Proxy node not found: {node_id}")

        await self._repository.mark_probing(node_id)
        try:
            async with self._semaphore, self._tun_allocator.allocate() as device:
                latency_ms, tunnel = await asyncio.gather(
                    self._latency_probe(
                        target.remote_host,
                        target.remote_port,
                        target.source_ping_ms,
                    ),
                    self._tunnel_manager.probe(target.config_text, device),
                )
        except Exception as exc:
            logger.warning("Node probe failed for %s: %s", node_id, exc)
            tunnel = self._failure_result(exc)
            latency_ms = 0

        probed_at = utc_now()
        result = ProbeResult(
            node_id=node_id,
            available=tunnel.success,
            latency_ms=latency_ms,
            tunnel=tunnel,
            probed_at=probed_at,
        )
        await self._repository.update_probe_result(
            node_id=node_id,
            available=result.available,
            latency_ms=result.latency_ms,
            probed_at=probed_at,
        )
        if self._history is not None:
            await self._history.record(result)
        if enrich and result.available and self._ip_info is not None:
            try:
                await self._ip_info.enrich(node_id, target.ip_address)
            except Exception:
                logger.warning("IP information lookup failed for node %s", node_id)
        logger.log(
            logging.INFO if result.available else logging.WARNING,
            "Node probe completed: %s available=%s latency_ms=%d message=%s",
            node_id,
            result.available,
            result.latency_ms,
            result.tunnel.message,
        )
        return result

    async def probe_many(self, node_ids: list[str]) -> list[ProbeResult]:
        if self._coordinator is not None:
            async with self._coordinator.acquire("probe"):
                return await self._probe_many(node_ids)
        return await self._probe_many(node_ids)

    async def _probe_many(self, node_ids: list[str]) -> list[ProbeResult]:
        unique_ids = list(dict.fromkeys(node_ids))
        results = await asyncio.gather(
            *(self._probe_one(node_id) for node_id in unique_ids)
        )
        if self._ip_info is not None:
            successful: dict[str, str] = {}
            for result in results:
                if not result.available:
                    continue
                target = await self._repository.get_target(result.node_id)
                if target is not None:
                    successful[result.node_id] = target.ip_address
            if successful:
                try:
                    await self._ip_info.enrich_many(successful)
                except Exception:
                    pass
        return list(results)

    async def _probe_one(self, node_id: str) -> ProbeResult:
        try:
            return await self.probe(node_id, enrich=False)
        except Exception as exc:
            logger.exception("Unexpected isolated probe failure for %s", node_id)
            result = ProbeResult(
                node_id=node_id,
                available=False,
                latency_ms=0,
                tunnel=self._failure_result(exc),
            )
            await self._repository.record_probe_failure(result)
            if self._history is not None:
                await self._history.record(result)
            return result

    @staticmethod
    def _failure_result(exc: Exception) -> TunnelStartResult:
        from free_proxy.domain.enums import TunnelFailureCode, TunnelStatus

        return TunnelStartResult(
            success=False,
            status=TunnelStatus.FAILED,
            message=f"Probe failed: {type(exc).__name__}: {exc}",
            failure_code=TunnelFailureCode.UNKNOWN,
        )

    async def probe_job(self, node_id: str) -> dict[str, Any]:
        result = await self.probe(node_id)
        return result.model_dump(mode="json")

    async def probe_many_job(self, node_ids: list[str]) -> dict[str, Any]:
        results = await self.probe_many(node_ids)
        return {"nodes": [result.model_dump(mode="json") for result in results]}
