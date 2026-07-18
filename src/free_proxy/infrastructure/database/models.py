from __future__ import annotations

from datetime import datetime
from typing import Any

from sqlalchemy import JSON, Boolean, DateTime, Integer, String, Text, UniqueConstraint
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column

from free_proxy.domain.enums import IpType, NodeStatus, TransportProtocol


class Base(DeclarativeBase):
    pass


class ProxyNodeRecord(Base):
    __tablename__ = "proxy_nodes"
    __table_args__ = (
        UniqueConstraint("provider", "provider_identity", name="uq_proxy_nodes_provider_identity"),
    )

    id: Mapped[str] = mapped_column(String(96), primary_key=True)
    provider: Mapped[str] = mapped_column(String(32), index=True)
    provider_node_id: Mapped[str] = mapped_column(String(255), default="")
    provider_identity: Mapped[str] = mapped_column(String(255), default="", index=True)
    country: Mapped[str] = mapped_column(String(128), default="", index=True)
    country_code: Mapped[str] = mapped_column(String(16), default="", index=True)
    host_name: Mapped[str] = mapped_column(String(255), default="")
    ip_address: Mapped[str] = mapped_column(String(64), index=True)
    remote_host: Mapped[str] = mapped_column(String(255))
    remote_port: Mapped[int] = mapped_column(Integer)
    transport: Mapped[TransportProtocol] = mapped_column(
        String(16), default=TransportProtocol.UNKNOWN
    )
    ip_type: Mapped[IpType] = mapped_column(String(16), default=IpType.UNKNOWN, index=True)
    owner: Mapped[str] = mapped_column(String(255), default="")
    asn: Mapped[str] = mapped_column(String(128), default="")
    as_name: Mapped[str] = mapped_column(String(255), default="")
    location: Mapped[str] = mapped_column(String(255), default="")
    quality: Mapped[str] = mapped_column(String(32), default="")
    status: Mapped[NodeStatus] = mapped_column(
        String(16), default=NodeStatus.DISCOVERED, index=True
    )

    source_score: Mapped[int] = mapped_column(Integer, default=0)
    source_ping_ms: Mapped[int] = mapped_column(Integer, default=0)
    source_speed_bps: Mapped[int] = mapped_column(Integer, default=0)
    source_sessions: Mapped[int] = mapped_column(Integer, default=0)
    latency_ms: Mapped[int] = mapped_column(Integer, default=0)
    consecutive_failures: Mapped[int] = mapped_column(Integer, default=0)
    success_count: Mapped[int] = mapped_column(Integer, default=0)
    failure_count: Mapped[int] = mapped_column(Integer, default=0)

    config_text: Mapped[str] = mapped_column(Text)
    fetched_at: Mapped[datetime] = mapped_column(DateTime(timezone=True))
    last_probed_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True), nullable=True)
    last_success_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True), nullable=True)
    ip_info_updated_at: Mapped[datetime | None] = mapped_column(
        DateTime(timezone=True), nullable=True
    )
    cooldown_until: Mapped[datetime | None] = mapped_column(DateTime(timezone=True), nullable=True)
    last_seen_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True), nullable=True)
    source_present: Mapped[bool] = mapped_column(Boolean, default=True, index=True)


class IpInfoCacheRecord(Base):
    __tablename__ = "ip_info_cache"

    ip_address: Mapped[str] = mapped_column(String(64), primary_key=True)
    owner: Mapped[str] = mapped_column(String(255), default="")
    asn: Mapped[str] = mapped_column(String(128), default="")
    as_name: Mapped[str] = mapped_column(String(255), default="")
    location: Mapped[str] = mapped_column(String(255), default="")
    ip_type: Mapped[IpType] = mapped_column(String(16), default=IpType.UNKNOWN)
    quality: Mapped[str] = mapped_column(String(32), default="")
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True))


class NodeAliasRecord(Base):
    __tablename__ = "node_aliases"

    alias_id: Mapped[str] = mapped_column(String(96), primary_key=True)
    node_id: Mapped[str] = mapped_column(String(96), index=True)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True))


class RuntimeSettingsRecord(Base):
    __tablename__ = "runtime_settings"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, default=1)
    routing_mode: Mapped[str] = mapped_column(String(32), default="auto")
    force_country: Mapped[str] = mapped_column(String(128), default="")
    routing_ip_type: Mapped[str] = mapped_column(String(32), default="all")
    connection_enabled: Mapped[bool] = mapped_column(Boolean, default=True)
    fixed_node_id: Mapped[str | None] = mapped_column(String(96), nullable=True)


class FavoriteRecord(Base):
    __tablename__ = "favorites"

    node_id: Mapped[str] = mapped_column(String(96), primary_key=True)


class BlacklistRecord(Base):
    __tablename__ = "node_blacklist"

    node_id: Mapped[str] = mapped_column(String(96), primary_key=True)
    reason: Mapped[str] = mapped_column(Text, default="")
    marked_at: Mapped[datetime] = mapped_column(DateTime(timezone=True))
    expires_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), index=True)


class JobRecord(Base):
    __tablename__ = "jobs"

    id: Mapped[str] = mapped_column(String(64), primary_key=True)
    name: Mapped[str] = mapped_column(String(128), index=True)
    status: Mapped[str] = mapped_column(String(32), index=True)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), index=True)
    started_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True), nullable=True)
    finished_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True), nullable=True)
    result: Mapped[dict[str, Any] | None] = mapped_column(JSON, nullable=True)
    error: Mapped[str | None] = mapped_column(Text, nullable=True)


class ProbeResultRecord(Base):
    __tablename__ = "probe_results"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    node_id: Mapped[str] = mapped_column(String(96), index=True)
    available: Mapped[bool] = mapped_column(Boolean, index=True)
    latency_ms: Mapped[int] = mapped_column(Integer)
    probed_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), index=True)
    result: Mapped[dict[str, Any]] = mapped_column(JSON)
