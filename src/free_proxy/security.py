from __future__ import annotations

import asyncio
import base64
import hashlib
import hmac
import json
import secrets
import string
import time

from pydantic import BaseModel, Field

from free_proxy.config import Settings


class AdminConfig(BaseModel):
    username: str
    password_hash: str
    secret_path: str = Field(pattern=r"^[A-Za-z0-9]+$")
    host: str
    port: int = Field(ge=1, le=65535)
    proxy_host: str = "127.0.0.1"
    proxy_port: int = Field(default=9527, ge=1, le=65535)

    @classmethod
    def from_password(
        cls,
        *,
        username: str,
        password: str,
        secret_path: str,
        host: str,
        port: int,
        proxy_host: str = "127.0.0.1",
        proxy_port: int = 9527,
    ) -> AdminConfig:
        return cls(
            username=username,
            password_hash=hash_password(password),
            secret_path=secret_path,
            host=host,
            port=port,
            proxy_host=proxy_host,
            proxy_port=proxy_port,
        )


class AdminConfigStore:
    def __init__(self, settings: Settings) -> None:
        self._settings = settings
        self.path = settings.data_dir / "web-config.json"
        self.bootstrap_path = settings.data_dir / "initial-admin-password"
        self._config = self._load_or_create()

    @property
    def config(self) -> AdminConfig:
        return self._config.model_copy(deep=True)

    def update(self, config: AdminConfig) -> None:
        self._config = config
        self._write(config)
        self.clear_bootstrap_password()

    def bootstrap_password(self) -> str | None:
        try:
            return self.bootstrap_path.read_text(encoding="utf-8").strip() or None
        except OSError:
            return None

    def clear_bootstrap_password(self) -> None:
        self.bootstrap_path.unlink(missing_ok=True)

    def _load_or_create(self) -> AdminConfig:
        if self.path.exists():
            try:
                raw = json.loads(self.path.read_text(encoding="utf-8"))
                if isinstance(raw, dict) and raw.get("password") and not raw.get("password_hash"):
                    config = AdminConfig.from_password(
                        username=str(raw.get("username") or random_credential()),
                        password=str(raw["password"]),
                        secret_path=str(raw.get("secret_path") or random_credential()),
                        host=str(raw.get("host") or self._settings.web_host),
                        port=int(raw.get("port") or self._settings.web_port),
                        proxy_host=str(raw.get("proxy_host") or self._settings.proxy_host),
                        proxy_port=int(raw.get("proxy_port") or self._settings.proxy_port),
                    )
                    self._write(config)
                    return config
                return AdminConfig.model_validate(raw)
            except (OSError, ValueError):
                pass
        password = self._settings.admin_password or random_credential()
        config = AdminConfig.from_password(
            username=self._settings.admin_username or random_credential(),
            password=password,
            secret_path=self._settings.admin_secret_path or random_credential(),
            host=self._settings.web_host,
            port=self._settings.web_port,
            proxy_host=self._settings.proxy_host,
            proxy_port=self._settings.proxy_port,
        )
        self._write(config)
        if self._settings.admin_password is None:
            self._write_bootstrap_password(password)
        return config

    def _write(self, config: AdminConfig) -> None:
        self._settings.ensure_directories()
        temporary = self.path.with_suffix(".tmp")
        temporary.write_text(
            json.dumps(config.model_dump(), ensure_ascii=False, indent=2),
            encoding="utf-8",
        )
        temporary.replace(self.path)
        try:
            self.path.chmod(0o600)
        except OSError:
            pass

    def _write_bootstrap_password(self, password: str) -> None:
        self.bootstrap_path.write_text(password + "\n", encoding="utf-8")
        try:
            self.bootstrap_path.chmod(0o600)
        except OSError:
            pass


class SessionManager:
    def __init__(self, ttl_seconds: int) -> None:
        self._ttl_seconds = ttl_seconds
        self._sessions: dict[str, float] = {}
        self._lock = asyncio.Lock()

    async def create(self) -> str:
        token = secrets.token_hex(32)
        async with self._lock:
            self._sessions[token] = time.time() + self._ttl_seconds
        return token

    async def valid(self, token: str | None) -> bool:
        if not token:
            return False
        now = time.time()
        async with self._lock:
            expires_at = self._sessions.get(token)
            if expires_at is None or expires_at <= now:
                self._sessions.pop(token, None)
                return False
            return True

    async def remove(self, token: str | None) -> None:
        if not token:
            return
        async with self._lock:
            self._sessions.pop(token, None)

    async def clear(self) -> None:
        async with self._lock:
            self._sessions.clear()


class AuthService:
    def __init__(
        self,
        settings: Settings,
        store: AdminConfigStore,
        sessions: SessionManager,
    ) -> None:
        self.settings = settings
        self.store = store
        self.sessions = sessions

    def verify(self, username: str, password: str) -> bool:
        config = self.store.config
        valid = secrets.compare_digest(username, config.username) and verify_password(
            password, config.password_hash
        )
        if valid:
            self.store.clear_bootstrap_password()
        return valid


def hash_password(password: str) -> str:
    salt = secrets.token_bytes(16)
    digest = hashlib.scrypt(
        password.encode("utf-8"),
        salt=salt,
        n=2**14,
        r=8,
        p=1,
        dklen=32,
    )
    encoded_salt = base64.urlsafe_b64encode(salt).decode("ascii")
    encoded_digest = base64.urlsafe_b64encode(digest).decode("ascii")
    return f"scrypt$16384$8$1${encoded_salt}${encoded_digest}"


def verify_password(password: str, encoded: str) -> bool:
    try:
        algorithm, n_text, r_text, p_text, salt_text, digest_text = encoded.split("$", 5)
        if algorithm != "scrypt":
            return False
        salt = base64.urlsafe_b64decode(salt_text.encode("ascii"))
        expected = base64.urlsafe_b64decode(digest_text.encode("ascii"))
        actual = hashlib.scrypt(
            password.encode("utf-8"),
            salt=salt,
            n=int(n_text),
            r=int(r_text),
            p=int(p_text),
            dklen=len(expected),
        )
    except (ValueError, TypeError):
        return False
    return hmac.compare_digest(actual, expected)


def random_credential(length: int = 12) -> str:
    alphabet = string.ascii_letters + string.digits
    while True:
        value = "".join(secrets.choice(alphabet) for _ in range(length))
        if value[0].isalpha() and any(char.islower() for char in value) and any(
            char.isupper() for char in value
        ) and any(char.isdigit() for char in value):
            return value
