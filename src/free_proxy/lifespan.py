from __future__ import annotations

import logging
from collections.abc import AsyncIterator, Callable
from contextlib import AbstractAsyncContextManager, asynccontextmanager

from fastapi import FastAPI

from free_proxy.config import Settings
from free_proxy.infrastructure.database import (
    Database,
    JobRepository,
    ProbeResultRepository,
    ProxyNodeRepository,
    SettingsRepository,
)
from free_proxy.infrastructure.ipinfo import IpInfoClient
from free_proxy.infrastructure.network import PolicyRouter, TunAllocator
from free_proxy.infrastructure.network.proxy_health import SocksProxyHealthChecker
from free_proxy.infrastructure.tunnel import OpenVpnManager
from free_proxy.logging import configure_logging
from free_proxy.providers.vpngate import VpnGateProvider
from free_proxy.proxy import ProxyGateway
from free_proxy.services.active_latency import ActiveLatencyMonitor
from free_proxy.services.auto_switch import AutoSwitchService
from free_proxy.services.diagnostics import NetworkDiagnosticsService
from free_proxy.services.discovery import DiscoveryService
from free_proxy.services.gateway import GatewayService
from free_proxy.services.health import HealthMonitor, HealthService
from free_proxy.services.ipinfo import IpInfoService
from free_proxy.services.jobs import JobService
from free_proxy.services.maintenance import MaintenanceMonitor, MaintenanceService
from free_proxy.services.operations import NetworkOperationCoordinator
from free_proxy.services.probe import ProbeService
from free_proxy.services.proxy_pool import ProxyPoolService
from free_proxy.services.settings import SettingsService

logger = logging.getLogger(__name__)


def create_lifespan(
    settings: Settings,
) -> Callable[[FastAPI], AbstractAsyncContextManager[None]]:
    @asynccontextmanager
    async def lifespan(app: FastAPI) -> AsyncIterator[None]:
        settings.ensure_directories()
        log_store = configure_logging(settings)
        database = Database(settings)
        await database.initialize()

        repository = ProxyNodeRepository(database.session_factory)
        probe_history = ProbeResultRepository(database.session_factory)
        settings_repository = SettingsRepository(database.session_factory)
        provider = VpnGateProvider(settings)
        ip_info_client = IpInfoClient(settings)
        ip_info = IpInfoService(settings, ip_info_client, repository)
        diagnostics = NetworkDiagnosticsService(settings)
        preflight = await diagnostics.startup_preflight()
        for check in preflight.checks:
            if not check.ok:
                logger.warning("Startup preflight failed: %s: %s", check.name, check.detail)
        discovery = DiscoveryService(provider, repository, diagnostics)
        jobs = JobService(JobRepository(database.session_factory))
        coordinator = NetworkOperationCoordinator()
        await jobs.initialize()
        tunnel_manager = OpenVpnManager(settings)
        policy_router = PolicyRouter(settings)
        proxy_gateway = ProxyGateway(settings)
        proxy_pool = ProxyPoolService(repository, settings_repository)
        gateway = GatewayService(
            settings,
            repository,
            tunnel_manager,
            policy_router,
            proxy_gateway,
            proxy_pool,
            settings_repository,
            coordinator,
        )
        auto_switch = AutoSwitchService(
            settings,
            repository,
            settings_repository,
            proxy_pool,
            gateway,
        )
        gateway.set_unexpected_exit_handler(auto_switch.handle_unexpected_exit)
        health = HealthService(
            settings,
            SocksProxyHealthChecker(settings),
            repository,
            settings_repository,
            gateway,
            auto_switch,
        )
        settings_service = SettingsService(
            repository,
            settings_repository,
            proxy_pool,
            gateway,
            auto_switch,
        )
        probe = ProbeService(
            settings,
            repository,
            tunnel_manager,
            TunAllocator(settings.test_tun_start, settings.test_tun_end),
            ip_info=ip_info,
            history=probe_history,
            coordinator=coordinator,
        )
        maintenance = MaintenanceService(
            settings,
            repository,
            settings_repository,
            discovery,
            probe,
            proxy_pool,
            gateway,
            auto_switch,
            coordinator,
        )
        maintenance_monitor = MaintenanceMonitor(settings, maintenance, gateway)
        active_latency_monitor = ActiveLatencyMonitor(settings, repository, gateway)
        health_monitor = HealthMonitor(settings, health, gateway)

        app.state.settings = settings
        app.state.log_store = log_store
        app.state.database = database
        app.state.node_repository = repository
        app.state.settings_repository = settings_repository
        app.state.probe_history_repository = probe_history
        app.state.provider = provider
        app.state.discovery_service = discovery
        app.state.diagnostics_service = diagnostics
        app.state.job_service = jobs
        app.state.tunnel_manager = tunnel_manager
        app.state.gateway_service = gateway
        app.state.probe_service = probe
        app.state.proxy_pool_service = proxy_pool
        app.state.auto_switch_service = auto_switch
        app.state.health_service = health
        app.state.settings_service = settings_service
        app.state.maintenance_service = maintenance
        app.state.network_operation_coordinator = coordinator
        app.state.maintenance_monitor = maintenance_monitor
        app.state.active_latency_monitor = active_latency_monitor
        app.state.health_monitor = health_monitor

        try:
            await tunnel_manager.cleanup_stale_processes()
            await policy_router.cleanup()
            try:
                await gateway.start()
            except OSError as exc:
                logger.error("Proxy gateway is degraded and did not start: %s", exc)
            health_monitor.start()
            active_latency_monitor.start()
            if settings.maintenance_enabled:
                maintenance_monitor.start()
            yield
        finally:
            await maintenance_monitor.stop()
            await active_latency_monitor.stop()
            await health_monitor.stop()
            await jobs.shutdown()
            await gateway.shutdown()
            await ip_info_client.close()
            await provider.close()
            await database.dispose()

    return lifespan
