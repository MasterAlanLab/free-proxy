from datetime import UTC, datetime

from free_proxy.domain.enums import (
    IpType,
    NodeStatus,
    ProxyPolicyMode,
    RoutingIpType,
    TransportProtocol,
)
from free_proxy.domain.models import ProxyNodeRead, ProxySettings
from free_proxy.services.proxy_pool import ProxyPoolService


def make_node(
    node_id: str,
    *,
    country: str,
    ip_type: IpType,
    latency_ms: int,
    score: int,
) -> ProxyNodeRead:
    return ProxyNodeRead(
        id=node_id,
        provider="vpngate",
        provider_node_id=node_id,
        country=country,
        country_code="JP",
        host_name=node_id,
        ip_address="198.51.100.1",
        remote_host="198.51.100.1",
        remote_port=443,
        transport=TransportProtocol.TCP,
        ip_type=ip_type,
        owner="",
        asn="",
        as_name="",
        location="",
        quality="",
        status=NodeStatus.READY,
        source_score=score,
        source_ping_ms=0,
        source_speed_bps=0,
        source_sessions=0,
        latency_ms=latency_ms,
        consecutive_failures=0,
        success_count=1,
        failure_count=0,
        fetched_at=datetime.now(UTC),
        last_probed_at=None,
        last_success_at=None,
        cooldown_until=None,
    )


def test_pool_preserves_country_favorite_and_ip_type_filters() -> None:
    residential = make_node(
        "jp-home",
        country="Japan",
        ip_type=IpType.RESIDENTIAL,
        latency_ms=60,
        score=100,
    )
    hosting = make_node(
        "jp-hosting",
        country="日本",
        ip_type=IpType.HOSTING,
        latency_ms=10,
        score=1000,
    )
    settings = ProxySettings(
        routing_mode=ProxyPolicyMode.COUNTRY,
        force_country="日本",
        routing_ip_type=RoutingIpType.RESIDENTIAL,
        favorite_node_ids=["jp-home"],
    )

    filtered = ProxyPoolService.apply_filters([hosting, residential], settings)

    assert filtered == [residential]
    assert ProxyPoolService.sort_key(residential) < ProxyPoolService.sort_key(hosting)


def test_pool_favorites_mode_only_keeps_favorites() -> None:
    first = make_node(
        "first",
        country="Japan",
        ip_type=IpType.RESIDENTIAL,
        latency_ms=20,
        score=100,
    )
    second = make_node(
        "second",
        country="Japan",
        ip_type=IpType.RESIDENTIAL,
        latency_ms=10,
        score=200,
    )
    settings = ProxySettings(
        routing_mode=ProxyPolicyMode.FAVORITES,
        favorite_node_ids=["first"],
    )

    assert ProxyPoolService.apply_filters([first, second], settings) == [first]
