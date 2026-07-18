import json
from pathlib import Path

from typer.testing import CliRunner

from free_proxy.cli import app
from free_proxy.config import get_settings

runner = CliRunner()


def test_cli_admin_config_status_and_logs(tmp_path: Path, monkeypatch: object) -> None:
    monkeypatch.setenv("FREE_PROXY_DATA_DIR", str(tmp_path))  # type: ignore[attr-defined]
    monkeypatch.setenv("FREE_PROXY_DATABASE_URL", f"sqlite+aiosqlite:///{tmp_path / 'test.db'}")  # type: ignore[attr-defined]
    monkeypatch.setenv("FREE_PROXY_ADMIN_USERNAME", "AdminUser1")  # type: ignore[attr-defined]
    monkeypatch.setenv("FREE_PROXY_ADMIN_PASSWORD", "AdminPass1")  # type: ignore[attr-defined]
    monkeypatch.setenv("FREE_PROXY_ADMIN_SECRET_PATH", "Secret123")  # type: ignore[attr-defined]
    get_settings.cache_clear()

    upgraded = runner.invoke(app, ["database-upgrade"])
    updated = runner.invoke(
        app,
        [
            "admin-config",
            "--username",
            "NewAdmin1",
            "--password",
            "NewPass123",
            "--secret-path",
            "NextPath9",
        ],
    )
    status = runner.invoke(app, ["status"])
    log_dir = tmp_path / "logs"
    log_dir.mkdir(exist_ok=True)
    (log_dir / "2026-07-17.json").write_text('{"message":"ready"}\n', encoding="utf-8")
    logs = runner.invoke(app, ["logs", "--lines", "1"])

    assert upgraded.exit_code == 0
    assert updated.exit_code == 0
    assert status.exit_code == 0
    payload = json.loads(status.stdout)
    assert payload["username"] == "NewAdmin1"
    assert payload["url"].endswith("/NextPath9/")
    assert "jobs" in payload["tables"]
    assert logs.exit_code == 0
    assert '"message":"ready"' in logs.stdout
    get_settings.cache_clear()
