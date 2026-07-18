import asyncio
from pathlib import Path

import pytest

from free_proxy.config import Settings
from free_proxy.domain.enums import NodeStatus, TransportProtocol, TunnelStatus
from free_proxy.domain.models import DiscoveredNode, TunnelStartResult
from free_proxy.infrastructure.database import (
    Database,
    ProbeResultRepository,
    ProxyNodeRepository,
)
from free_proxy.infrastructure.network.tun import TunAllocator
from free_proxy.services.probe import ProbeService


class SuccessfulTunnelManager:
    async def probe(self, config_text: str, device: str) -> TunnelStartResult:
        assert config_text == "client\n"
        assert device == "tun2"
        return TunnelStartResult(
            success=True,
            status=TunnelStatus.CONNECTED,
            message="connected",
        )


async def fixed_latency(host: str, port: int, timeout_seconds: float) -> int:
    assert host == "198.51.100.10"
    assert port == 1194
    return 42


class MixedTunnelManager:
    def __init__(self) -> None:
        self.active = 0
        self.max_active = 0

    async def probe(self, config_text: str, device: str) -> TunnelStartResult:
        self.active += 1
        self.max_active = max(self.max_active, self.active)
        await asyncio.sleep(0.01)
        self.active -= 1
        success = "success" in config_text
        return TunnelStartResult(
            success=success,
            status=TunnelStatus.CONNECTED if success else TunnelStatus.FAILED,
            message="connected" if success else "failed",
        )


async def any_latency(host: str, port: int, timeout_seconds: float) -> int:
    return 25


@pytest.mark.asyncio
async def test_probe_service_updates_node_status(tmp_path: Path) -> None:
    settings = Settings(
        data_dir=tmp_path,
        database_url=f"sqlite+aiosqlite:///{tmp_path / 'test.db'}",
    )
    database = Database(settings)
    await database.initialize()
    repository = ProxyNodeRepository(database.session_factory)
    history = ProbeResultRepository(database.session_factory)
    await repository.upsert_discovered(
        [
            DiscoveredNode(
                id="jp-node",
                provider="vpngate",
                ip_address="198.51.100.10",
                remote_host="198.51.100.10",
                remote_port=1194,
                transport=TransportProtocol.UDP,
                config_text="client\n",
            )
        ]
    )
    service = ProbeService(
        settings,
        repository,
        SuccessfulTunnelManager(),  # type: ignore[arg-type]
        TunAllocator(2, 2),
        latency_probe=fixed_latency,
        history=history,
    )

    try:
        result = await service.probe("jp-node")
        nodes = await repository.list_nodes()
        probe_history = await history.list_for_node("jp-node")
    finally:
        await database.dispose()

    assert result.available is True
    assert result.latency_ms == 42
    assert nodes[0].status is NodeStatus.READY
    assert nodes[0].success_count == 1
    assert len(probe_history) == 1
    assert probe_history[0].result["tunnel"]["message"] == "connected"


@pytest.mark.asyncio
async def test_probe_many_limits_concurrency_and_keeps_partial_failures(tmp_path: Path) -> None:
    settings = Settings(
        data_dir=tmp_path,
        database_url=f"sqlite+aiosqlite:///{tmp_path / 'test.db'}",
        max_probe_concurrency=2,
    )
    database = Database(settings)
    await database.initialize()
    repository = ProxyNodeRepository(database.session_factory)
    await repository.upsert_discovered(
        [
            DiscoveredNode(
                id=f"node-{index}",
                provider="vpngate",
                ip_address=f"198.51.100.{index}",
                remote_host=f"198.51.100.{index}",
                remote_port=443,
                transport=TransportProtocol.TCP,
                config_text="success\n" if index != 2 else "failure\n",
            )
            for index in range(1, 4)
        ]
    )
    tunnel = MixedTunnelManager()
    service = ProbeService(
        settings,
        repository,
        tunnel,  # type: ignore[arg-type]
        TunAllocator(2, 3),
        latency_probe=any_latency,
    )

    try:
        results = await service.probe_many(["node-1", "node-2", "node-3", "node-1"])
    finally:
        await database.dispose()

    assert [result.node_id for result in results] == ["node-1", "node-2", "node-3"]
    assert [result.available for result in results] == [True, False, True]
    assert tunnel.max_active == 2
