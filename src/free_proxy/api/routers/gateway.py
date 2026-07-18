from typing import Annotated

from fastapi import APIRouter, Depends, status

from free_proxy.api.dependencies import (
    get_auto_switch_service,
    get_gateway_service,
    get_health_service,
    get_job_service,
)
from free_proxy.domain.models import GatewayStatus, JobRead, ProxyHealthResult
from free_proxy.services.auto_switch import AutoSwitchService
from free_proxy.services.gateway import GatewayService
from free_proxy.services.health import HealthService
from free_proxy.services.jobs import JobService

router = APIRouter(prefix="/gateway", tags=["gateway"])


@router.get("/status", response_model=GatewayStatus)
async def gateway_status(
    gateway: Annotated[GatewayService, Depends(get_gateway_service)],
) -> GatewayStatus:
    return gateway.status()


@router.delete("/current", status_code=status.HTTP_204_NO_CONTENT)
async def disconnect_gateway(
    gateway: Annotated[GatewayService, Depends(get_gateway_service)],
) -> None:
    await gateway.disconnect()


@router.post("/check", response_model=ProxyHealthResult)
async def check_gateway(
    health: Annotated[HealthService, Depends(get_health_service)],
) -> ProxyHealthResult:
    return await health.check(recover=False)


@router.post("/rotate", response_model=JobRead, status_code=status.HTTP_202_ACCEPTED)
async def rotate_gateway(
    auto_switch: Annotated[AutoSwitchService, Depends(get_auto_switch_service)],
    jobs: Annotated[JobService, Depends(get_job_service)],
) -> JobRead:
    async def rotate() -> dict[str, object]:
        result = await auto_switch.switch()
        return {
            "connected": bool(result and result.success),
            "tunnel": result.model_dump(mode="json") if result is not None else None,
        }

    return await jobs.submit("rotate-gateway", rotate)
