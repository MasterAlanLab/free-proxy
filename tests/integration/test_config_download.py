import asyncio
from pathlib import Path

from fastapi.testclient import TestClient

from free_proxy.config import Settings
from free_proxy.domain.enums import TransportProtocol
from free_proxy.domain.models import DiscoveredNode
from free_proxy.infrastructure.database import Database, ProxyNodeRepository
from free_proxy.main import create_app


def test_openvpn_config_download_and_missing_node(tmp_path: Path) -> None:
    settings = Settings(
        data_dir=tmp_path,
        database_url=f"sqlite+aiosqlite:///{tmp_path / 'test.db'}",
        proxy_enabled=False,
        maintenance_enabled=False,
        admin_auth_enabled=False,
    )
    asyncio.run(insert_node(settings))

    with TestClient(create_app(settings)) as client:
        downloaded = client.get("/api/v1/proxies/jp-node/config")
        missing = client.get("/api/v1/proxies/missing/config")

    assert downloaded.status_code == 200
    assert downloaded.text == "client\nremote 198.51.100.10 443 tcp\n"
    assert downloaded.headers["content-type"].startswith("application/x-openvpn-profile")
    assert downloaded.headers["content-disposition"] == 'attachment; filename="jp-node.ovpn"'
    assert missing.status_code == 404


async def insert_node(settings: Settings) -> None:
    database = Database(settings)
    await database.initialize()
    try:
        repository = ProxyNodeRepository(database.session_factory)
        await repository.upsert_discovered(
            [
                DiscoveredNode(
                    id="jp-node",
                    provider="vpngate",
                    ip_address="198.51.100.10",
                    remote_host="198.51.100.10",
                    remote_port=443,
                    transport=TransportProtocol.TCP,
                    config_text="client\nremote 198.51.100.10 443 tcp\n",
                )
            ]
        )
    finally:
        await database.dispose()
