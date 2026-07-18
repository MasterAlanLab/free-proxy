from __future__ import annotations

from functools import lru_cache
from pathlib import Path

from pydantic import Field, model_validator
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_prefix="FREE_PROXY_",
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
    )

    app_name: str = "Free Proxy"
    environment: str = "development"
    data_dir: Path = Field(default=Path("free_proxy_data"))
    database_url: str | None = None
    sql_echo: bool = False

    web_host: str = "127.0.0.1"
    web_port: int = Field(default=8787, ge=1, le=65535)
    admin_auth_enabled: bool = True
    admin_username: str | None = None
    admin_password: str | None = None
    admin_secret_path: str | None = None
    session_ttl_seconds: int = Field(default=30 * 24 * 3600, ge=60)
    allow_process_restart: bool = True
    proxy_host: str = "127.0.0.1"
    proxy_port: int = Field(default=9527, ge=1, le=65535)
    proxy_enabled: bool = True
    proxy_username: str | None = None
    proxy_password: str | None = None
    proxy_max_connections: int = Field(default=256, ge=1, le=10000)
    proxy_connect_timeout_seconds: float = Field(default=20.0, gt=0, le=120)
    proxy_idle_timeout_seconds: float = Field(default=120.0, gt=0, le=3600)
    proxy_dns_server: str = "8.8.8.8"
    dns_repair_enabled: bool = False
    dns_repair_servers: str = "1.1.1.1,8.8.8.8"

    openvpn_command: str = "openvpn"
    openvpn_username: str = "vpn"
    openvpn_password: str = "vpn"
    openvpn_test_timeout_seconds: float = Field(default=15.0, gt=0, le=120)
    openvpn_connect_timeout_seconds: float = Field(default=35.0, gt=0, le=180)
    tunnel_interface: str = "tun0"
    test_tun_start: int = Field(default=2, ge=1, le=999)
    test_tun_end: int = Field(default=99, ge=1, le=999)
    max_probe_concurrency: int = Field(default=5, ge=1, le=50)
    policy_routing_table: int = Field(default=100, ge=1, le=2_147_483_647)

    vpngate_api_url: str = "https://www.vpngate.net/api/iphone/"
    discovery_limit: int = Field(default=300, ge=1, le=5000)
    request_timeout_seconds: float = Field(default=15.0, gt=0, le=120)
    upstream_proxy_url: str | None = None
    upstream_direct_fallback: bool = True
    ip_info_api_url: str = (
        "http://ip-api.com/batch?lang=zh-CN&fields="
        "status,message,query,country,regionName,city,isp,org,as,asname,proxy,hosting,mobile"
    )
    ip_info_cache_seconds: int = Field(default=7 * 24 * 3600, ge=0, le=30 * 24 * 3600)
    health_check_interval_seconds: float = Field(default=30.0, gt=0, le=3600)
    active_ping_interval_seconds: float = Field(default=10.0, gt=0, le=3600)
    maintenance_interval_seconds: float = Field(default=1260.0, gt=0, le=86400)
    disconnected_retry_seconds: float = Field(default=30.0, gt=0, le=3600)
    maintenance_enabled: bool = True
    initial_connect_test_limit: int = Field(default=10, ge=1, le=50)
    manual_test_node_limit: int = Field(default=5, ge=1, le=20)
    invalid_backoff_seconds: int = Field(default=1800, ge=1, le=86400)
    routing_setup_retries: int = Field(default=3, ge=1, le=10)
    routing_retry_interval_seconds: float = Field(default=1.0, ge=0, le=60)
    routing_strict_rp_filter: bool = False
    stale_node_grace_seconds: int = Field(default=7 * 24 * 3600, ge=0)

    @model_validator(mode="after")
    def normalize_paths(self) -> Settings:
        if self.test_tun_start > self.test_tun_end:
            raise ValueError("test_tun_start must not exceed test_tun_end")
        if (self.proxy_username is None) != (self.proxy_password is None):
            raise ValueError("proxy_username and proxy_password must be configured together")
        self.data_dir = self.data_dir.expanduser().resolve()
        if self.database_url is None:
            database_path = self.data_dir / "free-proxy.db"
            self.database_url = f"sqlite+aiosqlite:///{database_path}"
        return self

    def ensure_directories(self) -> None:
        self.data_dir.mkdir(parents=True, exist_ok=True)
        (self.data_dir / "configs").mkdir(parents=True, exist_ok=True)
        (self.data_dir / "logs").mkdir(parents=True, exist_ok=True)

    @property
    def proxy_auth_enabled(self) -> bool:
        return self.proxy_username is not None or self.proxy_password is not None

    @property
    def parsed_dns_repair_servers(self) -> list[str]:
        return [server.strip() for server in self.dns_repair_servers.split(",") if server.strip()]


@lru_cache
def get_settings() -> Settings:
    return Settings()
