from __future__ import annotations

import os
import urllib.parse
from dataclasses import dataclass

from free_proxy.config import Settings


@dataclass(frozen=True, slots=True)
class UpstreamProxy:
    kind: str
    host: str
    port: int
    username: str | None = None
    password: str | None = None

    @property
    def url(self) -> str:
        auth = ""
        if self.username is not None:
            username = urllib.parse.quote(self.username, safe="")
            password = urllib.parse.quote(self.password or "", safe="")
            auth = f"{username}:{password}@"
        scheme = "socks5" if self.kind == "socks" else "http"
        host = f"[{self.host}]" if ":" in self.host else self.host
        return f"{scheme}://{auth}{host}:{self.port}"


def get_upstream_proxy(settings: Settings) -> UpstreamProxy | None:
    candidates = [
        settings.upstream_proxy_url,
        os.environ.get("http_proxy"),
        os.environ.get("HTTP_PROXY"),
        os.environ.get("https_proxy"),
        os.environ.get("HTTPS_PROXY"),
    ]
    for value in candidates:
        if value:
            parsed = parse_upstream_proxy(value)
            if parsed is not None:
                return parsed
    return None


def parse_upstream_proxy(
    value: str,
    *,
    forced_kind: str | None = None,
) -> UpstreamProxy | None:
    normalized = value.strip()
    if not normalized:
        return None
    if "://" not in normalized:
        normalized = f"{forced_kind or 'http'}://{normalized}"
    parsed = urllib.parse.urlsplit(normalized)
    if not parsed.hostname:
        return None
    kind = forced_kind or ("socks" if parsed.scheme.lower().startswith("socks") else "http")
    # The legacy OpenVPN variables used a SOCKS endpoint on 10808 when no port
    # was specified; explicit URLs retain protocol-specific conventional ports.
    default_port = 1080 if kind == "socks" else 8080
    username = urllib.parse.unquote(parsed.username) if parsed.username is not None else None
    password = urllib.parse.unquote(parsed.password or "") if username is not None else None
    return UpstreamProxy(
        kind=kind,
        host=parsed.hostname,
        port=parsed.port or default_port,
        username=username,
        password=password,
    )
