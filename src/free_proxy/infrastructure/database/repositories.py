from __future__ import annotations

from collections.abc import Sequence
from datetime import UTC, datetime, timedelta

from sqlalchemy import delete, func, or_, select, update
from sqlalchemy.dialects.sqlite import insert as sqlite_insert
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker

from free_proxy.domain.enums import IpType, JobStatus, NodeStatus, ProxyPolicyMode, RoutingIpType
from free_proxy.domain.models import (
    DiscoveredNode,
    IpInfo,
    JobRead,
    ProbeHistoryRead,
    ProbeResult,
    ProxyNodeRead,
    ProxyNodeTarget,
    ProxySettings,
    ProxySettingsUpdate,
)
from free_proxy.infrastructure.database.models import (
    BlacklistRecord,
    FavoriteRecord,
    IpInfoCacheRecord,
    JobRecord,
    NodeAliasRecord,
    ProbeResultRecord,
    ProxyNodeRecord,
    RuntimeSettingsRecord,
)


class ProxyNodeRepository:
    def __init__(self, session_factory: async_sessionmaker[AsyncSession]) -> None:
        self._session_factory = session_factory

    async def upsert_discovered(self, nodes: Sequence[DiscoveredNode]) -> int:
        if not nodes:
            return 0

        async with self._session_factory() as session:
            values = []
            for node in nodes:
                identity = node.provider_identity or f"{node.provider}:{node.ip_address}"
                existing_id = await session.scalar(
                    select(ProxyNodeRecord.id).where(
                        ProxyNodeRecord.provider == node.provider,
                        ProxyNodeRecord.provider_identity == identity,
                    )
                )
                canonical_id = existing_id or node.id
                if existing_id and existing_id != node.id:
                    await self._add_alias(session, node.id, existing_id)
                values.append({
                "id": canonical_id,
                "provider": node.provider,
                "provider_node_id": node.provider_node_id,
                "provider_identity": identity,
                "country": node.country,
                "country_code": node.country_code,
                "host_name": node.host_name,
                "ip_address": node.ip_address,
                "remote_host": node.remote_host,
                "remote_port": node.remote_port,
                "transport": node.transport,
                "source_score": node.source_score,
                "source_ping_ms": node.source_ping_ms,
                "source_speed_bps": node.source_speed_bps,
                "source_sessions": node.source_sessions,
                "config_text": node.config_text,
                "fetched_at": node.fetched_at,
                "last_seen_at": node.fetched_at,
                "source_present": True,
            })

            statement = sqlite_insert(ProxyNodeRecord).values(values)
            statement = statement.on_conflict_do_update(
                index_elements=[ProxyNodeRecord.id],
                set_={
                    "provider_identity": statement.excluded.provider_identity,
                    "provider_node_id": statement.excluded.provider_node_id,
                    "country": statement.excluded.country,
                    "country_code": statement.excluded.country_code,
                    "host_name": statement.excluded.host_name,
                    "ip_address": statement.excluded.ip_address,
                    "remote_host": statement.excluded.remote_host,
                    "remote_port": statement.excluded.remote_port,
                    "transport": statement.excluded.transport,
                    "source_score": statement.excluded.source_score,
                    "source_ping_ms": statement.excluded.source_ping_ms,
                    "source_speed_bps": statement.excluded.source_speed_bps,
                    "source_sessions": statement.excluded.source_sessions,
                    "config_text": statement.excluded.config_text,
                    "fetched_at": statement.excluded.fetched_at,
                    "last_seen_at": statement.excluded.last_seen_at,
                    "source_present": True,
                },
            )
            await session.execute(statement)
            await session.commit()
        return len(nodes)

    async def list_nodes(
        self,
        *,
        limit: int = 100,
        offset: int = 0,
        ip_type: IpType | None = None,
        status: NodeStatus | None = None,
        country: str | None = None,
        search: str | None = None,
        current_only: bool = False,
    ) -> list[ProxyNodeRead]:
        statement = select(ProxyNodeRecord)
        if current_only:
            statement = statement.where(ProxyNodeRecord.source_present.is_(True))
        if ip_type is not None:
            statement = statement.where(ProxyNodeRecord.ip_type == ip_type)
        if status is not None:
            statement = statement.where(ProxyNodeRecord.status == status)
        if country:
            statement = statement.where(ProxyNodeRecord.country == country)
        if search:
            pattern = f"%{search.strip()}%"
            statement = statement.where(
                or_(
                    ProxyNodeRecord.ip_address.like(pattern),
                    ProxyNodeRecord.country.like(pattern),
                    ProxyNodeRecord.owner.like(pattern),
                    ProxyNodeRecord.as_name.like(pattern),
                )
            )
        statement = (
            statement.order_by(
                ProxyNodeRecord.status.asc(),
                ProxyNodeRecord.latency_ms.asc(),
                ProxyNodeRecord.source_score.desc(),
            )
            .limit(limit)
            .offset(offset)
        )

        async with self._session_factory() as session:
            result = await session.scalars(statement)
            return [ProxyNodeRead.model_validate(record) for record in result.all()]

    async def count_nodes(
        self,
        *,
        ip_type: IpType | None = None,
        status: NodeStatus | None = None,
        country: str | None = None,
        search: str | None = None,
        current_only: bool = False,
    ) -> int:
        statement = select(func.count()).select_from(ProxyNodeRecord)
        if current_only:
            statement = statement.where(ProxyNodeRecord.source_present.is_(True))
        if ip_type is not None:
            statement = statement.where(ProxyNodeRecord.ip_type == ip_type)
        if status is not None:
            statement = statement.where(ProxyNodeRecord.status == status)
        if country:
            statement = statement.where(ProxyNodeRecord.country == country)
        if search:
            pattern = f"%{search.strip()}%"
            statement = statement.where(
                or_(
                    ProxyNodeRecord.ip_address.like(pattern),
                    ProxyNodeRecord.country.like(pattern),
                    ProxyNodeRecord.owner.like(pattern),
                    ProxyNodeRecord.as_name.like(pattern),
                )
            )

        async with self._session_factory() as session:
            return int(await session.scalar(statement) or 0)

    async def get_target(self, node_id: str) -> ProxyNodeTarget | None:
        node_id = await self.resolve_alias(node_id)
        statement = select(ProxyNodeRecord).where(ProxyNodeRecord.id == node_id)
        async with self._session_factory() as session:
            record = await session.scalar(statement)
            if record is None:
                return None
            return ProxyNodeTarget(
                id=record.id,
                ip_address=record.ip_address,
                remote_host=record.remote_host,
                remote_port=record.remote_port,
                source_ping_ms=record.source_ping_ms,
                config_text=record.config_text,
            )

    async def get_node(self, node_id: str) -> ProxyNodeRead | None:
        node_id = await self.resolve_alias(node_id)
        statement = select(ProxyNodeRecord).where(ProxyNodeRecord.id == node_id)
        async with self._session_factory() as session:
            record = await session.scalar(statement)
            return ProxyNodeRead.model_validate(record) if record is not None else None

    async def resolve_alias(self, node_id: str) -> str:
        async with self._session_factory() as session:
            alias = await session.get(NodeAliasRecord, node_id)
            return alias.node_id if alias is not None else node_id

    async def add_alias(self, alias_id: str, node_id: str) -> None:
        async with self._session_factory() as session:
            await self._add_alias(session, alias_id, node_id)
            await session.commit()

    async def _add_alias(self, session: AsyncSession, alias_id: str, node_id: str) -> None:
        if alias_id == node_id:
            return
        statement = sqlite_insert(NodeAliasRecord).values(
            alias_id=alias_id, node_id=node_id, created_at=datetime.now(UTC)
        )
        await session.execute(
            statement.on_conflict_do_update(
                index_elements=[NodeAliasRecord.alias_id],
                set_={"node_id": node_id},
            )
        )

    async def mark_provider_snapshot(self, provider: str, present_identities: set[str]) -> None:
        statement = update(ProxyNodeRecord).where(ProxyNodeRecord.provider == provider)
        if present_identities:
            statement = statement.where(
                ProxyNodeRecord.provider_identity.not_in(present_identities)
            )
        statement = statement.values(source_present=False)
        async with self._session_factory() as session:
            await session.execute(statement)
            await session.commit()

    async def purge_stale_nodes(self, grace_seconds: int) -> int:
        cutoff = datetime.now(UTC) - timedelta(seconds=grace_seconds)
        protected = select(FavoriteRecord.node_id)
        blacklisted = select(BlacklistRecord.node_id)
        fixed = select(RuntimeSettingsRecord.fixed_node_id).where(
            RuntimeSettingsRecord.id == 1,
            RuntimeSettingsRecord.fixed_node_id.is_not(None),
        )
        statement = delete(ProxyNodeRecord).where(
            ProxyNodeRecord.source_present.is_(False),
            ProxyNodeRecord.last_seen_at.is_not(None),
            ProxyNodeRecord.last_seen_at < cutoff,
            ProxyNodeRecord.id.not_in(protected),
            ProxyNodeRecord.id.not_in(blacklisted),
            ProxyNodeRecord.id.not_in(fixed),
        )
        async with self._session_factory() as session:
            result = await session.execute(statement)
            await session.commit()
            return int(getattr(result, "rowcount", 0) or 0)

    async def cached_ip_info(self, ip_address: str) -> tuple[IpInfo, datetime] | None:
        async with self._session_factory() as session:
            record = await session.get(IpInfoCacheRecord, ip_address)
            if record is None:
                return None
            return (
                IpInfo(
                    ip_address=record.ip_address,
                    owner=record.owner,
                    asn=record.asn,
                    as_name=record.as_name,
                    location=record.location,
                    ip_type=record.ip_type,
                    quality=record.quality,
                ),
                record.updated_at,
            )

    async def cache_ip_info(self, info: IpInfo, *, updated_at: datetime | None = None) -> None:
        when = updated_at or datetime.now(UTC)
        values = {**info.model_dump(), "updated_at": when}
        statement = sqlite_insert(IpInfoCacheRecord).values(values)
        statement = statement.on_conflict_do_update(
            index_elements=[IpInfoCacheRecord.ip_address],
            set_={key: value for key, value in values.items() if key != "ip_address"},
        )
        async with self._session_factory() as session:
            await session.execute(statement)
            await session.execute(
                update(ProxyNodeRecord)
                .where(ProxyNodeRecord.ip_address == info.ip_address)
                .values(
                    owner=info.owner,
                    asn=info.asn,
                    as_name=info.as_name,
                    location=info.location,
                    ip_type=info.ip_type,
                    quality=info.quality,
                    ip_info_updated_at=when,
                )
            )
            await session.commit()

    async def update_probe_result(
        self,
        *,
        node_id: str,
        available: bool,
        latency_ms: int,
        probed_at: datetime,
    ) -> None:
        if available:
            values = {
                "status": NodeStatus.READY,
                "latency_ms": latency_ms,
                "last_probed_at": probed_at,
                "last_success_at": probed_at,
                "consecutive_failures": 0,
                "success_count": ProxyNodeRecord.success_count + 1,
            }
        else:
            values = {
                "status": NodeStatus.UNAVAILABLE,
                "latency_ms": latency_ms,
                "last_probed_at": probed_at,
                "consecutive_failures": ProxyNodeRecord.consecutive_failures + 1,
                "failure_count": ProxyNodeRecord.failure_count + 1,
            }
        statement = update(ProxyNodeRecord).where(ProxyNodeRecord.id == node_id).values(**values)
        async with self._session_factory() as session:
            await session.execute(statement)
            await session.commit()

    async def mark_probing(self, node_id: str) -> None:
        node_id = await self.resolve_alias(node_id)
        async with self._session_factory() as session:
            await session.execute(
                update(ProxyNodeRecord)
                .where(ProxyNodeRecord.id == node_id)
                .values(status=NodeStatus.PROBING)
            )
            await session.commit()

    async def record_probe_failure(self, result: ProbeResult) -> None:
        await self.update_probe_result(
            node_id=result.node_id,
            available=False,
            latency_ms=result.latency_ms,
            probed_at=result.probed_at,
        )

    async def update_ip_info(
        self,
        node_id: str,
        info: IpInfo,
        *,
        updated_at: datetime | None = None,
    ) -> None:
        statement = (
            update(ProxyNodeRecord)
            .where(ProxyNodeRecord.id == node_id)
            .values(
                owner=info.owner,
                asn=info.asn,
                as_name=info.as_name,
                location=info.location,
                ip_type=info.ip_type,
                quality=info.quality,
                ip_info_updated_at=updated_at or datetime.now(UTC),
            )
        )
        async with self._session_factory() as session:
            await session.execute(statement)
            await session.commit()

    async def stale_ip_info_node_ids(
        self,
        node_ids: list[str],
        max_age_seconds: int,
    ) -> set[str]:
        if not node_ids:
            return set()
        cutoff = datetime.now(UTC) - timedelta(seconds=max_age_seconds)
        statement = select(ProxyNodeRecord.id).where(
            ProxyNodeRecord.id.in_(node_ids),
            (
                (ProxyNodeRecord.ip_info_updated_at.is_(None))
                | (ProxyNodeRecord.ip_info_updated_at <= cutoff)
            ),
        )
        async with self._session_factory() as session:
            return set(await session.scalars(statement))

    async def mark_unavailable(self, node_id: str) -> None:
        node_id = await self.resolve_alias(node_id)
        statement = (
            update(ProxyNodeRecord)
            .where(ProxyNodeRecord.id == node_id)
            .values(
                status=NodeStatus.UNAVAILABLE,
                consecutive_failures=ProxyNodeRecord.consecutive_failures + 1,
                failure_count=ProxyNodeRecord.failure_count + 1,
            )
        )
        async with self._session_factory() as session:
            await session.execute(statement)
            await session.commit()

    async def blacklist(self, node_id: str, reason: str, backoff_seconds: int) -> None:
        node_id = await self.resolve_alias(node_id)
        now = datetime.now(UTC)
        expires_at = now + timedelta(seconds=backoff_seconds)
        async with self._session_factory() as session:
            record = await session.get(BlacklistRecord, node_id)
            if record is None:
                record = BlacklistRecord(
                    node_id=node_id,
                    reason=reason,
                    marked_at=now,
                    expires_at=expires_at,
                )
                session.add(record)
            else:
                record.reason = reason
                record.marked_at = now
                record.expires_at = expires_at
            await session.execute(
                update(ProxyNodeRecord)
                .where(ProxyNodeRecord.id == node_id)
                .values(status=NodeStatus.COOLDOWN, cooldown_until=expires_at)
            )
            await session.commit()

    async def active_blacklist_ids(self) -> set[str]:
        await self.clear_expired_blacklist()
        now = datetime.now(UTC)
        statement = select(BlacklistRecord.node_id).where(BlacklistRecord.expires_at > now)
        async with self._session_factory() as session:
            return set(await session.scalars(statement))

    async def clear_expired_blacklist(self) -> None:
        now = datetime.now(UTC)
        async with self._session_factory() as session:
            expired_ids = list(
                await session.scalars(
                    select(BlacklistRecord.node_id).where(BlacklistRecord.expires_at <= now)
                )
            )
            if expired_ids:
                await session.execute(
                    update(ProxyNodeRecord)
                    .where(ProxyNodeRecord.id.in_(expired_ids))
                    .values(status=NodeStatus.UNAVAILABLE, cooldown_until=None)
                )
                await session.execute(
                    delete(BlacklistRecord).where(BlacklistRecord.node_id.in_(expired_ids))
                )
                await session.commit()


class SettingsRepository:
    def __init__(self, session_factory: async_sessionmaker[AsyncSession]) -> None:
        self._session_factory = session_factory

    async def get(self) -> ProxySettings:
        async with self._session_factory() as session:
            record = await session.get(RuntimeSettingsRecord, 1)
            if record is None:
                record = RuntimeSettingsRecord(id=1)
                session.add(record)
                await session.commit()
                await session.refresh(record)
            favorites = list(await session.scalars(select(FavoriteRecord.node_id)))
            return ProxySettings(
                routing_mode=ProxyPolicyMode(record.routing_mode),
                force_country=record.force_country,
                routing_ip_type=RoutingIpType(record.routing_ip_type),
                connection_enabled=record.connection_enabled,
                fixed_node_id=record.fixed_node_id,
                favorite_node_ids=favorites,
            )

    async def update(self, settings: ProxySettingsUpdate) -> ProxySettings:
        if settings.fixed_node_id:
            settings = settings.model_copy(
                update={"fixed_node_id": await self._resolve_alias(settings.fixed_node_id)}
            )
        async with self._session_factory() as session:
            record = await session.get(RuntimeSettingsRecord, 1)
            if record is None:
                record = RuntimeSettingsRecord(id=1)
                session.add(record)
            record.routing_mode = settings.routing_mode
            record.force_country = settings.force_country
            record.routing_ip_type = settings.routing_ip_type
            record.connection_enabled = settings.connection_enabled
            record.fixed_node_id = settings.fixed_node_id
            await session.commit()
        return await self.get()

    async def set_connection_enabled(self, enabled: bool) -> ProxySettings:
        current = await self.get()
        return await self.update(
            ProxySettingsUpdate(
                routing_mode=current.routing_mode,
                force_country=current.force_country,
                routing_ip_type=current.routing_ip_type,
                connection_enabled=enabled,
                fixed_node_id=current.fixed_node_id,
            )
        )

    async def toggle_favorite(self, node_id: str) -> ProxySettings:
        node_id = await self._resolve_alias(node_id)
        async with self._session_factory() as session:
            favorite = await session.get(FavoriteRecord, node_id)
            if favorite is None:
                session.add(FavoriteRecord(node_id=node_id))
            else:
                await session.delete(favorite)
            await session.commit()
        return await self.get()

    async def replace_favorites(self, node_ids: list[str]) -> None:
        node_ids = [await self._resolve_alias(node_id) for node_id in node_ids]
        async with self._session_factory() as session:
            await session.execute(delete(FavoriteRecord))
            session.add_all(
                FavoriteRecord(node_id=node_id)
                for node_id in dict.fromkeys(node_ids)
            )
            await session.commit()

    async def _resolve_alias(self, node_id: str) -> str:
        async with self._session_factory() as session:
            alias = await session.get(NodeAliasRecord, node_id)
            return alias.node_id if alias is not None else node_id


class JobRepository:
    def __init__(self, session_factory: async_sessionmaker[AsyncSession]) -> None:
        self._session_factory = session_factory

    async def save(self, job: JobRead) -> None:
        values = job.model_dump()
        statement = sqlite_insert(JobRecord).values(values)
        statement = statement.on_conflict_do_update(
            index_elements=[JobRecord.id],
            set_={key: value for key, value in values.items() if key != "id"},
        )
        async with self._session_factory() as session:
            await session.execute(statement)
            await session.commit()

    async def get(self, job_id: str) -> JobRead | None:
        async with self._session_factory() as session:
            record = await session.get(JobRecord, job_id)
            return job_from_record(record) if record is not None else None

    async def cancel_incomplete(self) -> int:
        now = datetime.now(UTC)
        statuses = [JobStatus.PENDING, JobStatus.RUNNING]
        statement = (
            update(JobRecord)
            .where(JobRecord.status.in_(statuses))
            .values(
                status=JobStatus.CANCELLED,
                finished_at=now,
                error="Application restarted before the job completed",
            )
        )
        async with self._session_factory() as session:
            count = await session.scalar(
                select(func.count()).select_from(JobRecord).where(JobRecord.status.in_(statuses))
            )
            await session.execute(statement)
            await session.commit()
            return int(count or 0)


class ProbeResultRepository:
    def __init__(self, session_factory: async_sessionmaker[AsyncSession]) -> None:
        self._session_factory = session_factory

    async def record(self, result: ProbeResult) -> None:
        async with self._session_factory() as session:
            session.add(
                ProbeResultRecord(
                    node_id=result.node_id,
                    available=result.available,
                    latency_ms=result.latency_ms,
                    probed_at=result.probed_at,
                    result=result.model_dump(mode="json"),
                )
            )
            await session.commit()

    async def list_for_node(self, node_id: str, *, limit: int = 100) -> list[ProbeHistoryRead]:
        statement = (
            select(ProbeResultRecord)
            .where(ProbeResultRecord.node_id == node_id)
            .order_by(ProbeResultRecord.probed_at.desc())
            .limit(limit)
        )
        async with self._session_factory() as session:
            records = list(await session.scalars(statement))
        return [
            ProbeHistoryRead(
                id=record.id,
                node_id=record.node_id,
                available=record.available,
                latency_ms=record.latency_ms,
                probed_at=record.probed_at,
                result=record.result,
            )
            for record in records
        ]


def job_from_record(record: JobRecord) -> JobRead:
    return JobRead(
        id=record.id,
        name=record.name,
        status=JobStatus(record.status),
        created_at=record.created_at,
        started_at=record.started_at,
        finished_at=record.finished_at,
        result=record.result,
        error=record.error,
    )
