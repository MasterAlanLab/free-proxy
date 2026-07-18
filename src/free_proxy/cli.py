from __future__ import annotations

import asyncio
import json
import os
from pathlib import Path
from typing import Annotated, Any

import typer
import uvicorn
from alembic.config import Config

from alembic import command
from free_proxy.config import Settings, get_settings
from free_proxy.infrastructure.database import Database, ProxyNodeRepository
from free_proxy.providers.vpngate import VpnGateProvider
from free_proxy.security import AdminConfig, AdminConfigStore, hash_password
from free_proxy.services.diagnostics import NetworkDiagnosticsService
from free_proxy.services.discovery import DiscoveryService

app = typer.Typer(no_args_is_help=True, help="Free proxy discovery and SOCKS5 gateway")
PROJECT_ROOT = Path(__file__).resolve().parents[2]


@app.command()
def serve(
    host: Annotated[str | None, typer.Option(help="Web bind address")] = None,
    port: Annotated[int | None, typer.Option(help="Web bind port")] = None,
    reload: Annotated[bool, typer.Option(help="Enable development reload")] = False,
) -> None:
    settings = get_settings()
    admin_config = AdminConfigStore(settings).config
    resolved_host = host or admin_config.host
    resolved_port = port or admin_config.port
    if reload:
        os.environ["FREE_PROXY_WEB_HOST"] = resolved_host
        os.environ["FREE_PROXY_WEB_PORT"] = str(resolved_port)
        get_settings.cache_clear()
        application: Any = "free_proxy.main:app"
    else:
        from free_proxy.main import create_app

        application = create_app(
            settings.model_copy(
                update={
                    "web_host": resolved_host,
                    "web_port": resolved_port,
                    "proxy_host": admin_config.proxy_host,
                    "proxy_port": admin_config.proxy_port,
                }
            )
        )
    uvicorn.run(
        application,
        host=resolved_host,
        port=resolved_port,
        reload=reload,
    )


@app.command()
def discover() -> None:
    """Fetch nodes from the configured public provider and store them."""

    async def run() -> None:
        settings = get_settings()
        settings.ensure_directories()
        database = Database(settings)
        provider = VpnGateProvider(settings)
        try:
            await database.initialize()
            repository = ProxyNodeRepository(database.session_factory)
            result = await DiscoveryService(provider, repository).discover()
            typer.echo(json.dumps(result.model_dump(mode="json"), ensure_ascii=False, indent=2))
        finally:
            await provider.close()
            await database.dispose()

    asyncio.run(run())


@app.command()
def credentials() -> None:
    """Print the current Web management address and credentials."""
    settings = get_settings()
    store = AdminConfigStore(settings)
    config = store.config
    typer.echo(f"URL: http://{config.host}:{config.port}/{config.secret_path}/")
    typer.echo(f"Username: {config.username}")
    bootstrap = store.bootstrap_password()
    typer.echo(f"Password: {bootstrap or '[configured; cannot be recovered]'}")


@app.command()
def status() -> None:
    """Print local configuration and database schema status."""
    settings = get_settings()
    store = AdminConfigStore(settings)

    async def inspect_database() -> set[str]:
        database = Database(settings)
        try:
            return await database.schema_tables()
        finally:
            await database.dispose()

    config = store.config
    payload = {
        "environment": settings.environment,
        "url": f"http://{config.host}:{config.port}/{config.secret_path}/",
        "username": config.username,
        "bootstrap_password_available": store.bootstrap_password() is not None,
        "database_url": settings.database_url,
        "tables": sorted(asyncio.run(inspect_database())),
        "proxy": f"{settings.proxy_host}:{settings.proxy_port}",
    }
    typer.echo(json.dumps(payload, ensure_ascii=False, indent=2))


@app.command()
def preflight() -> None:
    """Run startup environment checks without preventing the Web control plane from starting."""

    async def run() -> None:
        result = await NetworkDiagnosticsService(get_settings()).startup_preflight()
        typer.echo(json.dumps(result.model_dump(mode="json"), ensure_ascii=False, indent=2))

    asyncio.run(run())


@app.command()
def logs(
    lines: Annotated[int, typer.Option(min=1, max=5000)] = 100,
) -> None:
    """Print the most recent structured application log entries."""
    settings = get_settings()
    paths = sorted((settings.data_dir / "logs").glob("*.json"))
    if not paths:
        return
    entries = paths[-1].read_text(encoding="utf-8").splitlines()[-lines:]
    typer.echo("\n".join(entries))


@app.command("admin-config")
def admin_config(
    username: Annotated[str | None, typer.Option()] = None,
    password: Annotated[str | None, typer.Option(prompt=False, hide_input=True)] = None,
    secret_path: Annotated[str | None, typer.Option()] = None,
    host: Annotated[str | None, typer.Option()] = None,
    port: Annotated[int | None, typer.Option(min=1, max=65535)] = None,
    proxy_host: Annotated[str | None, typer.Option()] = None,
    proxy_port: Annotated[int | None, typer.Option(min=1, max=65535)] = None,
) -> None:
    """Update Web administration credentials and listener settings."""
    settings = get_settings()
    store = AdminConfigStore(settings)
    previous = store.config
    updated = AdminConfig(
        username=username or previous.username,
        password_hash=hash_password(password) if password else previous.password_hash,
        secret_path=secret_path or previous.secret_path,
        host=host or previous.host,
        port=port or previous.port,
        proxy_host=proxy_host or previous.proxy_host,
        proxy_port=proxy_port or previous.proxy_port,
    )
    store.update(updated)
    typer.echo("Administration configuration updated; restart the service if the listener changed")


@app.command("database-upgrade")
def database_upgrade() -> None:
    """Upgrade the application database to the latest Alembic revision."""
    run_database_upgrade(get_settings())
    typer.echo("Database upgraded to head")


def run_database_upgrade(settings: Settings) -> None:
    """Run migrations."""
    config = Config(str(PROJECT_ROOT / "alembic.ini"))
    config.set_main_option("script_location", str(PROJECT_ROOT / "alembic"))
    config.set_main_option("sqlalchemy.url", settings.database_url or "")
    command.upgrade(config, "head")
