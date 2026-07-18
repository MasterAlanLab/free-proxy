from __future__ import annotations

import asyncio
import logging
import sys
from typing import Protocol

from free_proxy.config import Settings
from free_proxy.domain.exceptions import NetworkOperationError
from free_proxy.infrastructure.network.commands import CommandResult, SystemCommandRunner

logger = logging.getLogger(__name__)


class CommandRunner(Protocol):
    async def run(self, command: list[str], *, timeout_seconds: float = 5) -> CommandResult: ...


class PolicyRouter:
    def __init__(
        self,
        settings: Settings,
        runner: CommandRunner | None = None,
        platform: str | None = None,
    ) -> None:
        self._settings = settings
        self._runner = runner or SystemCommandRunner()
        self._platform = platform or sys.platform
        self._rp_filter_original: dict[str, str] = {}

    @property
    def supported(self) -> bool:
        return self._platform.startswith("linux")

    async def setup(self, interface: str | None = None) -> None:
        if not self.supported:
            raise NetworkOperationError("Policy routing is only supported on Linux")
        device = interface or self._settings.tunnel_interface
        table = str(self._settings.policy_routing_table)
        last_error: NetworkOperationError | None = None
        for attempt in range(1, self._settings.routing_setup_retries + 1):
            try:
                await self._setup_once(device, table)
                return
            except NetworkOperationError as exc:
                last_error = exc
                await self.cleanup()
                if attempt < self._settings.routing_setup_retries:
                    await asyncio.sleep(self._settings.routing_retry_interval_seconds)
        assert last_error is not None
        raise last_error

    async def _setup_once(self, device: str, table: str) -> None:
        await self.cleanup()

        route_result = await self._runner.run(
            ["ip", "route", "add", "default", "dev", device, "table", table]
        )
        if route_result.returncode != 0:
            raise NetworkOperationError(
                "Unable to add policy route for "
                f"{device}: command={' '.join(['ip', 'route', 'add'])} "
                f"returncode={route_result.returncode} stderr={route_result.stderr.strip()}"
            )

        rule_result = await self._runner.run(["ip", "rule", "add", "oif", device, "table", table])
        if rule_result.returncode != 0:
            await self.cleanup()
            raise NetworkOperationError(
                f"Unable to add policy rule for {device}: {rule_result.stderr.strip()}"
            )

        for target in ("all", "default", device):
            read_result = await self._runner.run(
                ["sysctl", "-n", f"net.ipv4.conf.{target}.rp_filter"]
            )
            if read_result.returncode == 0:
                self._rp_filter_original[target] = read_result.stdout.strip()
            result = await self._runner.run(
                ["sysctl", "-w", f"net.ipv4.conf.{target}.rp_filter=2"]
            )
            if result.returncode != 0:
                message = f"Unable to configure rp_filter for {target}: {result.stderr.strip()}"
                if self._settings.routing_strict_rp_filter:
                    raise NetworkOperationError(message)
                logger.warning(message)

    async def cleanup(self) -> None:
        if not self.supported:
            return
        table = str(self._settings.policy_routing_table)
        await self._runner.run(["ip", "rule", "del", "table", table])
        await self._runner.run(["ip", "route", "flush", "table", table])
        for target, value in self._rp_filter_original.items():
            await self._runner.run(["sysctl", "-w", f"net.ipv4.conf.{target}.rp_filter={value}"])
        self._rp_filter_original.clear()
