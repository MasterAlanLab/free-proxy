from enum import StrEnum


class IpType(StrEnum):
    RESIDENTIAL = "residential"
    MOBILE = "mobile"
    HOSTING = "hosting"
    UNKNOWN = "unknown"


class NodeStatus(StrEnum):
    DISCOVERED = "discovered"
    PROBING = "probing"
    READY = "ready"
    UNAVAILABLE = "unavailable"
    COOLDOWN = "cooldown"


class TransportProtocol(StrEnum):
    TCP = "tcp"
    UDP = "udp"
    UNKNOWN = "unknown"


class JobStatus(StrEnum):
    PENDING = "pending"
    RUNNING = "running"
    SUCCEEDED = "succeeded"
    FAILED = "failed"
    CANCELLED = "cancelled"


class ProxyPolicyMode(StrEnum):
    AUTO = "auto"
    RESIDENTIAL_FIRST = "residential_first"
    COUNTRY = "country"
    FIXED = "fixed"
    FAVORITES = "favorites"


class RoutingIpType(StrEnum):
    ALL = "all"
    RESIDENTIAL = "residential"
    HOSTING = "hosting"


class TunnelStatus(StrEnum):
    IDLE = "idle"
    STARTING = "starting"
    CONNECTED = "connected"
    FAILED = "failed"
    STOPPED = "stopped"


class TunnelFailureCode(StrEnum):
    COMMAND_NOT_FOUND = "command_not_found"
    START_FAILED = "start_failed"
    TUN_UNAVAILABLE = "tun_unavailable"
    AUTH_FAILED = "auth_failed"
    DNS_FAILED = "dns_failed"
    TLS_FAILED = "tls_failed"
    UNREACHABLE = "unreachable"
    PERMISSION_DENIED = "permission_denied"
    CONFIG_ERROR = "config_error"
    CONNECTION_REFUSED = "connection_refused"
    TIMEOUT = "timeout"
    UNKNOWN = "unknown"
