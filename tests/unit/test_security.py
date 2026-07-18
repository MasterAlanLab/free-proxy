import json
from collections.abc import Coroutine
from pathlib import Path
from types import SimpleNamespace
from typing import Any

import pytest

from free_proxy.api.routers import auth as auth_router
from free_proxy.api.routers.auth import CredentialsUpdate
from free_proxy.config import Settings
from free_proxy.security import AdminConfigStore, AuthService, SessionManager


def test_generated_password_is_one_time_and_hashed(tmp_path: Path) -> None:
    settings = Settings(data_dir=tmp_path)
    store = AdminConfigStore(settings)
    password = store.bootstrap_password()
    assert password is not None
    assert password not in store.path.read_text(encoding="utf-8")

    auth = AuthService(settings, store, SessionManager(60))
    assert auth.verify(store.config.username, password) is True
    assert store.bootstrap_password() is None


def test_plaintext_admin_config_is_migrated_to_scrypt(tmp_path: Path) -> None:
    settings = Settings(data_dir=tmp_path)
    settings.ensure_directories()
    path = tmp_path / "web-config.json"
    path.write_text(
        json.dumps(
            {
                "username": "LegacyAdmin",
                "password": "LegacyPass1",
                "secret_path": "Legacy123",
                "host": "127.0.0.1",
                "port": 8787,
            }
        ),
        encoding="utf-8",
    )

    store = AdminConfigStore(settings)
    auth = AuthService(settings, store, SessionManager(60))

    assert auth.verify("LegacyAdmin", "LegacyPass1") is True
    rewritten = path.read_text(encoding="utf-8")
    assert "LegacyPass1" not in rewritten
    assert "password_hash" in rewritten


@pytest.mark.asyncio
async def test_admin_network_change_schedules_process_restart(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    settings = Settings(
        data_dir=tmp_path,
        admin_username="AdminUser1",
        admin_password="AdminPass1",
        admin_secret_path="Secret123",
        allow_process_restart=True,
    )
    auth = AuthService(settings, AdminConfigStore(settings), SessionManager(60))
    request = SimpleNamespace(
        app=SimpleNamespace(state=SimpleNamespace(auth_service=auth))
    )
    scheduled: list[str | None] = []

    def fake_create_task(
        coroutine: Coroutine[Any, Any, None],
        *,
        name: str | None = None,
    ) -> None:
        scheduled.append(name)
        coroutine.close()

    monkeypatch.setattr(auth_router.asyncio, "create_task", fake_create_task)

    result = await auth_router.update_credentials(
        CredentialsUpdate(
            username="AdminUser1",
            password="",
            secret_path="NewSecret9",
            host="0.0.0.0",
            port=9999,
        ),
        request,  # type: ignore[arg-type]
    )

    assert result["restart_needed"] is True
    assert scheduled == ["restart-after-admin-config-update"]
