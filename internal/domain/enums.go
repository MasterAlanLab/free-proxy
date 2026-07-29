// Package domain holds the pure data types and enumerations shared across the
// application. It has no dependencies on other internal packages. Enum string
// values are identical to the former Python implementation so DB rows and API
// payloads remain compatible.
package domain

type IpType string

const (
	IpResidential IpType = "residential"
	IpMobile      IpType = "mobile"
	IpHosting     IpType = "hosting"
	IpUnknown     IpType = "unknown"
)

type NodeStatus string

const (
	NodeDiscovered  NodeStatus = "discovered"
	NodeProbing     NodeStatus = "probing"
	NodeReady       NodeStatus = "ready"
	NodeUnavailable NodeStatus = "unavailable"
	NodeCooldown    NodeStatus = "cooldown"
)

type TransportProtocol string

const (
	TransportTCP     TransportProtocol = "tcp"
	TransportUDP     TransportProtocol = "udp"
	TransportUnknown TransportProtocol = "unknown"
)

type JobStatus string

const (
	JobPending   JobStatus = "pending"
	JobRunning   JobStatus = "running"
	JobSucceeded JobStatus = "succeeded"
	JobFailed    JobStatus = "failed"
	JobCancelled JobStatus = "cancelled"
)

type ProxyPolicyMode string

const (
	PolicyAuto ProxyPolicyMode = "auto"
	// PolicySpeedFirst prefers nodes with the highest advertised throughput.
	// PolicyAuto remains the latency-first mode for backward compatibility.
	PolicySpeedFirst       ProxyPolicyMode = "speed_first"
	PolicySmart            ProxyPolicyMode = "smart"
	PolicyResidentialFirst ProxyPolicyMode = "residential_first"
	PolicyCountry          ProxyPolicyMode = "country"
	PolicyFixed            ProxyPolicyMode = "fixed"
	PolicyFavorites        ProxyPolicyMode = "favorites"
)

type RoutingIpType string

const (
	RoutingAll         RoutingIpType = "all"
	RoutingResidential RoutingIpType = "residential"
	RoutingHosting     RoutingIpType = "hosting"
)

type TunnelStatus string

const (
	TunnelIdle      TunnelStatus = "idle"
	TunnelStarting  TunnelStatus = "starting"
	TunnelConnected TunnelStatus = "connected"
	TunnelFailed    TunnelStatus = "failed"
	TunnelStopped   TunnelStatus = "stopped"
)

type TunnelFailureCode string

const (
	FailCommandNotFound   TunnelFailureCode = "command_not_found"
	FailStartFailed       TunnelFailureCode = "start_failed"
	FailTunUnavailable    TunnelFailureCode = "tun_unavailable"
	FailAuthFailed        TunnelFailureCode = "auth_failed"
	FailDNSFailed         TunnelFailureCode = "dns_failed"
	FailTLSFailed         TunnelFailureCode = "tls_failed"
	FailUnreachable       TunnelFailureCode = "unreachable"
	FailPermissionDenied  TunnelFailureCode = "permission_denied"
	FailConfigError       TunnelFailureCode = "config_error"
	FailConnectionRefused TunnelFailureCode = "connection_refused"
	FailTimeout           TunnelFailureCode = "timeout"
	FailUnknown           TunnelFailureCode = "unknown"
)
