from __future__ import annotations

import asyncio
import logging

from free_proxy.config import Settings
from free_proxy.infrastructure.database.repositories import ProxyNodeRepository
from free_proxy.infrastructure.network.latency import measure_node_latency
from free_proxy.services.gateway import GatewayService
from free_proxy.services.health import MonitorState

logger = logging.getLogger(__name__)


class ActiveLatencyMonitor:
    def __init__(
        self,
        settings: Settings,
        nodes: ProxyNodeRepository,
        gateway: GatewayService,
    ) -> None:
        self._settings = settings
        self._nodes = nodes
        self._gateway = gateway
        self._task: asyncio.Task[None] | None = None
        self.state = MonitorState()

    def start(self) -> None:
        if self._task is None:
            self._task = asyncio.create_task(self._run(), name="active-node-latency")

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
            try:
                active_id = self._gateway.status().active_node_id
                if active_id:
                    target = await self._nodes.get_target(active_id)
                    if target is not None:
                        latency = await measure_node_latency(
                            target.remote_host, target.remote_port, target.source_ping_ms
                        )
                        self._gateway.update_active_latency(latency)
                self.state.heartbeat(success=True)
            except asyncio.CancelledError:
                raise
            except Exception as exc:
                self.state.heartbeat(success=False, error=f"{type(exc).__name__}: {exc}")
                logger.exception("Active latency monitor cycle failed")
            finally:
                await asyncio.sleep(self._settings.active_ping_interval_seconds)
