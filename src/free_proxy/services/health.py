from __future__ import annotations

import asyncio
import logging
from dataclasses import dataclass
from datetime import UTC, datetime

from free_proxy.config import Settings
from free_proxy.domain.enums import ProxyPolicyMode
from free_proxy.domain.models import ProxyHealthResult
from free_proxy.infrastructure.database.repositories import (
    ProxyNodeRepository,
    SettingsRepository,
)
from free_proxy.infrastructure.network.proxy_health import SocksProxyHealthChecker
from free_proxy.services.auto_switch import AutoSwitchService
from free_proxy.services.gateway import GatewayService

logger = logging.getLogger(__name__)


@dataclass(slots=True)
class MonitorState:
    last_heartbeat_at: datetime | None = None
    last_success_at: datetime | None = None
    last_error: str | None = None
    consecutive_failures: int = 0

    def heartbeat(self, *, success: bool, error: str | None = None) -> None:
        now = datetime.now(UTC)
        self.last_heartbeat_at = now
        if success:
            self.last_success_at = now
            self.last_error = None
            self.consecutive_failures = 0
        else:
            self.last_error = error
            self.consecutive_failures += 1

    def as_dict(self) -> dict[str, object]:
        return {
            "last_heartbeat_at": (
                self.last_heartbeat_at.isoformat() if self.last_heartbeat_at else None
            ),
            "last_success_at": (
                self.last_success_at.isoformat() if self.last_success_at else None
            ),
            "last_error": self.last_error,
            "consecutive_failures": self.consecutive_failures,
        }


class HealthService:
    def __init__(
        self,
        app_settings: Settings,
        checker: SocksProxyHealthChecker,
        nodes: ProxyNodeRepository,
        settings: SettingsRepository,
        gateway: GatewayService,
        auto_switch: AutoSwitchService,
    ) -> None:
        self._app_settings = app_settings
        self._checker = checker
        self._nodes = nodes
        self._settings = settings
        self._gateway = gateway
        self._auto_switch = auto_switch

    async def check(self, *, recover: bool = True) -> ProxyHealthResult:
        try:
            result = await self._checker.check()
        except Exception as exc:
            logger.exception("Unexpected proxy health check failure")
            result = ProxyHealthResult(ok=False, error=f"{type(exc).__name__}: {exc}")
        self._gateway.update_health(
            exit_ip=result.exit_ip if result.ok else None,
            latency_ms=result.latency_ms if result.ok else 0,
        )
        active_id = self._gateway.status().active_node_id
        if result.ok or not active_id or not recover:
            logger.log(
                logging.INFO if result.ok else logging.WARNING,
                "Proxy health check completed: ok=%s exit_ip=%s error=%s",
                result.ok,
                result.exit_ip,
                result.error,
            )
            return result

        settings = await self._settings.get()
        if settings.connection_enabled:
            if settings.routing_mode is ProxyPolicyMode.FIXED:
                logger.warning("Fixed node %s failed health check; retrying in place", active_id)
                await self._gateway.activate(active_id)
            else:
                await self._nodes.blacklist(
                    active_id,
                    result.error or "Proxy health check failed",
                    self._app_settings.invalid_backoff_seconds,
                )
                logger.warning("Active node %s entered cooldown after health failure", active_id)
                await self._auto_switch.switch()
        return result


class HealthMonitor:
    def __init__(
        self,
        settings: Settings,
        health: HealthService,
        gateway: GatewayService,
    ) -> None:
        self._settings = settings
        self._health = health
        self._gateway = gateway
        self._task: asyncio.Task[None] | None = None
        self.state = MonitorState()

    def start(self) -> None:
        if self._task is None:
            self._task = asyncio.create_task(self._run(), name="proxy-health-monitor")

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
        await asyncio.sleep(self._settings.health_check_interval_seconds)
        while True:
            try:
                if self._gateway_active():
                    await self._health.check(recover=True)
                self.state.heartbeat(success=True)
            except asyncio.CancelledError:
                raise
            except Exception as exc:
                self.state.heartbeat(success=False, error=f"{type(exc).__name__}: {exc}")
                logger.exception("Health monitor cycle failed")
            finally:
                await asyncio.sleep(self._settings.health_check_interval_seconds)

    def _gateway_active(self) -> bool:
        return self._gateway.status().active_node_id is not None
