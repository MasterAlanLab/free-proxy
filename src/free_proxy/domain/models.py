from __future__ import annotations

from datetime import UTC, datetime
from typing import Any

from pydantic import BaseModel, ConfigDict, Field

from free_proxy.domain.enums import (
    IpType,
    JobStatus,
    NodeStatus,
    ProxyPolicyMode,
    RoutingIpType,
    TransportProtocol,
    TunnelFailureCode,
    TunnelStatus,
)


def utc_now() -> datetime:
    return datetime.now(UTC)


class DiscoveredNode(BaseModel):
    id: str
    provider: str
    provider_node_id: str = ""
    provider_identity: str = ""
    country: str = ""
    country_code: str = ""
    host_name: str = ""
    ip_address: str
    remote_host: str
    remote_port: int
    transport: TransportProtocol = TransportProtocol.UNKNOWN
    source_score: int = 0
    source_ping_ms: int = 0
    source_speed_bps: int = 0
    source_sessions: int = 0
    config_text: str
    fetched_at: datetime = Field(default_factory=utc_now)


class ProxyNodeRead(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: str
    provider: str
    provider_node_id: str
    provider_identity: str = ""
    country: str
    country_code: str
    host_name: str
    ip_address: str
    remote_host: str
    remote_port: int
    transport: TransportProtocol
    ip_type: IpType
    owner: str
    asn: str
    as_name: str
    location: str
    quality: str
    status: NodeStatus
    source_score: int
    source_ping_ms: int
    source_speed_bps: int
    source_sessions: int
    latency_ms: int
    consecutive_failures: int
    success_count: int
    failure_count: int
    fetched_at: datetime
    last_probed_at: datetime | None
    last_success_at: datetime | None
    ip_info_updated_at: datetime | None = None
    cooldown_until: datetime | None
    last_seen_at: datetime | None = None
    source_present: bool = True


class ProxyNodePage(BaseModel):
    items: list[ProxyNodeRead]
    total: int
    limit: int
    offset: int


class PoolStatistics(BaseModel):
    total: int
    ready: int
    discovered: int
    unavailable: int
    cooldown: int
    residential: int
    mobile: int
    hosting: int
    unknown: int
    favorites: int
    blacklisted: int
    countries: int


class ProxyNodeTarget(BaseModel):
    id: str
    ip_address: str
    remote_host: str
    remote_port: int
    source_ping_ms: int = 0
    config_text: str


class DiscoveryResult(BaseModel):
    provider: str
    discovered: int
    stored: int
    total_rows: int | None = None
    valid_rows: int | None = None
    duplicate_rows: int | None = None
    malformed_rows: int | None = None
    missing_field_rows: int | None = None


class MaintenanceResult(BaseModel):
    discovered: int
    probed: int
    available: int
    connected_node_id: str | None = None


class JobRead(BaseModel):
    id: str
    name: str
    status: JobStatus
    created_at: datetime
    started_at: datetime | None = None
    finished_at: datetime | None = None
    result: dict[str, Any] | None = None
    error: str | None = None


class TunnelStartResult(BaseModel):
    success: bool
    status: TunnelStatus
    message: str
    startup_time_ms: int = 0
    failure_code: TunnelFailureCode | None = None
    log_tail: list[str] = Field(default_factory=list)
    handshake_stage: str = "starting"


class ProbeResult(BaseModel):
    node_id: str
    available: bool
    latency_ms: int
    tunnel: TunnelStartResult
    probed_at: datetime = Field(default_factory=utc_now)


class ProbeHistoryRead(BaseModel):
    id: int
    node_id: str
    available: bool
    latency_ms: int
    probed_at: datetime
    result: dict[str, Any]


class ProbeManyRequest(BaseModel):
    ids: list[str]


class GatewayStatus(BaseModel):
    running: bool
    active_node_id: str | None = None
    tunnel_status: TunnelStatus = TunnelStatus.IDLE
    proxy_listener: str
    socks_listener: str
    http_listener: str
    last_error: str | None = None
    active_latency_ms: int = 0
    exit_ip: str | None = None
    exit_latency_ms: int = 0
    connection_enabled: bool = True
    monitor_status: dict[str, Any] = Field(default_factory=dict)


class IpInfo(BaseModel):
    ip_address: str
    owner: str = ""
    asn: str = ""
    as_name: str = ""
    location: str = ""
    ip_type: IpType = IpType.UNKNOWN
    quality: str = ""


class ProxySettings(BaseModel):
    routing_mode: ProxyPolicyMode = ProxyPolicyMode.AUTO
    force_country: str = ""
    routing_ip_type: RoutingIpType = RoutingIpType.ALL
    connection_enabled: bool = True
    fixed_node_id: str | None = None
    favorite_node_ids: list[str] = Field(default_factory=list)


class ProxySettingsUpdate(BaseModel):
    routing_mode: ProxyPolicyMode
    force_country: str = ""
    routing_ip_type: RoutingIpType = RoutingIpType.ALL
    connection_enabled: bool = True
    fixed_node_id: str | None = None


class ProxyHealthResult(BaseModel):
    ok: bool
    exit_ip: str | None = None
    latency_ms: int = 0
    error: str | None = None


class DiagnosticCheck(BaseModel):
    name: str
    ok: bool
    detail: str
    severity: str = "error"
    recoverable: bool = False


class SystemDiagnostics(BaseModel):
    healthy: bool
    checks: list[DiagnosticCheck]


class DnsRepairResult(BaseModel):
    repaired: bool
    interface: str
    servers: list[str]
    detail: str
