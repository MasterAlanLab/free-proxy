from pathlib import Path

import pytest

from free_proxy.config import Settings
from free_proxy.infrastructure.network.commands import CommandResult
from free_proxy.infrastructure.network.routing import PolicyRouter


class RecordingRunner:
    def __init__(self) -> None:
        self.commands: list[list[str]] = []

    async def run(
        self,
        command: list[str],
        *,
        timeout_seconds: float = 5,
    ) -> CommandResult:
        self.commands.append(command)
        return CommandResult(returncode=0, stdout="", stderr="")


@pytest.mark.asyncio
async def test_policy_router_builds_linux_route_and_rule_commands(tmp_path: Path) -> None:
    settings = Settings(data_dir=tmp_path, policy_routing_table=123)
    runner = RecordingRunner()
    router = PolicyRouter(settings, runner=runner, platform="linux")

    await router.setup("tun0")

    assert ["ip", "route", "add", "default", "dev", "tun0", "table", "123"] in runner.commands
    assert ["ip", "rule", "add", "oif", "tun0", "table", "123"] in runner.commands
    assert ["sysctl", "-w", "net.ipv4.conf.tun0.rp_filter=2"] in runner.commands
