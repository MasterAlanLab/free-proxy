from typing import Annotated

from fastapi import APIRouter, Depends, HTTPException, status

from free_proxy.api.dependencies import get_job_service
from free_proxy.domain.models import JobRead
from free_proxy.services.jobs import JobService

router = APIRouter(prefix="/jobs", tags=["jobs"])


@router.get("/{job_id}", response_model=JobRead)
async def get_job(
    job_id: str,
    jobs: Annotated[JobService, Depends(get_job_service)],
) -> JobRead:
    job = await jobs.get(job_id)
    if job is None:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="Job not found")
    return job
