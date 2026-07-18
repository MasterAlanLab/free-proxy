from dataclasses import asdict
from datetime import UTC, datetime
from typing import Annotated, Any

from fastapi import APIRouter, Depends, Request

from free_proxy import __version__
from free_proxy.api.dependencies import get_gateway_service, get_node_repository, get_settings
from free_proxy.config import Settings
from free_proxy.domain.models import DnsRepairResult, SystemDiagnostics
from free_proxy.infrastructure.database.repositories import ProxyNodeRepository
from free_proxy.services.diagnostics import NetworkDiagnosticsService
from free_proxy.services.gateway import GatewayService

router = APIRouter(prefix="/system", tags=["system"])


@router.get("/status")
async def system_status(
    request: Request,
    settings: Annotated[Settings, Depends(get_settings)],
    repository: Annotated[ProxyNodeRepository, Depends(get_node_repository)],
    gateway: Annotated[GatewayService, Depends(get_gateway_service)],
) -> dict[str, Any]:
    gateway_status = gateway.status()
    monitor_specs = {
        "maintenance": (
            request.app.state.maintenance_monitor,
            settings.maintenance_interval_seconds + settings.disconnected_retry_seconds,
        ),
        "active_latency": (
            request.app.state.active_latency_monitor,
            settings.active_ping_interval_seconds * 3,
        ),
        "health": (request.app.state.health_monitor, settings.health_check_interval_seconds * 3),
    }
    monitors = {
        name: monitor_payload(monitor, max_age)
        for name, (monitor, max_age) in monitor_specs.items()
    }
    operation = getattr(request.app.state, "network_operation_coordinator", None)
    degraded = any(item["status"] == "degraded" for item in monitors.values())
    return {
        "name": settings.app_name,
        "version": __version__,
        "environment": settings.environment,
        "status": "degraded" if degraded else "running",
        "nodes": await repository.count_nodes(),
        "gateway_running": gateway_status.running,
        "active_node_id": gateway_status.active_node_id,
        "listeners": {
            "web": f"{settings.web_host}:{settings.web_port}",
            "socks5": gateway_status.socks_listener,
            "http": gateway_status.http_listener,
        },
        "monitors": {name: item["running"] for name, item in monitors.items()},
        "monitor_details": monitors,
        "network_operation": asdict(operation.snapshot) if operation is not None else None,
    }


def monitor_payload(monitor: Any, max_age_seconds: float) -> dict[str, Any]:
    state = monitor.state
    heartbeat = state.last_heartbeat_at
    stale = heartbeat is not None and (
        datetime.now(UTC) - heartbeat
    ).total_seconds() > max_age_seconds
    healthy = monitor.running and heartbeat is not None and not stale
    return {
        "running": monitor.running,
        "status": "healthy" if healthy else "degraded",
        **state.as_dict(),
    }


@router.get("/diagnostics", response_model=SystemDiagnostics)
async def system_diagnostics(request: Request) -> SystemDiagnostics:
    diagnostics: NetworkDiagnosticsService = request.app.state.diagnostics_service
    return await diagnostics.diagnose()


@router.post("/dns/repair", response_model=DnsRepairResult)
async def repair_dns(request: Request) -> DnsRepairResult:
    diagnostics: NetworkDiagnosticsService = request.app.state.diagnostics_service
    return await diagnostics.repair_dns()
