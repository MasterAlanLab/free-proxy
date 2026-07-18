from __future__ import annotations

from http.cookies import SimpleCookie

from starlette.responses import JSONResponse, RedirectResponse
from starlette.types import ASGIApp, Receive, Scope, Send

from free_proxy.security import AuthService


class AdminAccessMiddleware:
    def __init__(self, app: ASGIApp, auth: AuthService) -> None:
        self.app = app
        self.auth = auth

    async def __call__(self, scope: Scope, receive: Receive, send: Send) -> None:
        if scope["type"] != "http" or not self.auth.settings.admin_auth_enabled:
            await self.app(scope, receive, send)
            return

        config = self.auth.store.config
        prefix = f"/{config.secret_path}"
        path = str(scope.get("path") or "/")
        if path == prefix:
            await RedirectResponse(f"{prefix}/", status_code=307)(scope, receive, send)
            return
        if not path.startswith(f"{prefix}/"):
            await JSONResponse({"detail": "Not found"}, status_code=404)(scope, receive, send)
            return

        stripped_path = path[len(prefix) :] or "/"
        scope["path"] = stripped_path
        scope["raw_path"] = stripped_path.encode()
        scope["root_path"] = f"{scope.get('root_path', '')}{prefix}"
        state = scope.setdefault("state", {})
        token = session_cookie(scope)
        authorized = await self.auth.sessions.valid(token)
        state["authorized"] = authorized
        state["session_token"] = token

        public_path = stripped_path in ("/", "/api/v1/auth/login") or stripped_path.startswith(
            "/static/"
        )
        if not authorized and not public_path:
            await JSONResponse({"detail": "Unauthorized"}, status_code=401)(scope, receive, send)
            return
        await self.app(scope, receive, send)


def session_cookie(scope: Scope) -> str | None:
    headers = dict(scope.get("headers") or [])
    raw_cookie = headers.get(b"cookie")
    if not raw_cookie:
        return None
    cookie = SimpleCookie()
    try:
        cookie.load(raw_cookie.decode("latin-1"))
    except ValueError:
        return None
    morsel = cookie.get("session")
    return morsel.value if morsel is not None else None
