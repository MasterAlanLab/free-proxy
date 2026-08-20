package platform

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/masteralanlab/free-proxy/internal/naming"
)

func writeEnv(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "free-proxy.env")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// An upgrade keeps its existing env file, so a host installed before the fix
// still claims tun0 and table 100 — and still fails exactly as issue #2
// describes. The migration has to move those defaults for it.
func TestMigrateLegacyNamingRewritesShippedDefaults(t *testing.T) {
	path := writeEnv(t, strings.Join([]string{
		"FREE_PROXY_ENVIRONMENT=production",
		"FREE_PROXY_TUNNEL_INTERFACE=tun0",
		"FREE_PROXY_TEST_TUN_START=2",
		"FREE_PROXY_TEST_TUN_END=99",
		"FREE_PROXY_POLICY_ROUTING_TABLE=100",
		"",
	}, "\n"))

	changed, err := migrateLegacyNaming(path)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if len(changed) != 5 {
		t.Errorf("expected 5 changes (4 rewrites + the new prefix), got %v", changed)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	for _, want := range []string{
		"FREE_PROXY_TUNNEL_INTERFACE=" + naming.ActiveDevice(),
		"FREE_PROXY_PROBE_DEVICE_PREFIX=" + naming.DevicePrefix,
		"FREE_PROXY_POLICY_ROUTING_TABLE=9527",
		"FREE_PROXY_ENVIRONMENT=production", // untouched keys survive
	} {
		if !strings.Contains(got, want) {
			t.Errorf("migrated env is missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "tun0") {
		t.Errorf("migrated env still claims tun0:\n%s", got)
	}
}

// A deliberately chosen name is the operator's, not ours to move.
func TestMigrateLegacyNamingPreservesCustomValues(t *testing.T) {
	path := writeEnv(t, strings.Join([]string{
		"FREE_PROXY_TUNNEL_INTERFACE=mytun3",
		"FREE_PROXY_POLICY_ROUTING_TABLE=220",
		"FREE_PROXY_PROBE_DEVICE_PREFIX=mytun",
		"",
	}, "\n"))

	changed, err := migrateLegacyNaming(path)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if len(changed) != 0 {
		t.Errorf("customised values must not be rewritten, got %v", changed)
	}
	data, _ := os.ReadFile(path)
	for _, want := range []string{"mytun3", "220", "mytun"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("lost operator value %q:\n%s", want, data)
		}
	}
}

func TestMigrateLegacyNamingIsIdempotent(t *testing.T) {
	path := writeEnv(t, "FREE_PROXY_TUNNEL_INTERFACE=tun0\n")
	if _, err := migrateLegacyNaming(path); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	changed, err := migrateLegacyNaming(path)
	if err != nil {
		t.Fatalf("second migrate: %v", err)
	}
	if len(changed) != 0 {
		t.Errorf("second run should be a no-op, got %v", changed)
	}
}

// The file the installer writes must agree with what the code claims at
// runtime, or a fresh install ships a config the binary would not have chosen.
func TestDefaultEnvMatchesNamingDefaults(t *testing.T) {
	env := defaultEnvContent()
	for _, want := range []string{
		"FREE_PROXY_TUNNEL_INTERFACE=" + naming.ActiveDevice(),
		"FREE_PROXY_PROBE_DEVICE_PREFIX=" + naming.DevicePrefix,
		"FREE_PROXY_POLICY_ROUTING_TABLE=9527",
	} {
		if !strings.Contains(env, want) {
			t.Errorf("default env is missing %q", want)
		}
	}
	for _, key := range []string{"FREE_PROXY_TUNNEL_INTERFACE", "FREE_PROXY_PROBE_DEVICE_PREFIX", "FREE_PROXY_POLICY_ROUTING_TABLE"} {
		if !infrastructureEnvKeys[key] {
			t.Errorf("%s is not in infrastructureEnvKeys and would be pruned from the env file", key)
		}
	}
}

// TestUpgradeFromLegacyEnvFile replays what `free-proxy install` does to an
// env file written by an older release, in the same order: WriteDefaultEnv (a
// no-op, the file exists) -> MigrateLegacyNaming -> PruneDatabaseSettingsEnv.
// The prune step is the trap — it deletes every key not on the infrastructure
// list, so a migration that adds a key the list does not know about would be
// silently undone and the upgraded host would fall back to defaults.
func TestUpgradeFromLegacyEnvFile(t *testing.T) {
	// A realistic pre-upgrade file: infrastructure keys plus settings that have
	// since moved into SQLite and are expected to be pruned.
	path := writeEnv(t, strings.Join([]string{
		"FREE_PROXY_ENVIRONMENT=production",
		"FREE_PROXY_DATA_DIR=/var/lib/free-proxy",
		"FREE_PROXY_OPENVPN_COMMAND=openvpn",
		"FREE_PROXY_TUNNEL_INTERFACE=tun0",
		"FREE_PROXY_TEST_TUN_START=2",
		"FREE_PROXY_TEST_TUN_END=99",
		"FREE_PROXY_POLICY_ROUTING_TABLE=100",
		"FREE_PROXY_WEB_PORT=39527",
		"FREE_PROXY_DISCOVERY_LIMIT=300",
		"",
	}, "\n"))

	if _, err := migrateLegacyNaming(path); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := pruneDatabaseSettingsEnv(path); err != nil {
		t.Fatalf("prune: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)

	// The migrated network identifiers must survive the prune.
	for _, want := range []string{
		"FREE_PROXY_TUNNEL_INTERFACE=" + naming.ActiveDevice(),
		"FREE_PROXY_PROBE_DEVICE_PREFIX=" + naming.DevicePrefix,
		"FREE_PROXY_POLICY_ROUTING_TABLE=9527",
		"FREE_PROXY_TEST_TUN_START=1",
		"FREE_PROXY_TEST_TUN_END=64",
		"FREE_PROXY_DATA_DIR=/var/lib/free-proxy",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("upgraded env lost %q:\n%s", want, got)
		}
	}
	// Database-backed settings are still pruned, as before.
	for _, gone := range []string{"FREE_PROXY_WEB_PORT", "FREE_PROXY_DISCOVERY_LIMIT"} {
		if strings.Contains(got, gone) {
			t.Errorf("%s should have been pruned:\n%s", gone, got)
		}
	}
	if strings.Contains(got, "tun0") {
		t.Errorf("upgraded env still claims tun0:\n%s", got)
	}
}

func TestDeviceNameCheckFlagsForeignNamespace(t *testing.T) {
	// A name in the shared tun* pool is flagged even when nothing holds it yet:
	// it is a collision waiting to happen.
	if c := deviceNameCheck("tun0", naming.DevicePrefix); c.OK {
		t.Errorf("tun0 should be flagged as a shared-pool name, got %+v", c)
	}
	if c := deviceNameCheck(naming.ActiveDevice(), naming.DevicePrefix); !c.OK {
		t.Errorf("our own device name should pass, got %+v", c)
	}
}

func findManager(name string) *PkgManager {
	for i := range managers {
		if managers[i].Name == name {
			return &managers[i]
		}
	}
	return nil
}

func TestAptSteps(t *testing.T) {
	pm := findManager("apt-get")
	steps := pm.steps([]string{"openvpn", "iproute2"})
	if len(steps) != 2 {
		t.Fatalf("apt should update then install, got %v", steps)
	}
	if !slices.Equal(steps[0], []string{"apt-get", "update"}) {
		t.Fatalf("first step = %v", steps[0])
	}
	if !slices.Equal(steps[1], []string{"apt-get", "install", "-y", "openvpn", "iproute2"}) {
		t.Fatalf("install step = %v", steps[1])
	}
}

func TestDnfMapsPackageNames(t *testing.T) {
	pm := findManager("dnf")
	steps := pm.steps([]string{"iproute2", "procps"})
	// dnf/yum use different package names than Debian.
	if !slices.Equal(steps[len(steps)-1], []string{"dnf", "install", "-y", "iproute", "procps-ng"}) {
		t.Fatalf("dnf mapping wrong: %v", steps[len(steps)-1])
	}
}

func TestMissingAndCritical(t *testing.T) {
	checks := []Check{
		{Name: "openvpn", OK: false, Fixable: true, Pkg: "openvpn"},
		{Name: "ip", OK: false, Fixable: true, Pkg: "iproute2"},
		{Name: "sysctl", OK: true},
		{Name: "tun_device", OK: false},
	}
	missing := MissingPackages(checks)
	if !slices.Equal(missing, []string{"openvpn", "iproute2"}) {
		t.Fatalf("missing = %v", missing)
	}
	if !CriticalMissing(checks) {
		t.Fatal("expected critical missing (openvpn/ip/tun)")
	}
	if CriticalMissing([]Check{{Name: "sysctl", OK: false}}) {
		t.Fatal("sysctl alone is not critical")
	}
}
