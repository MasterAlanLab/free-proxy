from __future__ import annotations

import asyncio
import os

from fastapi import APIRouter, HTTPException, Request, Response, status
from pydantic import BaseModel, Field

from free_proxy.security import AdminConfig, AuthService, hash_password

router = APIRouter(prefix="/auth", tags=["auth"])


class LoginRequest(BaseModel):
    username: str
    password: str


class CredentialsUpdate(BaseModel):
    username: str = Field(min_length=1)
    password: str = ""
    secret_path: str = Field(pattern=r"^[A-Za-z0-9]+$")
    host: str
    port: int = Field(ge=1, le=65535)
    proxy_host: str | None = None
    proxy_port: int | None = Field(default=None, ge=1, le=65535)


@router.post("/login")
async def login(payload: LoginRequest, request: Request, response: Response) -> dict[str, bool]:
    auth: AuthService = request.app.state.auth_service
    if not auth.verify(payload.username, payload.password):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Incorrect username or password",
        )
    token = await auth.sessions.create()
    cookie_path = request.scope.get("root_path") or "/"
    response.set_cookie(
        "session",
        token,
        max_age=auth.settings.session_ttl_seconds,
        httponly=True,
        samesite="lax",
        path=str(cookie_path),
    )
    return {"ok": True}


@router.post("/logout")
async def logout(request: Request, response: Response) -> dict[str, bool]:
    auth: AuthService = request.app.state.auth_service
    await auth.sessions.remove(getattr(request.state, "session_token", None))
    response.delete_cookie(
        "session",
        path=str(request.scope.get("root_path") or "/"),
    )
    return {"ok": True}


@router.get("/config")
async def auth_config(request: Request) -> dict[str, object]:
    config = request.app.state.auth_service.store.config
    return {
        "username": config.username,
        "secret_path": config.secret_path,
        "host": config.host,
        "port": config.port,
        "proxy_host": config.proxy_host,
        "proxy_port": config.proxy_port,
        "password_set": bool(config.password_hash),
    }


@router.put("/credentials")
async def update_credentials(
    payload: CredentialsUpdate,
    request: Request,
) -> dict[str, object]:
    auth: AuthService = request.app.state.auth_service
    previous = auth.store.config
    updated = AdminConfig(
        username=payload.username,
        password_hash=(
            hash_password(payload.password) if payload.password else previous.password_hash
        ),
        secret_path=payload.secret_path,
        host=payload.host,
        port=payload.port,
        proxy_host=payload.proxy_host or previous.proxy_host,
        proxy_port=payload.proxy_port or previous.proxy_port,
    )
    auth.store.update(updated)
    reauth_required = (
        updated.username != previous.username
        or updated.password_hash != previous.password_hash
    )
    if reauth_required:
        await auth.sessions.clear()
    restart_needed = (
        updated.host != previous.host
        or updated.port != previous.port
        or updated.secret_path != previous.secret_path
        or updated.proxy_host != previous.proxy_host
        or updated.proxy_port != previous.proxy_port
    )
    if restart_needed and auth.settings.allow_process_restart:
        asyncio.create_task(restart_process(), name="restart-after-admin-config-update")
    return {
        "ok": True,
        "restart_needed": restart_needed,
        "reauth_required": reauth_required,
    }


async def restart_process() -> None:
    await asyncio.sleep(2)
    os._exit(0)
