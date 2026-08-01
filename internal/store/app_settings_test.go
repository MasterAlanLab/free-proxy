package store

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/masteralanlab/free-proxy/internal/config"
)

func TestAppSettingsLegacyImportRunsOnce(t *testing.T) {
	db, err := Open("file:" + filepath.Join(t.TempDir(), "settings.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	repo := NewRepos(db).App
	cfg := &config.Config{
		ProxyEnabled: true, ProxyPort: 12345, ProxyUsername: "proxy-user",
		ProxyMaxConnections: 77, ProxyConnectTimeoutSecs: 12, ProxyIdleTimeoutSecs: 34, ProxyDNSServer: "1.1.1.1",
		VPNGateAPIURL: "https://example.test/vpn", DiscoveryLimit: 300, RequestTimeoutSecs: 9,
		IPInfoAPIURL: "https://example.test/ip", IPInfoCacheSeconds: 99,
		MaintenanceEnabled: true, MaintenanceIntervalSecs: 10800, HealthCheckIntervalSecs: 40,
		ActivePingIntervalSecs: 20, DisconnectedRetrySecs: 60, MaxProbeConcurrency: 3,
		InitialConnectTestLimit: 8, ManualTestNodeLimit: 4, OpenVPNTestTimeoutSecs: 16,
		OpenVPNConnectTimeoutSecs: 36, InvalidBackoffSeconds: 1900, StaleNodeGraceSeconds: 700000,
		DNSRepairEnabled: true, DNSRepairServers: "1.1.1.1", RoutingSetupRetries: 4,
		RoutingRetryIntervalSecs: 2, RoutingStrictRPFilter: true,
	}
	ctx := context.Background()
	if err := repo.InitializeFromLegacyEnv(ctx, cfg, "proxy-hash"); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.Admin.WebPort != 39527 || got.Admin.SessionTTLSeconds != 2592000 || got.Proxy.Port != 12345 || got.Proxy.PasswordHash != "proxy-hash" || got.Discovery.DiscoveryLimit != 300 || got.Maintenance.MaintenanceIntervalSeconds != 10800 {
		t.Fatalf("legacy import mismatch: %+v", got)
	}
	cfg.ProxyPort = 54321
	if err := repo.InitializeFromLegacyEnv(ctx, cfg, "changed"); err != nil {
		t.Fatal(err)
	}
	again, err := repo.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if again.Proxy.Port != 12345 || again.Proxy.PasswordHash != "proxy-hash" {
		t.Fatal("legacy settings were imported more than once")
	}
}
