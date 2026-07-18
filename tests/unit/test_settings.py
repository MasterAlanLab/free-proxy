from pathlib import Path

import pytest

from free_proxy.config import Settings


def test_settings_build_default_sqlite_url(tmp_path: Path) -> None:
    settings = Settings(data_dir=tmp_path)

    assert settings.database_url == f"sqlite+aiosqlite:///{tmp_path / 'free-proxy.db'}"
    assert settings.proxy_host == "127.0.0.1"
    assert settings.proxy_port == 9527


def test_settings_require_complete_proxy_credentials(tmp_path: Path) -> None:
    with pytest.raises(ValueError, match="configured together"):
        Settings(data_dir=tmp_path, proxy_username="user")
