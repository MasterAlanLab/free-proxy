// Package config loads runtime configuration from FREE_PROXY_-prefixed
// environment variables. It mirrors the semantics of the former Python
// pydantic-settings module so the external env contract is unchanged.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config holds all runtime settings. Field env tags omit the FREE_PROXY_
// prefix; it is applied globally in Load via env.Options.Prefix.
type Config struct {
	AppName     string `env:"APP_NAME" envDefault:"Free Proxy"`
	Environment string `env:"ENVIRONMENT" envDefault:"development"`
	DataDir     string `env:"DATA_DIR" envDefault:"free_proxy_data"`
	DatabaseURL string `env:"DATABASE_URL"`
	SQLEcho     bool   `env:"SQL_ECHO" envDefault:"false"`

	// Web/proxy fields keep former env tags solely for one-time upgrade import.
	// SQLite is authoritative after app_metadata records that import.
	WebHost             string `env:"WEB_HOST" envDefault:"0.0.0.0"`
	WebPort             int    `env:"WEB_PORT" envDefault:"39527"`
	AdminAuthEnabled    bool   `env:"ADMIN_AUTH_ENABLED" envDefault:"true"`
	AdminUsername       string `env:"ADMIN_USERNAME"`
	AdminPassword       string `env:"ADMIN_PASSWORD"`
	AdminSecretPath     string `env:"ADMIN_SECRET_PATH"`
	SessionTTLSeconds   int    `env:"SESSION_TTL_SECONDS" envDefault:"2592000"`
	AllowProcessRestart bool   `env:"ALLOW_PROCESS_RESTART" envDefault:"true"`

	ProxyHost               string  `env:"PROXY_HOST" envDefault:"0.0.0.0"`
	ProxyPort               int     `env:"PROXY_PORT" envDefault:"9527"`
	ProxyEnabled            bool    `env:"PROXY_ENABLED" envDefault:"true"`
	ProxyUsername           string  `env:"PROXY_USERNAME"`
	ProxyPassword           string  `env:"PROXY_PASSWORD"`
	ProxyMaxConnections     int     `env:"PROXY_MAX_CONNECTIONS" envDefault:"256"`
	ProxyConnectTimeoutSecs float64 `env:"PROXY_CONNECT_TIMEOUT_SECONDS" envDefault:"20"`
	ProxyIdleTimeoutSecs    float64 `env:"PROXY_IDLE_TIMEOUT_SECONDS" envDefault:"120"`
	ProxyDNSServer          string  `env:"PROXY_DNS_SERVER" envDefault:"8.8.8.8"`
	DNSRepairEnabled        bool    `env:"DNS_REPAIR_ENABLED" envDefault:"false"`
	DNSRepairServers        string  `env:"DNS_REPAIR_SERVERS" envDefault:"1.1.1.1,8.8.8.8"`

	// Machine-level OpenVPN/TUN values intentionally remain environment-backed.
	OpenVPNCommand            string  `env:"OPENVPN_COMMAND" envDefault:"openvpn"`
	OpenVPNUsername           string  `env:"OPENVPN_USERNAME" envDefault:"vpn"`
	OpenVPNPassword           string  `env:"OPENVPN_PASSWORD" envDefault:"vpn"`
	OpenVPNTestTimeoutSecs    float64 `env:"OPENVPN_TEST_TIMEOUT_SECONDS" envDefault:"15"`
	OpenVPNConnectTimeoutSecs float64 `env:"OPENVPN_CONNECT_TIMEOUT_SECONDS" envDefault:"35"`
	TunnelInterface           string  `env:"TUNNEL_INTERFACE" envDefault:"tun0"`
	TestTunStart              int     `env:"TEST_TUN_START" envDefault:"2"`
	TestTunEnd                int     `env:"TEST_TUN_END" envDefault:"99"`
	MaxProbeConcurrency       int     `env:"MAX_PROBE_CONCURRENCY" envDefault:"5"`
	PolicyRoutingTable        int     `env:"POLICY_ROUTING_TABLE" envDefault:"100"`

	// Discovery fields are legacy import inputs; SQLite overwrites them at run time.
	VPNGateAPIURL      string  `env:"VPNGATE_API_URL" envDefault:"https://www.vpngate.net/api/iphone/"`
	DiscoveryLimit     int     `env:"DISCOVERY_LIMIT" envDefault:"300"`
	RequestTimeoutSecs float64 `env:"REQUEST_TIMEOUT_SECONDS" envDefault:"15"`
	IPInfoAPIURL       string  `env:"IP_INFO_API_URL" envDefault:"http://ip-api.com/batch?lang=zh-CN&fields=status,message,query,country,regionName,city,isp,org,as,asname,proxy,hosting,mobile"`
	IPInfoCacheSeconds int     `env:"IP_INFO_CACHE_SECONDS" envDefault:"604800"`

	// Maintenance and network fields are also one-time legacy import inputs.
	HealthCheckIntervalSecs  float64 `env:"HEALTH_CHECK_INTERVAL_SECONDS" envDefault:"30"`
	ActivePingIntervalSecs   float64 `env:"ACTIVE_PING_INTERVAL_SECONDS" envDefault:"10"`
	MaintenanceIntervalSecs  float64 `env:"MAINTENANCE_INTERVAL_SECONDS" envDefault:"10800"`
	DisconnectedRetrySecs    float64 `env:"DISCONNECTED_RETRY_SECONDS" envDefault:"30"`
	MaintenanceEnabled       bool    `env:"MAINTENANCE_ENABLED" envDefault:"true"`
	InitialConnectTestLimit  int     `env:"INITIAL_CONNECT_TEST_LIMIT" envDefault:"10"`
	ManualTestNodeLimit      int     `env:"MANUAL_TEST_NODE_LIMIT" envDefault:"5"`
	InvalidBackoffSeconds    int     `env:"INVALID_BACKOFF_SECONDS" envDefault:"1800"`
	RoutingSetupRetries      int     `env:"ROUTING_SETUP_RETRIES" envDefault:"3"`
	RoutingRetryIntervalSecs float64 `env:"ROUTING_RETRY_INTERVAL_SECONDS" envDefault:"1"`
	RoutingStrictRPFilter    bool    `env:"ROUTING_STRICT_RP_FILTER" envDefault:"false"`
	StaleNodeGraceSeconds    int     `env:"STALE_NODE_GRACE_SECONDS" envDefault:"604800"`

	// PreflightStrict makes `serve` abort when a critical dependency is missing.
	PreflightStrict bool `env:"PREFLIGHT_STRICT" envDefault:"false"`
}

// Load parses the environment into a Config, applies derived defaults, and
// validates cross-field invariants.
func Load() (*Config, error) {
	loadEnvFile()
	cfg := &Config{}
	if err := env.ParseWithOptions(cfg, env.Options{Prefix: "FREE_PROXY_"}); err != nil {
		return nil, err
	}
	if err := cfg.finalize(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// loadEnvFile merges FREE_PROXY_* keys from the system env file (written by
// `free-proxy install`) into the process environment, so CLI invocations see
// the same configuration as the service. Variables already set win; the file
// path can be overridden with FREE_PROXY_ENV_FILE.
func loadEnvFile() {
	path := os.Getenv("FREE_PROXY_ENV_FILE")
	if path == "" {
		path = "/etc/free-proxy/free-proxy.env"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if !strings.HasPrefix(key, "FREE_PROXY_") {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, strings.Trim(strings.TrimSpace(value), `"'`))
		}
	}
}

func (c *Config) finalize() error {
	if c.TestTunStart > c.TestTunEnd {
		return errors.New("FREE_PROXY_TEST_TUN_START must not exceed FREE_PROXY_TEST_TUN_END")
	}
	if (c.ProxyUsername == "") != (c.ProxyPassword == "") {
		return errors.New("FREE_PROXY_PROXY_USERNAME and FREE_PROXY_PROXY_PASSWORD must be configured together")
	}
	abs, err := filepath.Abs(expandUser(c.DataDir))
	if err != nil {
		return fmt.Errorf("resolve data dir: %w", err)
	}
	c.DataDir = abs
	if c.DatabaseURL == "" {
		dbPath := filepath.Join(c.DataDir, "free-proxy.db")
		c.DatabaseURL = fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(on)", dbPath)
	}
	return nil
}

// EnsureDirectories creates the data directory tree used at runtime.
func (c *Config) EnsureDirectories() error {
	for _, dir := range []string{c.DataDir, c.ConfigsDir(), c.LogsDir()} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) ConfigsDir() string { return filepath.Join(c.DataDir, "configs") }
func (c *Config) LogsDir() string    { return filepath.Join(c.DataDir, "logs") }

// ParsedDNSRepairServers returns the trimmed, non-empty DNS repair servers.
func (c *Config) ParsedDNSRepairServers() []string {
	out := make([]string, 0, 4)
	for _, s := range strings.Split(c.DNSRepairServers, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// Duration accessors convert the numeric *_seconds fields to time.Duration.
func (c *Config) SessionTTL() time.Duration          { return secs(float64(c.SessionTTLSeconds)) }
func (c *Config) ProxyConnectTimeout() time.Duration { return secs(c.ProxyConnectTimeoutSecs) }
func (c *Config) ProxyIdleTimeout() time.Duration    { return secs(c.ProxyIdleTimeoutSecs) }
func (c *Config) OpenVPNTestTimeout() time.Duration  { return secs(c.OpenVPNTestTimeoutSecs) }
func (c *Config) OpenVPNConnectTimeout() time.Duration {
	return secs(c.OpenVPNConnectTimeoutSecs)
}
func (c *Config) RequestTimeout() time.Duration       { return secs(c.RequestTimeoutSecs) }
func (c *Config) HealthCheckInterval() time.Duration  { return secs(c.HealthCheckIntervalSecs) }
func (c *Config) ActivePingInterval() time.Duration   { return secs(c.ActivePingIntervalSecs) }
func (c *Config) MaintenanceInterval() time.Duration  { return secs(c.MaintenanceIntervalSecs) }
func (c *Config) DisconnectedRetry() time.Duration    { return secs(c.DisconnectedRetrySecs) }
func (c *Config) RoutingRetryInterval() time.Duration { return secs(c.RoutingRetryIntervalSecs) }
func (c *Config) IPInfoCacheTTL() time.Duration       { return secs(float64(c.IPInfoCacheSeconds)) }
func (c *Config) InvalidBackoff() time.Duration       { return secs(float64(c.InvalidBackoffSeconds)) }
func (c *Config) StaleNodeGrace() time.Duration       { return secs(float64(c.StaleNodeGraceSeconds)) }

func secs(v float64) time.Duration { return time.Duration(v * float64(time.Second)) }

func expandUser(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			if p == "~" {
				return home
			}
			return filepath.Join(home, p[2:])
		}
	}
	return p
}
