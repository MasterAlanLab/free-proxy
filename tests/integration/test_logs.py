import logging
from datetime import datetime, timedelta
from pathlib import Path

from fastapi.testclient import TestClient

from free_proxy.config import Settings
from free_proxy.logging import JsonLogStore
from free_proxy.main import create_app


def test_logs_can_be_filtered_and_exported(tmp_path: Path) -> None:
    settings = Settings(
        data_dir=tmp_path,
        database_url=f"sqlite+aiosqlite:///{tmp_path / 'test.db'}",
        proxy_enabled=False,
        maintenance_enabled=False,
        admin_auth_enabled=False,
        allow_process_restart=False,
    )

    with TestClient(create_app(settings)) as client:
        store = client.app.state.log_store
        store.emit(logging.LogRecord("free_proxy.health", logging.WARNING, "", 0, "down", (), None))
        store.emit(logging.LogRecord("free_proxy.jobs", logging.INFO, "", 0, "ready", (), None))

        filtered = client.get("/api/v1/logs", params={"level": "WARNING", "module": "health"})
        exported = client.get("/api/v1/logs/export", params={"module": "jobs"})

    assert filtered.status_code == 200
    assert [entry["message"] for entry in filtered.json()["logs"]] == ["down"]
    assert exported.status_code == 200
    assert exported.headers["content-type"].startswith("application/x-ndjson")
    assert 'filename="free-proxy-today.jsonl"' in exported.headers["content-disposition"]
    assert '"message": "ready"' in exported.text
    assert '"message": "down"' not in exported.text


def test_log_store_skips_corruption_and_removes_three_day_old_files(tmp_path: Path) -> None:
    settings = Settings(data_dir=tmp_path)
    store = JsonLogStore(settings)
    today = datetime.now().astimezone().date()
    today_path = tmp_path / "logs" / f"{today:%Y-%m-%d}.json"
    today_path.write_text(
        'not-json\n{"level":"INFO","module":"jobs","message":"valid"}\n',
        encoding="utf-8",
    )
    expired = tmp_path / "logs" / f"{today - timedelta(days=3):%Y-%m-%d}.json"
    retained = tmp_path / "logs" / f"{today - timedelta(days=2):%Y-%m-%d}.json"
    expired.write_text("{}\n", encoding="utf-8")
    retained.write_text("{}\n", encoding="utf-8")

    assert [entry["message"] for entry in store.read()] == ["valid"]
    store.cleanup()

    assert expired.exists() is False
    assert retained.exists() is True
