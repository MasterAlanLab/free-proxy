package domain

import "time"

// DiscoveredNode is a raw candidate parsed from a provider before probing.
type DiscoveredNode struct {
	ID               string            `json:"id"`
	Provider         string            `json:"provider"`
	ProviderNodeID   string            `json:"provider_node_id"`
	ProviderIdentity string            `json:"provider_identity"`
	Country          string            `json:"country"`
	CountryCode      string            `json:"country_code"`
	HostName         string            `json:"host_name"`
	IPAddress        string            `json:"ip_address"`
	RemoteHost       string            `json:"remote_host"`
	RemotePort       int               `json:"remote_port"`
	Transport        TransportProtocol `json:"transport"`
	SourceScore      int               `json:"source_score"`
	SourcePingMS     int               `json:"source_ping_ms"`
	SourceSpeedBPS   int64             `json:"source_speed_bps"`
	SourceSessions   int               `json:"source_sessions"`
	ConfigText       string            `json:"config_text"`
	FetchedAt        time.Time         `json:"fetched_at"`
}

// ProxyNodeRead is the API/DB view of a stored node.
type ProxyNodeRead struct {
	ID                  string            `json:"id"`
	Provider            string            `json:"provider"`
	ProviderNodeID      string            `json:"provider_node_id"`
	ProviderIdentity    string            `json:"provider_identity"`
	Country             string            `json:"country"`
	CountryCode         string            `json:"country_code"`
	HostName            string            `json:"host_name"`
	IPAddress           string            `json:"ip_address"`
	RemoteHost          string            `json:"remote_host"`
	RemotePort          int               `json:"remote_port"`
	Transport           TransportProtocol `json:"transport"`
	IPType              IpType            `json:"ip_type"`
	Owner               string            `json:"owner"`
	ASN                 string            `json:"asn"`
	ASName              string            `json:"as_name"`
	Location            string            `json:"location"`
	Quality             string            `json:"quality"`
	Status              NodeStatus        `json:"status"`
	SourceScore         int               `json:"source_score"`
	SourcePingMS        int               `json:"source_ping_ms"`
	SourceSpeedBPS      int64             `json:"source_speed_bps"`
	SourceSessions      int               `json:"source_sessions"`
	LatencyMS           int               `json:"latency_ms"`
	ConsecutiveFailures int               `json:"consecutive_failures"`
	SuccessCount        int               `json:"success_count"`
	FailureCount        int               `json:"failure_count"`
	FetchedAt           time.Time         `json:"fetched_at"`
	LastProbedAt        *time.Time        `json:"last_probed_at"`
	LastSuccessAt       *time.Time        `json:"last_success_at"`
	IPInfoUpdatedAt     *time.Time        `json:"ip_info_updated_at"`
	CooldownUntil       *time.Time        `json:"cooldown_until"`
	LastSeenAt          *time.Time        `json:"last_seen_at"`
	SourcePresent       bool              `json:"source_present"`
}

type ProxyNodePage struct {
	Items  []ProxyNodeRead `json:"items"`
	Total  int64           `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

type PoolStatistics struct {
	Total       int64 `json:"total"`
	Ready       int64 `json:"ready"`
	Discovered  int64 `json:"discovered"`
	Unavailable int64 `json:"unavailable"`
	Cooldown    int64 `json:"cooldown"`
	Residential int64 `json:"residential"`
	Mobile      int64 `json:"mobile"`
	Hosting     int64 `json:"hosting"`
	Unknown     int64 `json:"unknown"`
	Favorites   int64 `json:"favorites"`
	Blacklisted int64 `json:"blacklisted"`
	Countries   int64 `json:"countries"`
}

// ProxyNodeTarget carries the minimum needed to activate a node.
type ProxyNodeTarget struct {
	ID           string `json:"id"`
	IPAddress    string `json:"ip_address"`
	RemoteHost   string `json:"remote_host"`
	RemotePort   int    `json:"remote_port"`
	SourcePingMS int    `json:"source_ping_ms"`
	ConfigText   string `json:"config_text"`
}

type DiscoveryResult struct {
	Provider         string `json:"provider"`
	Discovered       int    `json:"discovered"`
	Stored           int    `json:"stored"`
	TotalRows        *int   `json:"total_rows,omitempty"`
	ValidRows        *int   `json:"valid_rows,omitempty"`
	DuplicateRows    *int   `json:"duplicate_rows,omitempty"`
	MalformedRows    *int   `json:"malformed_rows,omitempty"`
	MissingFieldRows *int   `json:"missing_field_rows,omitempty"`
}

type MaintenanceResult struct {
	Discovered      int     `json:"discovered"`
	Probed          int     `json:"probed"`
	Available       int     `json:"available"`
	ConnectedNodeID *string `json:"connected_node_id,omitempty"`
}

// LivenessResult reports one full-pool reachability sweep.
type LivenessResult struct {
	Checked int `json:"checked"`
	Alive   int `json:"alive"`
	Dead    int `json:"dead"`
	Deleted int `json:"deleted"`
}

type JobRead struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Status     JobStatus      `json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
	StartedAt  *time.Time     `json:"started_at"`
	FinishedAt *time.Time     `json:"finished_at"`
	Result     map[string]any `json:"result"`
	Error      *string        `json:"error"`
}

type TunnelStartResult struct {
	Success        bool               `json:"success"`
	Status         TunnelStatus       `json:"status"`
	Message        string             `json:"message"`
	StartupTimeMS  int                `json:"startup_time_ms"`
	FailureCode    *TunnelFailureCode `json:"failure_code"`
	LogTail        []string           `json:"log_tail"`
	HandshakeStage string             `json:"handshake_stage"`
}

type ProbeResult struct {
	NodeID    string            `json:"node_id"`
	Available bool              `json:"available"`
	LatencyMS int               `json:"latency_ms"`
	Tunnel    TunnelStartResult `json:"tunnel"`
	ProbedAt  time.Time         `json:"probed_at"`
}

type ProbeHistoryRead struct {
	ID        int64          `json:"id"`
	NodeID    string         `json:"node_id"`
	Available bool           `json:"available"`
	LatencyMS int            `json:"latency_ms"`
	ProbedAt  time.Time      `json:"probed_at"`
	Result    map[string]any `json:"result"`
}

type ProbeManyRequest struct {
	IDs []string `json:"ids" validate:"required,min=1,dive,required"`
}

type GatewayStatus struct {
	Running           bool           `json:"running"`
	ActiveNodeID      *string        `json:"active_node_id"`
	TunnelStatus      TunnelStatus   `json:"tunnel_status"`
	ProxyListener     string         `json:"proxy_listener"`
	SocksListener     string         `json:"socks_listener"`
	HTTPListener      string         `json:"http_listener"`
	LastError         *string        `json:"last_error"`
	ActiveLatencyMS   int            `json:"active_latency_ms"`
	ExitIP            *string        `json:"exit_ip"`
	ExitLatencyMS     int            `json:"exit_latency_ms"`
	ConnectionEnabled bool           `json:"connection_enabled"`
	MonitorStatus     map[string]any `json:"monitor_status"`
}

type IpInfo struct {
	IPAddress string `json:"ip_address"`
	Owner     string `json:"owner"`
	ASN       string `json:"asn"`
	ASName    string `json:"as_name"`
	Location  string `json:"location"`
	IPType    IpType `json:"ip_type"`
	Quality   string `json:"quality"`
}

type ProxySettings struct {
	RoutingMode       ProxyPolicyMode `json:"routing_mode"`
	ForceCountry      string          `json:"force_country"`
	RoutingIPType     RoutingIpType   `json:"routing_ip_type"`
	ConnectionEnabled bool            `json:"connection_enabled"`
	FixedNodeID       *string         `json:"fixed_node_id"`
	FavoriteNodeIDs   []string        `json:"favorite_node_ids"`
}

type ProxySettingsUpdate struct {
	RoutingMode       ProxyPolicyMode `json:"routing_mode" validate:"required"`
	ForceCountry      string          `json:"force_country"`
	RoutingIPType     RoutingIpType   `json:"routing_ip_type"`
	ConnectionEnabled bool            `json:"connection_enabled"`
	FixedNodeID       *string         `json:"fixed_node_id"`
}

type ProxyHealthResult struct {
	OK        bool    `json:"ok"`
	ExitIP    *string `json:"exit_ip"`
	LatencyMS int     `json:"latency_ms"`
	Error     *string `json:"error"`
}

type DiagnosticCheck struct {
	Name        string `json:"name"`
	OK          bool   `json:"ok"`
	Detail      string `json:"detail"`
	Severity    string `json:"severity"`
	Recoverable bool   `json:"recoverable"`
}

type SystemDiagnostics struct {
	Healthy bool              `json:"healthy"`
	Checks  []DiagnosticCheck `json:"checks"`
}

type DnsRepairResult struct {
	Repaired  bool     `json:"repaired"`
	Interface string   `json:"interface"`
	Servers   []string `json:"servers"`
	Detail    string   `json:"detail"`
}
