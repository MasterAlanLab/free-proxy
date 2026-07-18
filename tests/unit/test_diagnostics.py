from pathlib import Path

import pytest

from free_proxy.config import Settings
from free_proxy.domain.exceptions import ProviderError
from free_proxy.infrastructure.network.commands import CommandResult
from free_proxy.services.diagnostics import NetworkDiagnosticsService, is_dns_error
from free_proxy.services.discovery import DiscoveryService


class FakeRunner:
    def __init__(self) -> None:
        self.commands: list[list[str]] = []

    async def run(self, command: list[str], *, timeout_seconds: float = 5) -> CommandResult:
        self.commands.append(command)
        if command == ["ip", "route", "show", "default"]:
            return CommandResult(0, "default via 192.0.2.1 dev eth0 proto dhcp", "")
        return CommandResult(0, "", "")


@pytest.mark.asyncio
async def test_dns_repair_updates_default_interface(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    settings = Settings(data_dir=tmp_path, dns_repair_enabled=True)
    runner = FakeRunner()
    monkeypatch.setattr("free_proxy.services.diagnostics.shutil.which", lambda _: "/usr/bin/tool")
    diagnostics = NetworkDiagnosticsService(settings, runner=runner, platform="linux")

    result = await diagnostics.repair_dns()

    assert result.repaired is True
    assert result.interface == "eth0"
    assert runner.commands == [
        ["ip", "route", "show", "default"],
        ["resolvectl", "dns", "eth0", "1.1.1.1", "8.8.8.8"],
        ["resolvectl", "domain", "eth0", "~."],
        ["resolvectl", "flush-caches"],
    ]


class RetryProvider:
    name = "retry"

    def __init__(self) -> None:
        self.calls = 0

    async def discover(self) -> list[object]:
        self.calls += 1
        if self.calls == 1:
            raise ProviderError("temporary DNS failure")
        return []


class EmptyRepository:
    async def active_blacklist_ids(self) -> set[str]:
        return set()

    async def upsert_discovered(self, nodes: list[object]) -> int:
        return len(nodes)


class RepairTracker:
    auto_repair_enabled = True

    def __init__(self) -> None:
        self.repairs = 0

    async def repair_dns(self) -> None:
        self.repairs += 1


@pytest.mark.asyncio
async def test_discovery_repairs_dns_and_retries_once() -> None:
    provider = RetryProvider()
    repository = EmptyRepository()
    diagnostics = RepairTracker()
    service = DiscoveryService(provider, repository, diagnostics)  # type: ignore[arg-type]

    result = await service.discover()

    assert result.discovered == 0
    assert provider.calls == 2
    assert diagnostics.repairs == 1
    assert is_dns_error(ProviderError("cannot resolve host")) is True
