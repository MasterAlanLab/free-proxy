import asyncio
from datetime import UTC, datetime
from pathlib import Path
from types import SimpleNamespace

import pytest

from free_proxy.config import Settings
from free_proxy.domain.enums import (
    IpType,
    NodeStatus,
    ProxyPolicyMode,
    RoutingIpType,
    TransportProtocol,
    TunnelStatus,
)
from free_proxy.domain.models import (
    DiscoveryResult,
    ProbeResult,
    ProxyNodeRead,
    ProxySettings,
    TunnelStartResult,
)
from free_proxy.services.active_latency import ActiveLatencyMonitor
from free_proxy.services.maintenance import MaintenanceMonitor, MaintenanceService
from free_proxy.services.proxy_pool import ProxyPoolService


def node(node_id: str) -> ProxyNodeRead:
    return ProxyNodeRead(
        id=node_id,
        provider="vpngate",
        provider_node_id=node_id,
        country="Japan",
        country_code="JP",
        host_name=node_id,
        ip_address="198.51.100.10",
        remote_host="198.51.100.10",
        remote_port=443,
        transport=TransportProtocol.TCP,
        ip_type=IpType.UNKNOWN,
        owner="",
        asn="",
        as_name="",
        location="",
        quality="",
        status=NodeStatus.DISCOVERED,
        source_score=100,
        source_ping_ms=10,
        source_speed_bps=1000,
        source_sessions=1,
        latency_ms=0,
        consecutive_failures=0,
        success_count=0,
        failure_count=0,
        fetched_at=datetime.now(UTC),
        last_probed_at=None,
        last_success_at=None,
        cooldown_until=None,
    )


class NodeStore:
    def __init__(self, nodes: list[ProxyNodeRead]) -> None:
        self.nodes = nodes
        self.cleared = 0

    async def clear_expired_blacklist(self) -> None:
        self.cleared += 1

    async def list_nodes(self, *, limit: int = 1000) -> list[ProxyNodeRead]:
        return self.nodes

    async def count_nodes(self, *, status: NodeStatus | None = None) -> int:
        return sum(status is None or item.status is status for item in self.nodes)


class RuntimeSettings:
    async def get(self) -> ProxySettings:
        return ProxySettings(
            routing_mode=ProxyPolicyMode.AUTO,
            routing_ip_type=RoutingIpType.ALL,
            connection_enabled=True,
        )


class Discovery:
    async def discover(self) -> DiscoveryResult:
        return DiscoveryResult(provider="test", discovered=1, stored=1)


class Probe:
    def __init__(self) -> None:
        self.calls: list[list[str]] = []

    async def probe_many(self, node_ids: list[str]) -> list[ProbeResult]:
        self.calls.append(node_ids)
        return [
            ProbeResult(
                node_id=node_id,
                available=True,
                latency_ms=20,
                tunnel=TunnelStartResult(
                    success=True,
                    status=TunnelStatus.CONNECTED,
                    message="connected",
                ),
            )
            for node_id in node_ids
        ]


class Gateway:
    def __init__(self) -> None:
        self.active_node_id: str | None = None
        self.latency = 0

    def status(self) -> SimpleNamespace:
        return SimpleNamespace(active_node_id=self.active_node_id)

    def update_active_latency(self, latency_ms: int) -> None:
        self.latency = latency_ms


class AutoSwitch:
    def __init__(self, gateway: Gateway) -> None:
        self.gateway = gateway

    async def switch(self) -> TunnelStartResult:
        self.gateway.active_node_id = "node-1"
        return TunnelStartResult(
            success=True,
            status=TunnelStatus.CONNECTED,
            message="connected",
        )


class Pool:
    apply_filters = staticmethod(ProxyPoolService.apply_filters)


@pytest.mark.asyncio
async def test_maintenance_fast_probes_and_connects_first_available_node(tmp_path: Path) -> None:
    settings = Settings(data_dir=tmp_path, initial_connect_test_limit=1)
    nodes = NodeStore([node("node-1"), node("node-2")])
    probe = Probe()
    gateway = Gateway()
    service = MaintenanceService(
        settings,
        nodes,  # type: ignore[arg-type]
        RuntimeSettings(),  # type: ignore[arg-type]
        Discovery(),  # type: ignore[arg-type]
        probe,  # type: ignore[arg-type]
        Pool(),  # type: ignore[arg-type]
        gateway,  # type: ignore[arg-type]
        AutoSwitch(gateway),  # type: ignore[arg-type]
    )

    result = await service.run()

    assert nodes.cleared == 1
    assert probe.calls == [["node-1"]]
    assert result.probed == 1
    assert result.connected_node_id == "node-1"


class FailingMaintenance:
    async def run(self) -> None:
        raise RuntimeError("failed")


@pytest.mark.asyncio
async def test_maintenance_monitor_uses_disconnected_retry(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    settings = Settings(
        data_dir=tmp_path,
        disconnected_retry_seconds=3,
        maintenance_interval_seconds=30,
    )
    gateway = Gateway()
    monitor = MaintenanceMonitor(
        settings,
        FailingMaintenance(),  # type: ignore[arg-type]
        gateway,  # type: ignore[arg-type]
    )
    delays: list[float] = []

    async def stop_after_sleep(delay: float) -> None:
        delays.append(delay)
        raise asyncio.CancelledError

    monkeypatch.setattr("free_proxy.services.maintenance.asyncio.sleep", stop_after_sleep)
    with pytest.raises(asyncio.CancelledError):
        await monitor._run()

    assert delays == [3]


class ActiveNodeStore:
    async def get_target(self, node_id: str) -> SimpleNamespace:
        return SimpleNamespace(
            remote_host="198.51.100.10",
            remote_port=443,
            source_ping_ms=10,
        )


@pytest.mark.asyncio
async def test_active_latency_monitor_refreshes_and_stops_cleanly(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    settings = Settings(data_dir=tmp_path, active_ping_interval_seconds=3600)
    gateway = Gateway()
    gateway.active_node_id = "node-1"
    monitor = ActiveLatencyMonitor(
        settings,
        ActiveNodeStore(),  # type: ignore[arg-type]
        gateway,  # type: ignore[arg-type]
    )

    async def latency(host: str, port: int, timeout_seconds: float) -> int:
        return 77

    async def stop_after_sleep(delay: float) -> None:
        raise asyncio.CancelledError

    monkeypatch.setattr("free_proxy.services.active_latency.measure_node_latency", latency)
    monkeypatch.setattr("free_proxy.services.active_latency.asyncio.sleep", stop_after_sleep)
    with pytest.raises(asyncio.CancelledError):
        await monitor._run()
    assert gateway.latency == 77

    async def wait_forever() -> None:
        await asyncio.Event().wait()

    monkeypatch.setattr(monitor, "_run", wait_forever)
    monitor.start()
    assert monitor.running is True
    await monitor.stop()
    assert monitor.running is False
