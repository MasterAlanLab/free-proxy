from __future__ import annotations

import asyncio
from collections.abc import Awaitable, Callable
from contextlib import suppress
from typing import Any
from uuid import uuid4

from free_proxy.domain.enums import JobStatus
from free_proxy.domain.models import JobRead, utc_now
from free_proxy.infrastructure.database.repositories import JobRepository

JobOperation = Callable[[], Awaitable[dict[str, Any]]]


class JobService:
    def __init__(self, repository: JobRepository) -> None:
        self._repository = repository
        self._tasks: dict[str, asyncio.Task[None]] = {}
        self._lock = asyncio.Lock()

    async def initialize(self) -> None:
        await self._repository.cancel_incomplete()

    async def submit(self, name: str, operation: JobOperation) -> JobRead:
        job = JobRead(
            id=uuid4().hex,
            name=name,
            status=JobStatus.PENDING,
            created_at=utc_now(),
        )
        async with self._lock:
            await self._repository.save(job)
            task = asyncio.create_task(self._run(job.id, operation), name=f"job:{name}:{job.id}")
            self._tasks[job.id] = task
        return job.model_copy(deep=True)

    async def get(self, job_id: str) -> JobRead | None:
        return await self._repository.get(job_id)

    async def shutdown(self) -> None:
        async with self._lock:
            tasks = list(self._tasks.values())
        for task in tasks:
            task.cancel()
        for task in tasks:
            with suppress(asyncio.CancelledError):
                await task

    async def _run(self, job_id: str, operation: JobOperation) -> None:
        await self._update(
            job_id,
            status=JobStatus.RUNNING,
            started_at=utc_now(),
        )
        try:
            result = await operation()
        except asyncio.CancelledError:
            await self._update(
                job_id,
                status=JobStatus.CANCELLED,
                finished_at=utc_now(),
            )
            raise
        except Exception as exc:
            await self._update(
                job_id,
                status=JobStatus.FAILED,
                error=str(exc),
                finished_at=utc_now(),
            )
        else:
            await self._update(
                job_id,
                status=JobStatus.SUCCEEDED,
                result=result,
                finished_at=utc_now(),
            )
        finally:
            async with self._lock:
                self._tasks.pop(job_id, None)

    async def _update(self, job_id: str, **changes: Any) -> None:
        async with self._lock:
            job = await self._repository.get(job_id)
            if job is None:
                return
            await self._repository.save(job.model_copy(update=changes))
