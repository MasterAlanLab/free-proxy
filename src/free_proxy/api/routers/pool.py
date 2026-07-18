from typing import Annotated

from fastapi import APIRouter, Depends

from free_proxy.api.dependencies import get_proxy_pool_service
from free_proxy.domain.models import PoolStatistics
from free_proxy.services.proxy_pool import ProxyPoolService

router = APIRouter(prefix="/pool", tags=["pool"])


@router.get("/statistics", response_model=PoolStatistics)
async def pool_statistics(
    pool: Annotated[ProxyPoolService, Depends(get_proxy_pool_service)],
) -> PoolStatistics:
    return await pool.statistics()
