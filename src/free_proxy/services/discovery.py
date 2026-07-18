from __future__ import annotations

import logging
from typing import Any

from free_proxy.domain.exceptions import ProviderError
from free_proxy.domain.models import DiscoveryResult
from free_proxy.infrastructure.database.repositories import ProxyNodeRepository
from free_proxy.providers.base import ProxyProvider
from free_proxy.services.diagnostics import NetworkDiagnosticsService, is_dns_error

logger = logging.getLogger(__name__)


class DiscoveryService:
    def __init__(
        self,
        provider: ProxyProvider,
        repository: ProxyNodeRepository,
        diagnostics: NetworkDiagnosticsService | None = None,
    ) -> None:
        self._provider = provider
        self._repository = repository
        self._diagnostics = diagnostics

    async def discover(self) -> DiscoveryResult:
        logger.info("Starting public proxy node discovery from %s", self._provider.name)
        try:
            nodes = await self._provider.discover()
        except ProviderError as exc:
            if not (
                self._diagnostics is not None
                and self._diagnostics.auto_repair_enabled
                and is_dns_error(exc)
            ):
                if self._diagnostics is not None:
                    diagnose_failure = getattr(
                        self._diagnostics,
                        "diagnose_provider_failure",
                        None,
                    )
                    if diagnose_failure is not None:
                        await diagnose_failure(exc)
                raise
            logger.warning("Provider DNS lookup failed; attempting automatic repair")
            await self._diagnostics.repair_dns()
            try:
                nodes = await self._provider.discover()
            except ProviderError as retry_exc:
                diagnose_failure = getattr(
                    self._diagnostics,
                    "diagnose_provider_failure",
                    None,
                )
                if diagnose_failure is not None:
                    await diagnose_failure(retry_exc)
                raise
        if self._diagnostics is not None:
            clear_failure = getattr(self._diagnostics, "clear_provider_failure", None)
            if clear_failure is not None:
                clear_failure()
        present_identities = {
            node.provider_identity or f"{node.provider}:{node.ip_address}" for node in nodes
        }
        blacklisted = await self._repository.active_blacklist_ids()
        nodes = [node for node in nodes if node.id not in blacklisted]
        stored = await self._repository.upsert_discovered(nodes)
        mark_snapshot = getattr(self._repository, "mark_provider_snapshot", None)
        stats = getattr(self._provider, "last_parse_stats", None)
        complete = stats is None or (
            getattr(stats, "malformed_rows", 0) == 0
            and getattr(stats, "missing_field_rows", 0) == 0
        )
        if mark_snapshot is not None and complete:
            await mark_snapshot(
                self._provider.name,
                present_identities,
            )
        logger.info("Discovered %d nodes and stored %d nodes", len(nodes), stored)
        return DiscoveryResult(
            provider=self._provider.name,
            discovered=len(nodes),
            stored=stored,
            total_rows=getattr(stats, "total_rows", None),
            valid_rows=getattr(stats, "valid_rows", None),
            duplicate_rows=getattr(stats, "duplicate_rows", None),
            malformed_rows=getattr(stats, "malformed_rows", None),
            missing_field_rows=getattr(stats, "missing_field_rows", None),
        )

    async def discover_job(self) -> dict[str, Any]:
        return self._result_to_dict(await self.discover())

    @staticmethod
    def _result_to_dict(result: DiscoveryResult) -> dict[str, Any]:
        return result.model_dump(mode="json")
