from __future__ import annotations

import httpx

from free_proxy.config import Settings
from free_proxy.domain.enums import IpType
from free_proxy.domain.models import IpInfo


class IpInfoClient:
    def __init__(self, settings: Settings, client: httpx.AsyncClient | None = None) -> None:
        self._settings = settings
        self._owns_client = client is None
        self._client = client or httpx.AsyncClient(
            timeout=settings.request_timeout_seconds,
            headers={"User-Agent": "free-proxy/0.1"},
        )

    async def lookup_many(self, ip_addresses: list[str]) -> dict[str, IpInfo]:
        if not ip_addresses:
            return {}
        results: dict[str, IpInfo] = {}
        for index in range(0, len(ip_addresses), 100):
            response = await self._client.post(
                self._settings.ip_info_api_url,
                json=ip_addresses[index : index + 100],
            )
            response.raise_for_status()
            payload = response.json()
            if not isinstance(payload, list):
                continue
            for item in payload:
                info = parse_ip_info(item)
                if info is not None:
                    results[info.ip_address] = info
        return results

    async def close(self) -> None:
        if self._owns_client:
            await self._client.aclose()


def parse_ip_info(item: object) -> IpInfo | None:
    if not isinstance(item, dict) or item.get("status") != "success":
        return None
    ip_address = str(item.get("query") or "")
    if not ip_address:
        return None
    ip_type = IpType.RESIDENTIAL
    if item.get("mobile"):
        ip_type = IpType.MOBILE
    elif item.get("hosting") or item.get("proxy"):
        ip_type = IpType.HOSTING

    quality = "normal"
    if item.get("proxy"):
        quality = "proxy"
    elif item.get("hosting"):
        quality = "datacenter"
    elif item.get("mobile"):
        quality = "mobile"

    location = " ".join(
        str(part)
        for part in (item.get("country"), item.get("regionName"), item.get("city"))
        if part
    )
    return IpInfo(
        ip_address=ip_address,
        owner=str(item.get("org") or item.get("isp") or ""),
        asn=str(item.get("as") or ""),
        as_name=str(item.get("asname") or ""),
        location=location,
        ip_type=ip_type,
        quality=quality,
    )
