from __future__ import annotations

from pathlib import Path

from fastapi import FastAPI, Request
from fastapi.responses import FileResponse, HTMLResponse, JSONResponse, Response
from fastapi.staticfiles import StaticFiles
from fastapi.templating import Jinja2Templates

from free_proxy.api.routers import (
    auth_router,
    gateway_router,
    jobs_router,
    logs_router,
    pool_router,
    proxies_router,
    settings_router,
    system_router,
)
from free_proxy.config import Settings, get_settings
from free_proxy.domain.exceptions import FreeProxyError, ResourceNotFoundError
from free_proxy.lifespan import create_lifespan
from free_proxy.middleware import AdminAccessMiddleware
from free_proxy.security import AdminConfigStore, AuthService, SessionManager

WEB_DIR = Path(__file__).resolve().parent / "web"
WEB_DIST = WEB_DIR / "dist"


def create_app(settings: Settings | None = None) -> FastAPI:
    resolved_settings = settings or get_settings()
    admin_store = AdminConfigStore(resolved_settings)
    auth_service = AuthService(
        resolved_settings,
        admin_store,
        SessionManager(resolved_settings.session_ttl_seconds),
    )
    application = FastAPI(
        title="Free Proxy API",
        description="Free proxy discovery, classification, and SOCKS5 gateway",
        version="0.1.0",
        lifespan=create_lifespan(resolved_settings),
        docs_url=None,
        redoc_url=None,
    )
    application.state.auth_service = auth_service
    application.add_middleware(AdminAccessMiddleware, auth=auth_service)

    application.include_router(auth_router, prefix="/api/v1")
    application.include_router(system_router, prefix="/api/v1")
    application.include_router(proxies_router, prefix="/api/v1")
    application.include_router(jobs_router, prefix="/api/v1")
    application.include_router(gateway_router, prefix="/api/v1")
    application.include_router(settings_router, prefix="/api/v1")
    application.include_router(logs_router, prefix="/api/v1")
    application.include_router(pool_router, prefix="/api/v1")

    @application.exception_handler(ResourceNotFoundError)
    async def resource_not_found(
        request: Request,
        exc: ResourceNotFoundError,
    ) -> JSONResponse:
        return JSONResponse(status_code=404, content={"detail": str(exc)})

    @application.exception_handler(FreeProxyError)
    async def application_error(request: Request, exc: FreeProxyError) -> JSONResponse:
        return JSONResponse(status_code=400, content={"detail": str(exc)})

    application.mount("/static", StaticFiles(directory=WEB_DIR / "static"), name="static")
    if (WEB_DIST / "assets").is_dir():
        application.mount("/assets", StaticFiles(directory=WEB_DIST / "assets"), name="assets")
    templates = Jinja2Templates(directory=WEB_DIR / "templates")

    @application.get("/", response_class=HTMLResponse, include_in_schema=False)
    async def home(request: Request) -> Response:
        authorized = not resolved_settings.admin_auth_enabled or bool(
            getattr(request.state, "authorized", False)
        )
        if authorized and (WEB_DIST / "index.html").exists():
            return FileResponse(WEB_DIST / "index.html")
        return templates.TemplateResponse(
            request=request,
            name="index.html" if authorized else "login.html",
            context={"app_name": resolved_settings.app_name},
        )

    return application


app = create_app()
