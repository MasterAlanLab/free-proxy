from pathlib import Path

from fastapi.testclient import TestClient

from free_proxy.config import Settings
from free_proxy.main import create_app


def test_secret_path_login_and_logout(tmp_path: Path) -> None:
    settings = Settings(
        data_dir=tmp_path,
        database_url=f"sqlite+aiosqlite:///{tmp_path / 'test.db'}",
        proxy_enabled=False,
        maintenance_enabled=False,
        allow_process_restart=False,
        admin_auth_enabled=True,
        admin_username="AdminUser1",
        admin_password="AdminPass1",
        admin_secret_path="Secret123",
    )

    with TestClient(create_app(settings)) as client:
        assert client.get("/").status_code == 404
        login_page = client.get("/Secret123/")
        unauthorized = client.get("/Secret123/api/v1/system/status")
        bad_login = client.post(
            "/Secret123/api/v1/auth/login",
            json={"username": "AdminUser1", "password": "wrong"},
        )
        login = client.post(
            "/Secret123/api/v1/auth/login",
            json={"username": "AdminUser1", "password": "AdminPass1"},
        )
        authorized = client.get("/Secret123/api/v1/system/status")
        credentials = client.put(
            "/Secret123/api/v1/auth/credentials",
            json={
                "username": "NewAdmin1",
                "password": "NewPass123",
                "secret_path": "NextPath9",
                "host": "127.0.0.1",
                "port": 8787,
            },
        )
        old_path = client.get("/Secret123/api/v1/system/status")
        invalidated = client.get("/NextPath9/api/v1/system/status")
        relogin = client.post(
            "/NextPath9/api/v1/auth/login",
            json={"username": "NewAdmin1", "password": "NewPass123"},
        )
        logout = client.post("/NextPath9/api/v1/auth/logout")
        after_logout = client.get("/NextPath9/api/v1/system/status")

    assert login_page.status_code == 200
    assert "管理登录" in login_page.text
    assert unauthorized.status_code == 401
    assert bad_login.status_code == 403
    assert login.status_code == 200
    assert authorized.status_code == 200
    assert credentials.status_code == 200
    assert credentials.json()["reauth_required"] is True
    assert credentials.json()["restart_needed"] is True
    assert old_path.status_code == 404
    assert invalidated.status_code == 401
    assert relogin.status_code == 200
    assert logout.status_code == 200
    assert after_logout.status_code == 401
    config_text = (tmp_path / "web-config.json").read_text(encoding="utf-8")
    assert "AdminPass1" not in config_text
    assert "NewPass123" not in config_text
    assert "password_hash" in config_text
