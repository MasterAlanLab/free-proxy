from __future__ import annotations

from free_proxy.domain.enums import TunnelFailureCode


class OpenVpnLogParser:
    READY_MARKER = "initialization sequence completed"

    @classmethod
    def is_ready(cls, line: str) -> bool:
        return cls.READY_MARKER in line.lower()

    @staticmethod
    def failure_code(lines: list[str]) -> TunnelFailureCode:
        text = "\n".join(lines).lower()
        if any(
            pattern in text
            for pattern in (
                "cannot allocate tun",
                "cannot open tun/tap",
                "/dev/net/tun",
                "cannot ioctl",
            )
        ):
            return TunnelFailureCode.TUN_UNAVAILABLE
        if "auth_failed" in text or "authentication failed" in text:
            return TunnelFailureCode.AUTH_FAILED
        if "cannot resolve host address" in text or "resolve: host name" in text:
            return TunnelFailureCode.DNS_FAILED
        if "tls key negotiation failed" in text or "tls handshake failed" in text:
            return TunnelFailureCode.TLS_FAILED
        if any(
            pattern in text
            for pattern in (
                "permission denied",
                "need root",
                "root privileges",
                "operation not permitted",
            )
        ):
            return TunnelFailureCode.PERMISSION_DENIED
        if "connection refused" in text:
            return TunnelFailureCode.CONNECTION_REFUSED
        if any(
            pattern in text
            for pattern in ("options error", "fatal error", "unrecognized option", "option error")
        ):
            return TunnelFailureCode.CONFIG_ERROR
        if "network is unreachable" in text:
            return TunnelFailureCode.UNREACHABLE
        if "timed out" in text or "timeout" in text:
            return TunnelFailureCode.TIMEOUT
        return TunnelFailureCode.UNKNOWN

    @classmethod
    def failure_message(cls, lines: list[str]) -> str:
        code = cls.failure_code(lines)
        messages = {
            TunnelFailureCode.TUN_UNAVAILABLE: "TUN device is unavailable or not permitted",
            TunnelFailureCode.AUTH_FAILED: "The public node rejected authentication",
            TunnelFailureCode.DNS_FAILED: "The public node address could not be resolved",
            TunnelFailureCode.TLS_FAILED: "OpenVPN TLS negotiation failed",
            TunnelFailureCode.PERMISSION_DENIED: "OpenVPN requires elevated network permissions",
            TunnelFailureCode.CONFIG_ERROR: "The OpenVPN configuration contains an invalid option",
            TunnelFailureCode.CONNECTION_REFUSED: "The public node refused the connection",
            TunnelFailureCode.UNREACHABLE: "The public node is unreachable",
            TunnelFailureCode.TIMEOUT: "OpenVPN connection timed out",
            TunnelFailureCode.UNKNOWN: "OpenVPN failed before the tunnel became ready",
        }
        return messages[code]

    @classmethod
    def is_terminal_failure(cls, line: str) -> bool:
        code = cls.failure_code([line])
        return code in {
            TunnelFailureCode.AUTH_FAILED,
            TunnelFailureCode.CONFIG_ERROR,
            TunnelFailureCode.PERMISSION_DENIED,
            TunnelFailureCode.TUN_UNAVAILABLE,
        }

    @staticmethod
    def handshake_stage(lines: list[str]) -> str:
        text = "\n".join(lines).lower()
        if "initialization sequence completed" in text:
            return "connected"
        if "auth" in text:
            return "authentication"
        if "tls" in text:
            return "tls"
        if "tun" in text or "tap" in text:
            return "interface"
        if "resolve" in text:
            return "resolving"
        return "starting"
