from datetime import UTC, datetime, timedelta
from pathlib import Path

import pytest
from sqlalchemy import update

from free_proxy.config import Settings
from free_proxy.domain.enums import IpType, NodeStatus, TransportProtocol
from free_proxy.domain.models import DiscoveredNode, IpInfo
from free_proxy.infrastructure.database import Database, ProxyNodeRepository
from free_proxy.infrastructure.database.models import BlacklistRecord


@pytest.mark.asyncio
async def test_blacklist_expiry_and_ip_info_cache_lifecycle(tmp_path: Path) -> None:
    settings = Settings(
        data_dir=tmp_path,
        database_url=f"sqlite+aiosqlite:///{tmp_path / 'test.db'}",
    )
    database = Database(settings)
    await database.initialize()
    repository = ProxyNodeRepository(database.session_factory)
    await repository.upsert_discovered(
        [
            DiscoveredNode(
                id="node-1",
                provider="vpngate",
                ip_address="198.51.100.10",
                remote_host="198.51.100.10",
                remote_port=443,
                transport=TransportProtocol.TCP,
                config_text="client\n",
            )
        ]
    )

    try:
        assert await repository.stale_ip_info_node_ids(["node-1"], 3600) == {"node-1"}
        await repository.update_ip_info(
            "node-1",
            IpInfo(
                ip_address="198.51.100.10",
                owner="Example ISP",
                ip_type=IpType.RESIDENTIAL,
            ),
        )
        assert await repository.stale_ip_info_node_ids(["node-1"], 3600) == set()

        await repository.blacklist("node-1", "failed", 3600)
        node = await repository.get_node("node-1")
        assert node is not None
        assert node.status is NodeStatus.COOLDOWN
        assert await repository.active_blacklist_ids() == {"node-1"}

        async with database.session_factory() as session:
            await session.execute(
                update(BlacklistRecord).values(expires_at=datetime.now(UTC) - timedelta(seconds=1))
            )
            await session.commit()
        await repository.clear_expired_blacklist()
        recovered = await repository.get_node("node-1")
        assert recovered is not None
        assert recovered.status is NodeStatus.UNAVAILABLE
        assert recovered.cooldown_until is None
        assert await repository.active_blacklist_ids() == set()
    finally:
        await database.dispose()
