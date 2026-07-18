from __future__ import annotations

from sqlalchemy import inspect
from sqlalchemy.ext.asyncio import (
    AsyncEngine,
    AsyncSession,
    async_sessionmaker,
    create_async_engine,
)

from free_proxy.config import Settings
from free_proxy.infrastructure.database.models import Base


class Database:
    def __init__(self, settings: Settings) -> None:
        if settings.database_url is None:
            raise ValueError("database_url must be configured")
        self.engine: AsyncEngine = create_async_engine(
            settings.database_url,
            echo=settings.sql_echo,
        )
        self.session_factory = async_sessionmaker(
            self.engine,
            class_=AsyncSession,
            expire_on_commit=False,
        )

    async def initialize(self) -> None:
        async with self.engine.begin() as connection:
            await connection.run_sync(Base.metadata.create_all)

    async def schema_tables(self) -> set[str]:
        async with self.engine.connect() as connection:
            return await connection.run_sync(
                lambda sync_connection: set(inspect(sync_connection).get_table_names())
            )

    async def dispose(self) -> None:
        await self.engine.dispose()
