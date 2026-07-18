from typing import Annotated

from fastapi import APIRouter, Depends, HTTPException, Query, status
from fastapi.responses import Response

from free_proxy.api.dependencies import (
    get_discovery_service,
    get_gateway_service,
    get_job_service,
    get_maintenance_service,
    get_node_repository,
    get_operation_coordinator,
    get_probe_history_repository,
    get_probe_service,
    get_settings,
    get_settings_service,
)
from free_proxy.config import Settings
from free_proxy.domain.enums import IpType, NodeStatus
from free_proxy.domain.models import JobRead, ProbeHistoryRead, ProbeManyRequest, ProxyNodePage
from free_proxy.infrastructure.database.repositories import (
    ProbeResultRepository,
    ProxyNodeRepository,
)
from free_proxy.services.discovery import DiscoveryService
from free_proxy.services.gateway import GatewayService
from free_proxy.services.jobs import JobService
from free_proxy.services.maintenance import MaintenanceService
from free_proxy.services.operations import NetworkOperationCoordinator
from free_proxy.services.probe import ProbeService
from free_proxy.services.settings import SettingsService

router = APIRouter(prefix="/proxies", tags=["proxies"])


@router.get("", response_model=ProxyNodePage)
async def list_proxies(
    repository: Annotated[ProxyNodeRepository, Depends(get_node_repository)],
    limit: Annotated[int, Query(ge=1, le=500)] = 100,
    offset: Annotated[int, Query(ge=0)] = 0,
    ip_type: IpType | None = None,
    node_status: Annotated[NodeStatus | None, Query(alias="status")] = None,
    country: str | None = None,
    search: str | None = None,
    include_history: bool = False,
) -> ProxyNodePage:
    items = await repository.list_nodes(
        limit=limit,
        offset=offset,
        ip_type=ip_type,
        status=node_status,
        country=country,
        search=search,
        current_only=not include_history,
    )
    total = await repository.count_nodes(
        ip_type=ip_type,
        status=node_status,
        country=country,
        search=search,
        current_only=not include_history,
    )
    return ProxyNodePage(items=items, total=total, limit=limit, offset=offset)


@router.post("/discover", response_model=JobRead, status_code=status.HTTP_202_ACCEPTED)
async def discover_proxies(
    discovery: Annotated[DiscoveryService, Depends(get_discovery_service)],
    jobs: Annotated[JobService, Depends(get_job_service)],
) -> JobRead:
    return await jobs.submit("discover-proxies", discovery.discover_job)


@router.post("/refresh", response_model=JobRead, status_code=status.HTTP_202_ACCEPTED)
async def refresh_proxies(
    maintenance: Annotated[MaintenanceService, Depends(get_maintenance_service)],
    jobs: Annotated[JobService, Depends(get_job_service)],
    coordinator: Annotated[NetworkOperationCoordinator, Depends(get_operation_coordinator)],
) -> JobRead:
    if coordinator.snapshot.operation is not None:
        raise HTTPException(
            status_code=status.HTTP_409_CONFLICT,
            detail="Another network operation is running",
        )
    return await jobs.submit("refresh-proxies", maintenance.run_job)


@router.post("/probe", response_model=JobRead, status_code=status.HTTP_202_ACCEPTED)
async def probe_multiple_proxies(
    payload: ProbeManyRequest,
    probe: Annotated[ProbeService, Depends(get_probe_service)],
    jobs: Annotated[JobService, Depends(get_job_service)],
    settings: Annotated[Settings, Depends(get_settings)],
    coordinator: Annotated[NetworkOperationCoordinator, Depends(get_operation_coordinator)],
) -> JobRead:
    if coordinator.snapshot.operation is not None:
        raise HTTPException(
            status_code=status.HTTP_409_CONFLICT,
            detail="Another network operation is running",
        )
    node_ids = list(dict.fromkeys(node_id.strip() for node_id in payload.ids if node_id.strip()))
    if len(node_ids) > settings.manual_test_node_limit:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"At most {settings.manual_test_node_limit} nodes can be tested at once",
        )
    return await jobs.submit(
        "probe-proxies",
        lambda: probe.probe_many_job(node_ids),
    )


@router.post("/{node_id}/probe", response_model=JobRead, status_code=status.HTTP_202_ACCEPTED)
async def probe_proxy(
    node_id: str,
    probe: Annotated[ProbeService, Depends(get_probe_service)],
    jobs: Annotated[JobService, Depends(get_job_service)],
) -> JobRead:
    return await jobs.submit(
        "probe-proxy",
        lambda: probe.probe_job(node_id),
    )


@router.get("/{node_id}/probes", response_model=list[ProbeHistoryRead])
async def probe_history(
    node_id: str,
    history: Annotated[ProbeResultRepository, Depends(get_probe_history_repository)],
    limit: Annotated[int, Query(ge=1, le=500)] = 100,
) -> list[ProbeHistoryRead]:
    return await history.list_for_node(node_id, limit=limit)


@router.post(
    "/{node_id}/activate",
    response_model=JobRead,
    status_code=status.HTTP_202_ACCEPTED,
)
async def activate_proxy(
    node_id: str,
    gateway: Annotated[GatewayService, Depends(get_gateway_service)],
    jobs: Annotated[JobService, Depends(get_job_service)],
) -> JobRead:
    return await jobs.submit(
        "activate-proxy",
        lambda: gateway.activate_job(node_id),
    )


@router.post("/{node_id}/favorite")
async def toggle_favorite(
    node_id: str,
    settings: Annotated[SettingsService, Depends(get_settings_service)],
) -> dict[str, object]:
    updated = await settings.toggle_favorite(node_id)
    return {"ok": True, "favorite_node_ids": updated.favorite_node_ids}


@router.get("/{node_id}/config", response_class=Response)
async def download_proxy_config(
    node_id: str,
    repository: Annotated[ProxyNodeRepository, Depends(get_node_repository)],
) -> Response:
    target = await repository.get_target(node_id)
    if target is None:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="Proxy node not found")
    return Response(
        target.config_text,
        media_type="application/x-openvpn-profile",
        headers={"Content-Disposition": f'attachment; filename="{node_id}.ovpn"'},
    )
