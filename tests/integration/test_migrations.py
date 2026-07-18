import sqlite3
from pathlib import Path

from alembic.config import Config

from alembic import command
from free_proxy.cli import PROJECT_ROOT
from free_proxy.config import Settings


def test_alembic_upgrade_and_downgrade(tmp_path: Path) -> None:
    database_path = tmp_path / "test.db"
    settings = Settings(
        data_dir=tmp_path,
        database_url=f"sqlite+aiosqlite:///{database_path}",
    )
    config = migration_config(settings)

    command.upgrade(config, "head")
    assert version(database_path) == "20260717_0002"
    assert {"proxy_nodes", "jobs", "probe_results"} <= tables(database_path)

    command.downgrade(config, "base")
    assert "proxy_nodes" not in tables(database_path)


def migration_config(settings: Settings) -> Config:
    config = Config(str(PROJECT_ROOT / "alembic.ini"))
    config.set_main_option("script_location", str(PROJECT_ROOT / "alembic"))
    config.set_main_option("sqlalchemy.url", settings.database_url or "")
    return config


def tables(path: Path) -> set[str]:
    with sqlite3.connect(path) as connection:
        return {
            str(row[0])
            for row in connection.execute(
                "SELECT name FROM sqlite_master WHERE type = 'table'"
            )
        }


def version(path: Path) -> str:
    with sqlite3.connect(path) as connection:
        row = connection.execute("SELECT version_num FROM alembic_version").fetchone()
    assert row is not None
    return str(row[0])
