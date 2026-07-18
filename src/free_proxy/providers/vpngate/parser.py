from __future__ import annotations

import base64
import csv
import hashlib
import re
import shlex
from dataclasses import dataclass

from free_proxy.domain.enums import TransportProtocol
from free_proxy.domain.exceptions import ProviderError
from free_proxy.domain.models import DiscoveredNode


@dataclass(frozen=True, slots=True)
class VpnGateParseStats:
    total_rows: int = 0
    valid_rows: int = 0
    duplicate_rows: int = 0
    malformed_rows: int = 0
    missing_field_rows: int = 0


@dataclass(frozen=True, slots=True)
class VpnGateParseResult:
    nodes: list[DiscoveredNode]
    stats: VpnGateParseStats


def safe_int(value: object) -> int:
    try:
        return int(str(value or "0"))
    except (TypeError, ValueError):
        return 0


def parse_remote(config_text: str, fallback_host: str) -> tuple[str, int, TransportProtocol]:
    remote_host = fallback_host
    remote_port = 0
    transport = TransportProtocol.UNKNOWN

    for raw_line in config_text.splitlines():
        line = raw_line.strip()
        if not line or line.startswith(("#", ";")):
            continue
        try:
            parts = shlex.split(line, comments=False, posix=True)
        except ValueError:
            parts = line.split()
        if not parts:
            continue

        directive = parts[0].lower()
        if directive == "proto" and len(parts) >= 2:
            transport = normalize_transport(parts[1])
        elif directive == "remote" and len(parts) >= 3:
            remote_host = parts[1]
            remote_port = safe_int(parts[2])
            if len(parts) >= 4:
                transport = normalize_transport(parts[3])

    return remote_host, remote_port, transport


def normalize_transport(value: str) -> TransportProtocol:
    normalized = value.lower()
    if "tcp" in normalized:
        return TransportProtocol.TCP
    if "udp" in normalized:
        return TransportProtocol.UDP
    return TransportProtocol.UNKNOWN


def decode_config(encoded: str) -> str:
    try:
        return base64.b64decode(encoded.encode("ascii"), validate=False).decode(
            "utf-8", errors="replace"
        )
    except Exception as exc:
        raise ProviderError("VPNGate returned an invalid OpenVPN configuration") from exc


def build_node_id(
    country_code: str,
    ip_address: str,
    remote_port: int,
    transport: TransportProtocol,
) -> str:
    identity = f"vpngate:{ip_address}"
    digest = hashlib.sha256(identity.encode("utf-8")).hexdigest()[:12]
    prefix = re.sub(r"[^A-Za-z0-9]+", "", country_code.upper()) or "XX"
    return f"{prefix.lower()}-{digest}"


def provider_identity(ip_address: str) -> str:
    return f"vpngate:{ip_address.strip()}"


def parse_vpngate_response(text: str, *, limit: int = 300) -> list[DiscoveredNode]:
    return parse_vpngate_response_with_stats(text, limit=limit).nodes


def parse_vpngate_response_with_stats(text: str, *, limit: int = 300) -> VpnGateParseResult:
    lines = [line for line in text.splitlines() if line and not line.startswith("*")]
    if not lines:
        raise ProviderError("VPNGate response is empty or has no header")
    if lines[0].startswith("#"):
        lines[0] = lines[0][1:]
    if "IP" not in lines[0] or "OpenVPN_ConfigData_Base64" not in lines[0]:
        raise ProviderError("VPNGate response has no usable header")

    nodes: list[DiscoveredNode] = []
    seen_identities: set[str] = set()
    total = valid = duplicates = malformed = missing = 0
    for row in csv.DictReader(lines):
        total += 1
        if len(nodes) >= limit:
            break
        ip_address = str(row.get("IP") or "").strip()
        encoded_config = str(row.get("OpenVPN_ConfigData_Base64") or "").strip()
        if not ip_address or not encoded_config:
            missing += 1
            continue
        try:
            config_text = decode_config(encoded_config)
            remote_host, remote_port, transport = parse_remote(config_text, ip_address)
            if not remote_port:
                raise ValueError("missing remote port")
        except (ProviderError, ValueError, UnicodeError):
            malformed += 1
            continue
        identity = provider_identity(ip_address)
        if identity in seen_identities:
            duplicates += 1
            continue
        country_code = str(row.get("CountryShort") or "").strip().upper()
        node_id = build_node_id(country_code, ip_address, remote_port, transport)

        nodes.append(
            DiscoveredNode(
                id=node_id,
                provider="vpngate",
                provider_identity=identity,
                provider_node_id=str(row.get("HostName") or ""),
                country=str(row.get("CountryLong") or ""),
                country_code=country_code,
                host_name=str(row.get("HostName") or ""),
                ip_address=ip_address,
                remote_host=remote_host,
                remote_port=remote_port,
                transport=transport,
                source_score=safe_int(row.get("Score")),
                source_ping_ms=safe_int(row.get("Ping")),
                source_speed_bps=safe_int(row.get("Speed")),
                source_sessions=safe_int(row.get("NumVpnSessions")),
                config_text=config_text,
            )
        )
        seen_identities.add(identity)
        valid += 1

    return VpnGateParseResult(
        nodes=nodes,
        stats=VpnGateParseStats(total, valid, duplicates, malformed, missing),
    )
