from fastapi import Request

from free_proxy.config import Settings
from free_proxy.infrastructure.database.repositories import (
    ProbeResultRepository,
    ProxyNodeRepository,
    SettingsRepository,
)
from free_proxy.services.auto_switch import AutoSwitchService
from free_proxy.services.discovery import DiscoveryService
from free_proxy.services.gateway import GatewayService
from free_proxy.services.health import HealthService
from free_proxy.services.jobs import JobService
from free_proxy.services.maintenance import MaintenanceService
from free_proxy.services.operations import NetworkOperationCoordinator
from free_proxy.services.probe import ProbeService
from free_proxy.services.proxy_pool import ProxyPoolService
from free_proxy.services.settings import SettingsService


def get_settings(request: Request) -> Settings:
    return request.app.state.settings


def get_node_repository(request: Request) -> ProxyNodeRepository:
    return request.app.state.node_repository


def get_probe_history_repository(request: Request) -> ProbeResultRepository:
    return request.app.state.probe_history_repository


def get_discovery_service(request: Request) -> DiscoveryService:
    return request.app.state.discovery_service


def get_job_service(request: Request) -> JobService:
    return request.app.state.job_service


def get_probe_service(request: Request) -> ProbeService:
    return request.app.state.probe_service


def get_gateway_service(request: Request) -> GatewayService:
    return request.app.state.gateway_service


def get_auto_switch_service(request: Request) -> AutoSwitchService:
    return request.app.state.auto_switch_service


def get_settings_repository(request: Request) -> SettingsRepository:
    return request.app.state.settings_repository


def get_proxy_pool_service(request: Request) -> ProxyPoolService:
    return request.app.state.proxy_pool_service


def get_health_service(request: Request) -> HealthService:
    return request.app.state.health_service


def get_settings_service(request: Request) -> SettingsService:
    return request.app.state.settings_service


def get_maintenance_service(request: Request) -> MaintenanceService:
    return request.app.state.maintenance_service


def get_operation_coordinator(request: Request) -> NetworkOperationCoordinator:
    return request.app.state.network_operation_coordinator
