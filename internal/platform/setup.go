package platform

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/masteralanlab/free-proxy/internal/naming"
)

// System installation layout. The binary owns the whole install lifecycle
// (dependencies, environment file, init-system service); external scripts only
// need to place the binary and run `free-proxy install`.
const (
	BinPath   = "/usr/local/bin/free-proxy"
	ConfigDir = "/etc/free-proxy"
	EnvFile   = ConfigDir + "/free-proxy.env"
	DataDir   = "/var/lib/free-proxy"

	systemdUnitPath  = "/etc/systemd/system/free-proxy.service"
	openrcScriptPath = "/etc/init.d/free-proxy"
)

// defaultEnvContent renders the initial environment file. The host-global
// identifiers come from internal/naming so the installed default can never
// drift from what the code claims at runtime.
func defaultEnvContent() string {
	return `FREE_PROXY_ENVIRONMENT=production
FREE_PROXY_DATA_DIR=` + DataDir + `
FREE_PROXY_SQL_ECHO=false
FREE_PROXY_ALLOW_PROCESS_RESTART=true
FREE_PROXY_OPENVPN_COMMAND=openvpn
FREE_PROXY_OPENVPN_USERNAME=vpn
FREE_PROXY_OPENVPN_PASSWORD=vpn
# Network identifiers below live in namespaces shared with every other program
# on this host. They use a project-private prefix so free-proxy can coexist with
# 3x-ui, WARP, and other tunnel managers. Change them only to resolve a conflict.
FREE_PROXY_TUNNEL_INTERFACE=` + naming.ActiveDevice() + `
FREE_PROXY_PROBE_DEVICE_PREFIX=` + naming.DevicePrefix + `
FREE_PROXY_TEST_TUN_START=1
FREE_PROXY_TEST_TUN_END=64
FREE_PROXY_POLICY_ROUTING_TABLE=` + strconv.Itoa(naming.DefaultRoutingTable) + `
`
}

var infrastructureEnvKeys = map[string]bool{
	"FREE_PROXY_ENVIRONMENT": true, "FREE_PROXY_DATA_DIR": true,
	"FREE_PROXY_DATABASE_URL": true, "FREE_PROXY_SQL_ECHO": true,
	"FREE_PROXY_ALLOW_PROCESS_RESTART": true,
	"FREE_PROXY_OPENVPN_COMMAND":       true, "FREE_PROXY_OPENVPN_USERNAME": true,
	"FREE_PROXY_OPENVPN_PASSWORD": true, "FREE_PROXY_TUNNEL_INTERFACE": true,
	"FREE_PROXY_PROBE_DEVICE_PREFIX": true,
	"FREE_PROXY_TEST_TUN_START":      true, "FREE_PROXY_TEST_TUN_END": true,
	"FREE_PROXY_POLICY_ROUTING_TABLE": true,
	"FREE_PROXY_ENV_FILE":             true, "FREE_PROXY_REPO": true, "FREE_PROXY_RELEASE": true,
}

const systemdUnit = `[Unit]
Description=Free Proxy exit pool and local proxy gateway
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=` + DataDir + `
EnvironmentFile=-` + EnvFile + `
ExecStartPre=` + BinPath + ` doctor
ExecStart=` + BinPath + ` serve
Restart=on-failure
RestartSec=5
TimeoutStopSec=30
KillMode=mixed
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
`

const openrcScript = `#!/sbin/openrc-run

name="Free Proxy"
description="Free Proxy exit pool and local proxy gateway"
command="` + BinPath + `"
command_args="serve"
command_background="yes"
directory="` + DataDir + `"
pidfile="/run/free-proxy.pid"
output_log="/var/log/free-proxy.log"
error_log="/var/log/free-proxy.log"
env_file="` + EnvFile + `"

depend() {
    need net
    after firewall
}

start_pre() {
    if [ -f "$env_file" ]; then
        set -a
        . "$env_file"
        set +a
    fi
    "$command" doctor || true
}
`

// RequireInstallSupport validates that system installation can proceed here.
func RequireInstallSupport() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("system install is only supported on Linux")
	}
	if os.Geteuid() != 0 {
		return fmt.Errorf("this command must run as root")
	}
	return nil
}

// InstallSelf copies the running executable to BinPath. It is a no-op when the
// process already runs from there. The copy goes through a temp file + rename
// so a binary currently used by the service is replaced atomically.
func InstallSelf() error {
	src, err := os.Executable()
	if err != nil {
		return err
	}
	if src, err = filepath.EvalSymlinks(src); err != nil {
		return err
	}
	if src == BinPath {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp, err := os.CreateTemp(filepath.Dir(BinPath), ".free-proxy-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err = io.Copy(tmp, in); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Chmod(0o755); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), BinPath)
}

// WriteDefaultEnv creates the config and data directories and writes the
// default environment file. An existing env file is left untouched.
func WriteDefaultEnv() error {
	for _, dir := range []string{ConfigDir, DataDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	if _, err := os.Stat(EnvFile); err == nil {
		return nil
	}
	return os.WriteFile(EnvFile, []byte(defaultEnvContent()), 0o600)
}

// MigrateLegacyNaming moves an existing install off the shared namespaces it
// used to claim (tun0, routing table 100) and onto the project-private ones.
//
// Only values still equal to the old shipped defaults are rewritten: an
// operator who deliberately chose a name keeps it. Returns the changes made,
// for the installer to report. Upgrades matter here — WriteDefaultEnv leaves an
// existing env file alone, so without this an upgraded host would keep failing
// exactly as issue #2 describes.
func MigrateLegacyNaming() ([]string, error) { return migrateLegacyNaming(EnvFile) }

func migrateLegacyNaming(envFile string) ([]string, error) {
	data, err := os.ReadFile(envFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	rewrites := map[string][2]string{
		"FREE_PROXY_TUNNEL_INTERFACE":     {naming.LegacyTunnelInterface, naming.ActiveDevice()},
		"FREE_PROXY_POLICY_ROUTING_TABLE": {strconv.Itoa(naming.LegacyRoutingTable), strconv.Itoa(naming.DefaultRoutingTable)},
		// The old probe range started at 2 to stay clear of tun0/tun1; the
		// private pool only needs to clear index 0.
		"FREE_PROXY_TEST_TUN_START": {"2", "1"},
		"FREE_PROXY_TEST_TUN_END":   {"99", "64"},
	}

	lines := strings.Split(string(data), "\n")
	var changed []string
	seenPrefix := false
	for i, line := range lines {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok || strings.HasPrefix(key, "#") {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "FREE_PROXY_PROBE_DEVICE_PREFIX" {
			seenPrefix = true
			continue
		}
		rewrite, ok := rewrites[key]
		if !ok || strings.Trim(strings.TrimSpace(value), `"'`) != rewrite[0] {
			continue
		}
		lines[i] = key + "=" + rewrite[1]
		changed = append(changed, fmt.Sprintf("%s: %s -> %s", key, rewrite[0], rewrite[1]))
	}
	if !seenPrefix {
		lines = append(lines, "FREE_PROXY_PROBE_DEVICE_PREFIX="+naming.DevicePrefix)
		changed = append(changed, "FREE_PROXY_PROBE_DEVICE_PREFIX: set to "+naming.DevicePrefix)
	}
	if len(changed) == 0 {
		return nil, nil
	}
	out := strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
	tmp := envFile + ".tmp"
	if err := os.WriteFile(tmp, []byte(out), 0o600); err != nil {
		return nil, err
	}
	if err := os.Rename(tmp, envFile); err != nil {
		return nil, err
	}
	return changed, nil
}

// RegisterRoutingTable names our table id in rt_tables.d. This claims nothing
// the kernel enforces, but it makes the id visible as taken in `ip rule show`
// and to anyone reading the host's routing configuration — the courtesy half of
// not colliding with the next tool installed here.
func RegisterRoutingTable(table int) error {
	dir := filepath.Dir(naming.RoutingTableFile())
	if _, err := os.Stat(dir); err != nil {
		return nil // iproute2 without rt_tables.d support; nothing to register
	}
	return os.WriteFile(naming.RoutingTableFile(), []byte(naming.RoutingTableEntry(table)), 0o644)
}

// PruneDatabaseSettingsEnv removes legacy settings after their one-time SQLite
// import. It keeps only machine/bootstrap values that are intentionally outside
// the web control plane.
func PruneDatabaseSettingsEnv() error { return pruneDatabaseSettingsEnv(EnvFile) }

func pruneDatabaseSettingsEnv(envFile string) error {
	data, err := os.ReadFile(envFile)
	if err != nil {
		return err
	}
	var kept []string
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			kept = append(kept, line)
			continue
		}
		key, _, ok := strings.Cut(trimmed, "=")
		if ok && infrastructureEnvKeys[strings.TrimSpace(key)] {
			kept = append(kept, line)
		}
	}
	out := strings.TrimRight(strings.Join(kept, "\n"), "\n") + "\n"
	tmp := envFile + ".tmp"
	if err := os.WriteFile(tmp, []byte(out), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, envFile)
}

// InstallService writes the init-system service, enables it, and (re)starts it.
func InstallService(ctx context.Context) error {
	switch {
	case hasCommand("systemctl"):
		if err := os.WriteFile(systemdUnitPath, []byte(systemdUnit), 0o644); err != nil {
			return err
		}
		return runAll(ctx,
			[]string{"systemctl", "daemon-reload"},
			[]string{"systemctl", "enable", "free-proxy.service"},
			[]string{"systemctl", "restart", "free-proxy.service"},
		)
	case hasCommand("rc-update"):
		if err := os.WriteFile(openrcScriptPath, []byte(openrcScript), 0o755); err != nil {
			return err
		}
		_ = runAll(ctx, []string{"rc-update", "add", "free-proxy", "default"})
		return runAll(ctx, []string{"rc-service", "free-proxy", "restart"})
	default:
		return fmt.Errorf("neither systemd nor OpenRC detected")
	}
}

// Uninstall stops and removes the service, configuration, and binary.
// Data under DataDir is removed only when purgeData is set.
func Uninstall(ctx context.Context, purgeData bool) error {
	if hasCommand("systemctl") {
		_ = runAll(ctx,
			[]string{"systemctl", "stop", "free-proxy.service"},
			[]string{"systemctl", "disable", "free-proxy.service"},
		)
		_ = os.Remove(systemdUnitPath)
		_ = runAll(ctx, []string{"systemctl", "daemon-reload"})
	}
	if hasCommand("rc-update") {
		_ = runAll(ctx,
			[]string{"rc-service", "free-proxy", "stop"},
			[]string{"rc-update", "del", "free-proxy", "default"},
		)
		_ = os.Remove(openrcScriptPath)
	}
	_ = os.Remove(naming.RoutingTableFile())
	if err := os.RemoveAll(ConfigDir); err != nil {
		return err
	}
	if purgeData {
		if err := os.RemoveAll(DataDir); err != nil {
			return err
		}
	}
	return os.Remove(BinPath)
}

func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// runAll executes the given command lines sequentially, streaming output, and
// stops at the first failure.
func runAll(ctx context.Context, cmds ...[]string) error {
	for _, c := range cmds {
		fmt.Printf("+ %v\n", c)
		cmd := exec.CommandContext(ctx, c[0], c[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%v failed: %w", c, err)
		}
	}
	return nil
}
