from typing import Annotated

from fastapi import APIRouter, Depends, HTTPException, status

from free_proxy.api.dependencies import get_settings_service
from free_proxy.domain.enums import ProxyPolicyMode
from free_proxy.domain.models import ProxySettings, ProxySettingsUpdate
from free_proxy.services.settings import SettingsService

router = APIRouter(prefix="/settings", tags=["settings"])


@router.get("", response_model=ProxySettings)
async def get_proxy_settings(
    service: Annotated[SettingsService, Depends(get_settings_service)],
) -> ProxySettings:
    return await service.get()


@router.put("", response_model=ProxySettings)
async def update_proxy_settings(
    payload: ProxySettingsUpdate,
    service: Annotated[SettingsService, Depends(get_settings_service)],
) -> ProxySettings:
    if payload.routing_mode is ProxyPolicyMode.COUNTRY and not payload.force_country:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="force_country is required for country mode",
        )
    if payload.routing_mode is ProxyPolicyMode.FIXED and not payload.fixed_node_id:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="fixed_node_id is required for fixed mode",
        )
    return await service.update(payload)
