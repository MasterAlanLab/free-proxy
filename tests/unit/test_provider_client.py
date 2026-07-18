import base64
from pathlib import Path
from typing import Any

import httpx
import pytest

from free_proxy.config import Settings
from free_proxy.infrastructure.network.upstream import parse_upstream_proxy
from free_proxy.providers.vpngate.client import VpnGateProvider


class FakeResponse:
    def __init__(self, text: str) -> None:
        self.text = text

    def raise_for_status(self) -> None:
        return None


class FakeClient:
    def __init__(self, outcome: Exception | FakeResponse) -> None:
        self.outcome = outcome
        self.url = ""

    async def __aenter__(self) -> "FakeClient":
        return self

    async def __aexit__(self, *args: object) -> None:
        return None

    async def get(self, url: str) -> FakeResponse:
        self.url = url
        if isinstance(self.outcome, Exception):
            raise self.outcome
        return self.outcome


@pytest.mark.asyncio
async def test_provider_uses_tls_fallback_http_and_upstream_proxy(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    config = "client\nproto tcp\nremote 198.51.100.8 443 tcp\n"
    encoded = base64.b64encode(config.encode()).decode()
    csv_text = (
        "#HostName,IP,Score,Ping,Speed,CountryLong,CountryShort,NumVpnSessions,"
        "OpenVPN_ConfigData_Base64\n"
        f"vpn1,198.51.100.8,1000,12,9000000,Japan,JP,3,{encoded}\n"
    )
    outcomes: list[Exception | FakeResponse] = [
        httpx.ConnectError("tls failed"),
        httpx.ConnectError("insecure tls failed"),
        FakeResponse(csv_text),
    ]
    client_options: list[dict[str, Any]] = []
    clients: list[FakeClient] = []

    def client_factory(**options: Any) -> FakeClient:
        client_options.append(options)
        client = FakeClient(outcomes.pop(0))
        clients.append(client)
        return client

    monkeypatch.setattr("free_proxy.providers.vpngate.client.httpx.AsyncClient", client_factory)
    settings = Settings(
        data_dir=tmp_path,
        vpngate_api_url="https://provider.example/api/",
        upstream_proxy_url="socks5://user:pass@127.0.0.1:1080",
    )

    nodes = await VpnGateProvider(settings).discover()

    assert len(nodes) == 1
    assert [options["verify"] for options in client_options] == [True, False, True]
    assert all(
        options["proxy"] == "socks5://user:pass@127.0.0.1:1080"
        for options in client_options
    )
    assert [client.url for client in clients] == [
        "https://provider.example/api/",
        "https://provider.example/api/",
        "http://provider.example/api/",
    ]


def test_upstream_proxy_parses_http_socks_auth_and_ipv6() -> None:
    http = parse_upstream_proxy("http://user:p%40ss@proxy.example:3128")
    socks = parse_upstream_proxy("[2001:db8::10]:1080", forced_kind="socks")

    assert http is not None
    assert http.kind == "http"
    assert http.password == "p@ss"
    assert socks is not None
    assert socks.kind == "socks"
    assert socks.url == "socks5://[2001:db8::10]:1080"
