package domain

type AdminSettings struct {
	Username          string `json:"username"`
	PasswordHash      string `json:"-"`
	SecretPath        string `json:"secret_path"`
	SessionTTLSeconds int    `json:"session_ttl_seconds"`
	WebPort           int    `json:"web_port"`
	WebExternalAccess bool   `json:"web_external_access"`
	PasswordSet       bool   `json:"password_set"`
}

type ProxyServiceSettings struct {
	Enabled               bool    `json:"enabled"`
	Port                  int     `json:"port"`
	Username              string  `json:"username"`
	PasswordHash          string  `json:"-"`
	PasswordSet           bool    `json:"password_set"`
	ExternalAccess        bool    `json:"external_access"`
	MaxConnections        int     `json:"max_connections"`
	ConnectTimeoutSeconds float64 `json:"connect_timeout_seconds"`
	IdleTimeoutSeconds    float64 `json:"idle_timeout_seconds"`
	DNSServer             string  `json:"dns_server"`
}

type DiscoverySettings struct {
	VPNGateAPIURL      string  `json:"vpngate_api_url"`
	DiscoveryLimit     int     `json:"discovery_limit"`
	RequestTimeoutSecs float64 `json:"request_timeout_seconds"`
	IPInfoAPIURL       string  `json:"ip_info_api_url"`
	IPInfoCacheSeconds int     `json:"ip_info_cache_seconds"`
}

type MaintenanceSettings struct {
	Enabled                      bool    `json:"enabled"`
	MaintenanceIntervalSeconds   float64 `json:"maintenance_interval_seconds"`
	HealthCheckIntervalSeconds   float64 `json:"health_check_interval_seconds"`
	ActivePingIntervalSeconds    float64 `json:"active_ping_interval_seconds"`
	DisconnectedRetrySeconds     float64 `json:"disconnected_retry_seconds"`
	MaxProbeConcurrency          int     `json:"max_probe_concurrency"`
	InitialConnectTestLimit      int     `json:"initial_connect_test_limit"`
	ManualTestNodeLimit          int     `json:"manual_test_node_limit"`
	OpenVPNTestTimeoutSeconds    float64 `json:"openvpn_test_timeout_seconds"`
	OpenVPNConnectTimeoutSeconds float64 `json:"openvpn_connect_timeout_seconds"`
	InvalidBackoffSeconds        int     `json:"invalid_backoff_seconds"`
	StaleNodeGraceSeconds        int     `json:"stale_node_grace_seconds"`
}

type NetworkSettings struct {
	DNSRepairEnabled            bool    `json:"dns_repair_enabled"`
	DNSRepairServers            string  `json:"dns_repair_servers"`
	RoutingSetupRetries         int     `json:"routing_setup_retries"`
	RoutingRetryIntervalSeconds float64 `json:"routing_retry_interval_seconds"`
	RoutingStrictRPFilter       bool    `json:"routing_strict_rp_filter"`
}

type AppSettings struct {
	Admin       AdminSettings        `json:"admin"`
	Proxy       ProxyServiceSettings `json:"proxy"`
	Discovery   DiscoverySettings    `json:"discovery"`
	Maintenance MaintenanceSettings  `json:"maintenance"`
	Network     NetworkSettings      `json:"network"`
}
