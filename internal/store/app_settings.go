package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/masteralanlab/free-proxy/internal/config"
	"github.com/masteralanlab/free-proxy/internal/domain"
)

// AppSettingsRepository persists all settings that can be managed from the web
// control plane. Infrastructure/bootstrap values remain in free-proxy.env.
type settingsDB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type AppSettingsRepository struct {
	db   settingsDB
	root *sql.DB
}

func (r *AppSettingsRepository) Get(ctx context.Context) (domain.AppSettings, error) {
	var out domain.AppSettings
	var adminExternal, proxyEnabled, proxyExternal, maintenanceEnabled int64
	var dnsRepair, strictRPF int64
	err := r.db.QueryRowContext(ctx, `SELECT username,password_hash,secret_path,session_ttl_seconds,web_port,web_external_access FROM admin_settings WHERE id=1`).Scan(
		&out.Admin.Username, &out.Admin.PasswordHash, &out.Admin.SecretPath, &out.Admin.SessionTTLSeconds, &out.Admin.WebPort, &adminExternal)
	if err != nil {
		return out, err
	}
	err = r.db.QueryRowContext(ctx, `SELECT enabled,port,username,password_hash,external_access,max_connections,connect_timeout_seconds,idle_timeout_seconds,dns_server FROM proxy_settings WHERE id=1`).Scan(
		&proxyEnabled, &out.Proxy.Port, &out.Proxy.Username, &out.Proxy.PasswordHash, &proxyExternal, &out.Proxy.MaxConnections, &out.Proxy.ConnectTimeoutSeconds, &out.Proxy.IdleTimeoutSeconds, &out.Proxy.DNSServer)
	if err != nil {
		return out, err
	}
	err = r.db.QueryRowContext(ctx, `SELECT vpngate_api_url,discovery_limit,request_timeout_secs,ip_info_api_url,ip_info_cache_seconds FROM discovery_settings WHERE id=1`).Scan(
		&out.Discovery.VPNGateAPIURL, &out.Discovery.DiscoveryLimit, &out.Discovery.RequestTimeoutSecs, &out.Discovery.IPInfoAPIURL, &out.Discovery.IPInfoCacheSeconds)
	if err != nil {
		return out, err
	}
	err = r.db.QueryRowContext(ctx, `SELECT enabled,maintenance_interval_seconds,health_check_interval_seconds,active_ping_interval_seconds,disconnected_retry_seconds,max_probe_concurrency,initial_connect_test_limit,manual_test_node_limit,openvpn_test_timeout_seconds,openvpn_connect_timeout_seconds,invalid_backoff_seconds,stale_node_grace_seconds FROM maintenance_settings WHERE id=1`).Scan(
		&maintenanceEnabled, &out.Maintenance.MaintenanceIntervalSeconds, &out.Maintenance.HealthCheckIntervalSeconds, &out.Maintenance.ActivePingIntervalSeconds, &out.Maintenance.DisconnectedRetrySeconds, &out.Maintenance.MaxProbeConcurrency, &out.Maintenance.InitialConnectTestLimit, &out.Maintenance.ManualTestNodeLimit, &out.Maintenance.OpenVPNTestTimeoutSeconds, &out.Maintenance.OpenVPNConnectTimeoutSeconds, &out.Maintenance.InvalidBackoffSeconds, &out.Maintenance.StaleNodeGraceSeconds)
	if err != nil {
		return out, err
	}
	err = r.db.QueryRowContext(ctx, `SELECT dns_repair_enabled,dns_repair_servers,routing_setup_retries,routing_retry_interval_seconds,routing_strict_rp_filter FROM network_settings WHERE id=1`).Scan(
		&dnsRepair, &out.Network.DNSRepairServers, &out.Network.RoutingSetupRetries, &out.Network.RoutingRetryIntervalSeconds, &strictRPF)
	if err != nil {
		return out, err
	}
	out.Admin.WebExternalAccess = adminExternal != 0
	out.Admin.PasswordSet = out.Admin.PasswordHash != ""
	out.Proxy.Enabled = proxyEnabled != 0
	out.Proxy.ExternalAccess = proxyExternal != 0
	out.Proxy.PasswordSet = out.Proxy.PasswordHash != ""
	out.Maintenance.Enabled = maintenanceEnabled != 0
	out.Network.DNSRepairEnabled = dnsRepair != 0
	out.Network.RoutingStrictRPFilter = strictRPF != 0
	return out, nil
}

func (r *AppSettingsRepository) UpdateAdmin(ctx context.Context, s domain.AdminSettings) error {
	_, err := r.db.ExecContext(ctx, `UPDATE admin_settings SET username=?,password_hash=?,secret_path=?,session_ttl_seconds=?,web_port=?,web_external_access=? WHERE id=1`,
		s.Username, s.PasswordHash, s.SecretPath, s.SessionTTLSeconds, s.WebPort, b2i(s.WebExternalAccess))
	return err
}

func (r *AppSettingsRepository) UpdateProxy(ctx context.Context, s domain.ProxyServiceSettings) error {
	_, err := r.db.ExecContext(ctx, `UPDATE proxy_settings SET enabled=?,port=?,username=?,password_hash=?,external_access=?,max_connections=?,connect_timeout_seconds=?,idle_timeout_seconds=?,dns_server=? WHERE id=1`,
		b2i(s.Enabled), s.Port, s.Username, s.PasswordHash, b2i(s.ExternalAccess), s.MaxConnections, s.ConnectTimeoutSeconds, s.IdleTimeoutSeconds, s.DNSServer)
	return err
}

func (r *AppSettingsRepository) UpdateDiscovery(ctx context.Context, s domain.DiscoverySettings) error {
	_, err := r.db.ExecContext(ctx, `UPDATE discovery_settings SET vpngate_api_url=?,discovery_limit=?,request_timeout_secs=?,ip_info_api_url=?,ip_info_cache_seconds=? WHERE id=1`,
		s.VPNGateAPIURL, s.DiscoveryLimit, s.RequestTimeoutSecs, s.IPInfoAPIURL, s.IPInfoCacheSeconds)
	return err
}

func (r *AppSettingsRepository) UpdateMaintenance(ctx context.Context, s domain.MaintenanceSettings) error {
	_, err := r.db.ExecContext(ctx, `UPDATE maintenance_settings SET enabled=?,maintenance_interval_seconds=?,health_check_interval_seconds=?,active_ping_interval_seconds=?,disconnected_retry_seconds=?,max_probe_concurrency=?,initial_connect_test_limit=?,manual_test_node_limit=?,openvpn_test_timeout_seconds=?,openvpn_connect_timeout_seconds=?,invalid_backoff_seconds=?,stale_node_grace_seconds=? WHERE id=1`,
		b2i(s.Enabled), s.MaintenanceIntervalSeconds, s.HealthCheckIntervalSeconds, s.ActivePingIntervalSeconds, s.DisconnectedRetrySeconds, s.MaxProbeConcurrency, s.InitialConnectTestLimit, s.ManualTestNodeLimit, s.OpenVPNTestTimeoutSeconds, s.OpenVPNConnectTimeoutSeconds, s.InvalidBackoffSeconds, s.StaleNodeGraceSeconds)
	return err
}

func (r *AppSettingsRepository) UpdateNetwork(ctx context.Context, s domain.NetworkSettings) error {
	_, err := r.db.ExecContext(ctx, `UPDATE network_settings SET dns_repair_enabled=?,dns_repair_servers=?,routing_setup_retries=?,routing_retry_interval_seconds=?,routing_strict_rp_filter=? WHERE id=1`,
		b2i(s.DNSRepairEnabled), s.DNSRepairServers, s.RoutingSetupRetries, s.RoutingRetryIntervalSeconds, b2i(s.RoutingStrictRPFilter))
	return err
}

func (r *AppSettingsRepository) UpdateAll(ctx context.Context, s domain.AppSettings) error {
	tx, err := r.root.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := &AppSettingsRepository{db: tx}
	if err = q.UpdateAdmin(ctx, s.Admin); err != nil {
		return err
	}
	if err = q.UpdateProxy(ctx, s.Proxy); err != nil {
		return err
	}
	if err = q.UpdateDiscovery(ctx, s.Discovery); err != nil {
		return err
	}
	if err = q.UpdateMaintenance(ctx, s.Maintenance); err != nil {
		return err
	}
	if err = q.UpdateNetwork(ctx, s.Network); err != nil {
		return err
	}
	return tx.Commit()
}

// InitializeFromLegacyEnv imports the former database-owned environment values
// once. Subsequent starts always use the database and ignore those legacy keys.
func (r *AppSettingsRepository) InitializeFromLegacyEnv(ctx context.Context, cfg *config.Config, proxyPasswordHash string) error {
	var marker string
	err := r.db.QueryRowContext(ctx, `SELECT value FROM app_metadata WHERE key='legacy_env_imported'`).Scan(&marker)
	if err == nil {
		return nil
	}
	if err != sql.ErrNoRows {
		return err
	}
	tx, err := r.root.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	webPort := cfg.WebPort
	if webPort == 0 || webPort == 8787 {
		webPort = 39527
	}
	sessionTTL := cfg.SessionTTLSeconds
	if sessionTTL <= 0 {
		sessionTTL = 2592000
	}
	queries := []struct {
		q    string
		args []any
	}{
		{`UPDATE admin_settings SET session_ttl_seconds=?,web_port=? WHERE id=1`, []any{sessionTTL, webPort}},
		{`UPDATE proxy_settings SET enabled=?,port=?,username=?,password_hash=?,max_connections=?,connect_timeout_seconds=?,idle_timeout_seconds=?,dns_server=? WHERE id=1`, []any{b2i(cfg.ProxyEnabled), cfg.ProxyPort, cfg.ProxyUsername, proxyPasswordHash, cfg.ProxyMaxConnections, cfg.ProxyConnectTimeoutSecs, cfg.ProxyIdleTimeoutSecs, cfg.ProxyDNSServer}},
		{`UPDATE discovery_settings SET vpngate_api_url=?,discovery_limit=?,request_timeout_secs=?,ip_info_api_url=?,ip_info_cache_seconds=? WHERE id=1`, []any{cfg.VPNGateAPIURL, cfg.DiscoveryLimit, cfg.RequestTimeoutSecs, cfg.IPInfoAPIURL, cfg.IPInfoCacheSeconds}},
		{`UPDATE maintenance_settings SET enabled=?,maintenance_interval_seconds=?,health_check_interval_seconds=?,active_ping_interval_seconds=?,disconnected_retry_seconds=?,max_probe_concurrency=?,initial_connect_test_limit=?,manual_test_node_limit=?,openvpn_test_timeout_seconds=?,openvpn_connect_timeout_seconds=?,invalid_backoff_seconds=?,stale_node_grace_seconds=? WHERE id=1`, []any{b2i(cfg.MaintenanceEnabled), cfg.MaintenanceIntervalSecs, cfg.HealthCheckIntervalSecs, cfg.ActivePingIntervalSecs, cfg.DisconnectedRetrySecs, cfg.MaxProbeConcurrency, cfg.InitialConnectTestLimit, cfg.ManualTestNodeLimit, cfg.OpenVPNTestTimeoutSecs, cfg.OpenVPNConnectTimeoutSecs, cfg.InvalidBackoffSeconds, cfg.StaleNodeGraceSeconds}},
		{`UPDATE network_settings SET dns_repair_enabled=?,dns_repair_servers=?,routing_setup_retries=?,routing_retry_interval_seconds=?,routing_strict_rp_filter=? WHERE id=1`, []any{b2i(cfg.DNSRepairEnabled), cfg.DNSRepairServers, cfg.RoutingSetupRetries, cfg.RoutingRetryIntervalSecs, b2i(cfg.RoutingStrictRPFilter)}},
	}
	for _, item := range queries {
		if _, err = tx.ExecContext(ctx, item.q, item.args...); err != nil {
			return fmt.Errorf("import legacy settings: %w", err)
		}
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO app_metadata(key,value) VALUES('legacy_env_imported','1')`); err != nil {
		return err
	}
	return tx.Commit()
}

// ApplyToConfig hydrates existing services from the database-backed settings.
// This keeps one consistent runtime snapshot while settings changes trigger a restart.
func ApplyToConfig(cfg *config.Config, s domain.AppSettings) {
	cfg.WebHost, cfg.ProxyHost = "0.0.0.0", "0.0.0.0"
	cfg.AdminAuthEnabled = true
	cfg.WebPort, cfg.SessionTTLSeconds = s.Admin.WebPort, s.Admin.SessionTTLSeconds
	cfg.ProxyEnabled, cfg.ProxyPort = s.Proxy.Enabled, s.Proxy.Port
	cfg.ProxyUsername, cfg.ProxyPassword = s.Proxy.Username, ""
	cfg.ProxyMaxConnections, cfg.ProxyConnectTimeoutSecs, cfg.ProxyIdleTimeoutSecs, cfg.ProxyDNSServer = s.Proxy.MaxConnections, s.Proxy.ConnectTimeoutSeconds, s.Proxy.IdleTimeoutSeconds, s.Proxy.DNSServer
	cfg.VPNGateAPIURL, cfg.DiscoveryLimit, cfg.RequestTimeoutSecs = s.Discovery.VPNGateAPIURL, s.Discovery.DiscoveryLimit, s.Discovery.RequestTimeoutSecs
	cfg.IPInfoAPIURL, cfg.IPInfoCacheSeconds = s.Discovery.IPInfoAPIURL, s.Discovery.IPInfoCacheSeconds
	cfg.MaintenanceEnabled, cfg.MaintenanceIntervalSecs = s.Maintenance.Enabled, s.Maintenance.MaintenanceIntervalSeconds
	cfg.HealthCheckIntervalSecs, cfg.ActivePingIntervalSecs, cfg.DisconnectedRetrySecs = s.Maintenance.HealthCheckIntervalSeconds, s.Maintenance.ActivePingIntervalSeconds, s.Maintenance.DisconnectedRetrySeconds
	cfg.MaxProbeConcurrency, cfg.InitialConnectTestLimit, cfg.ManualTestNodeLimit = s.Maintenance.MaxProbeConcurrency, s.Maintenance.InitialConnectTestLimit, s.Maintenance.ManualTestNodeLimit
	cfg.OpenVPNTestTimeoutSecs, cfg.OpenVPNConnectTimeoutSecs = s.Maintenance.OpenVPNTestTimeoutSeconds, s.Maintenance.OpenVPNConnectTimeoutSeconds
	cfg.InvalidBackoffSeconds, cfg.StaleNodeGraceSeconds = s.Maintenance.InvalidBackoffSeconds, s.Maintenance.StaleNodeGraceSeconds
	cfg.DNSRepairEnabled, cfg.DNSRepairServers = s.Network.DNSRepairEnabled, s.Network.DNSRepairServers
	cfg.RoutingSetupRetries, cfg.RoutingRetryIntervalSecs, cfg.RoutingStrictRPFilter = s.Network.RoutingSetupRetries, s.Network.RoutingRetryIntervalSeconds, s.Network.RoutingStrictRPFilter
}
