import asyncio
from pathlib import Path

import pytest

from free_proxy.config import Settings
from free_proxy.domain.enums import JobStatus
from free_proxy.domain.models import JobRead, utc_now
from free_proxy.infrastructure.database import Database, JobRepository
from free_proxy.services.jobs import JobService


@pytest.mark.asyncio
async def test_jobs_persist_and_incomplete_jobs_are_cancelled(tmp_path: Path) -> None:
    settings = Settings(
        data_dir=tmp_path,
        database_url=f"sqlite+aiosqlite:///{tmp_path / 'test.db'}",
    )
    database = Database(settings)
    await database.initialize()
    repository = JobRepository(database.session_factory)
    service = JobService(repository)
    await service.initialize()

    async def operation() -> dict[str, object]:
        return {"ok": True}

    try:
        submitted = await service.submit("persistent-job", operation)
        completed = await wait_for_job(service, submitted.id)
        reloaded = await JobService(repository).get(submitted.id)
        assert completed.status is JobStatus.SUCCEEDED
        assert reloaded is not None
        assert reloaded.result == {"ok": True}

        interrupted = JobRead(
            id="interrupted",
            name="old-job",
            status=JobStatus.RUNNING,
            created_at=utc_now(),
            started_at=utc_now(),
        )
        await repository.save(interrupted)
        restarted = JobService(repository)
        await restarted.initialize()
        recovered = await restarted.get("interrupted")
        assert recovered is not None
        assert recovered.status is JobStatus.CANCELLED
        assert recovered.finished_at is not None
    finally:
        await service.shutdown()
        await database.dispose()


async def wait_for_job(service: JobService, job_id: str) -> JobRead:
    for _ in range(100):
        job = await service.get(job_id)
        if job is not None and job.status in {
            JobStatus.SUCCEEDED,
            JobStatus.FAILED,
            JobStatus.CANCELLED,
        }:
            return job
        await asyncio.sleep(0.01)
    raise AssertionError(f"Job did not finish: {job_id}")
