import json
from typing import Any

from fastapi import APIRouter, Query, Request
from fastapi.responses import Response

from free_proxy.logging import JsonLogStore

router = APIRouter(prefix="/logs", tags=["logs"])


@router.get("")
async def get_logs(
    request: Request,
    date: str | None = None,
    level: str | None = None,
    module: str | None = None,
    limit: int = Query(default=1000, ge=1, le=5000),
) -> dict[str, list[dict[str, Any]]]:
    store: JsonLogStore = request.app.state.log_store
    return {
        "logs": store.read(date=date, level=level, module=module, limit=limit)
    }


@router.get("/export")
async def export_logs(
    request: Request,
    date: str | None = None,
    level: str | None = None,
    module: str | None = None,
) -> Response:
    store: JsonLogStore = request.app.state.log_store
    entries = store.read(date=date, level=level, module=module, limit=5000)
    content = "".join(json.dumps(entry, ensure_ascii=False) + "\n" for entry in entries)
    filename = f"free-proxy-{date or 'today'}.jsonl"
    return Response(
        content,
        media_type="application/x-ndjson",
        headers={"Content-Disposition": f'attachment; filename="{filename}"'},
    )
