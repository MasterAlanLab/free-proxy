from __future__ import annotations

import httpx

from free_proxy.config import Settings
from free_proxy.domain.exceptions import ProviderError
from free_proxy.domain.models import DiscoveredNode
from free_proxy.infrastructure.network.upstream import get_upstream_proxy
from free_proxy.providers.vpngate.parser import VpnGateParseStats, parse_vpngate_response_with_stats


class VpnGateProvider:
    name = "vpngate"

    def __init__(self, settings: Settings, client: httpx.AsyncClient | None = None) -> None:
        self._settings = settings
        self._client = client
        self.last_parse_stats: VpnGateParseStats | None = None

    async def discover(self) -> list[DiscoveredNode]:
        if self._client is not None:
            return await self._fetch_with_client(self._client, self._settings.vpngate_api_url)

        targets = [(self._settings.vpngate_api_url, True)]
        if self._settings.vpngate_api_url.startswith("https://"):
            targets.append((self._settings.vpngate_api_url, False))
            targets.append((self._settings.vpngate_api_url.replace("https://", "http://", 1), True))
        upstream = get_upstream_proxy(self._settings)
        last_error: Exception | None = None
        async def attempt_targets(
            proxy: str | None,
            target_list: list[tuple[str, bool]],
        ) -> list[DiscoveredNode] | None:
            nonlocal last_error
            for url, verify in target_list:
                try:
                    async with httpx.AsyncClient(
                        follow_redirects=True,
                        timeout=self._settings.request_timeout_seconds,
                        headers={"User-Agent": "free-proxy/0.1"},
                        proxy=proxy,
                        verify=verify,
                    ) as client:
                        return await self._fetch_with_client(client, url)
                except (httpx.HTTPError, ProviderError) as exc:
                    last_error = exc
            return None

        result = await attempt_targets(upstream.url if upstream is not None else None, targets)
        if result is not None:
            return result
        if upstream is None:
            # Without a configured proxy, retry the primary endpoint once.
            result = await attempt_targets(None, [targets[0]])
        elif self._settings.upstream_direct_fallback:
            result = await attempt_targets(None, targets)
        if result is not None:
            return result
        raise ProviderError(f"Unable to fetch VPNGate nodes: {last_error}")

    async def _fetch_with_client(
        self,
        client: httpx.AsyncClient,
        url: str,
    ) -> list[DiscoveredNode]:
        try:
            response = await client.get(url)
            response.raise_for_status()
        except httpx.HTTPError as exc:
            raise ProviderError(f"Unable to fetch VPNGate nodes: {exc}") from exc

        parsed = parse_vpngate_response_with_stats(
            response.text,
            limit=self._settings.discovery_limit,
        )
        self.last_parse_stats = parsed.stats
        return parsed.nodes

    async def close(self) -> None:
        return None
