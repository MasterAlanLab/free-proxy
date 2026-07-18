from free_proxy.api.routers.gateway import router as gateway_router
from free_proxy.api.routers.jobs import router as jobs_router
from free_proxy.api.routers.logs import router as logs_router
from free_proxy.api.routers.pool import router as pool_router
from free_proxy.api.routers.proxies import router as proxies_router
from free_proxy.api.routers.settings import router as settings_router
from free_proxy.api.routers.system import router as system_router

__all__ = [
    "auth_router",
    "gateway_router",
    "jobs_router",
    "logs_router",
    "pool_router",
    "proxies_router",
    "settings_router",
    "system_router",
]
from free_proxy.api.routers.auth import router as auth_router
