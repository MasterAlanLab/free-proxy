from pathlib import Path

import httpx
import pytest

from free_proxy.config import Settings
from free_proxy.domain.enums import IpType
from free_proxy.infrastructure.ipinfo import IpInfoClient


@pytest.mark.asyncio
async def test_ip_info_client_uses_existing_classification_rules(tmp_path: Path) -> None:
    async def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            200,
            json=[
                {
                    "status": "success",
                    "query": "198.51.100.1",
                    "country": "Japan",
                    "regionName": "Tokyo",
                    "city": "Chiyoda",
                    "isp": "Example ISP",
                    "org": "Example Org",
                    "as": "AS64500 Example",
                    "asname": "EXAMPLE-AS",
                    "proxy": False,
                    "hosting": False,
                    "mobile": False,
                },
                {
                    "status": "success",
                    "query": "198.51.100.2",
                    "hosting": False,
                    "proxy": False,
                    "mobile": True,
                },
                {
                    "status": "success",
                    "query": "198.51.100.3",
                    "hosting": True,
                    "proxy": False,
                    "mobile": False,
                },
            ],
        )

    http_client = httpx.AsyncClient(transport=httpx.MockTransport(handler))
    client = IpInfoClient(Settings(data_dir=tmp_path), client=http_client)
    try:
        result = await client.lookup_many(["198.51.100.1", "198.51.100.2", "198.51.100.3"])
    finally:
        await http_client.aclose()

    assert result["198.51.100.1"].ip_type is IpType.RESIDENTIAL
    assert result["198.51.100.1"].location == "Japan Tokyo Chiyoda"
    assert result["198.51.100.2"].ip_type is IpType.MOBILE
    assert result["198.51.100.2"].quality == "mobile"
    assert result["198.51.100.3"].ip_type is IpType.HOSTING
    assert result["198.51.100.3"].quality == "datacenter"
