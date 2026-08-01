package platform

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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

const defaultEnv = `FREE_PROXY_ENVIRONMENT=production
FREE_PROXY_DATA_DIR=` + DataDir + `
FREE_PROXY_SQL_ECHO=false
FREE_PROXY_ALLOW_PROCESS_RESTART=true
FREE_PROXY_PREFLIGHT_STRICT=false
FREE_PROXY_OPENVPN_COMMAND=openvpn
FREE_PROXY_OPENVPN_USERNAME=vpn
FREE_PROXY_OPENVPN_PASSWORD=vpn
FREE_PROXY_TUNNEL_INTERFACE=tun0
FREE_PROXY_TEST_TUN_START=2
FREE_PROXY_TEST_TUN_END=99
FREE_PROXY_POLICY_ROUTING_TABLE=100
`

var infrastructureEnvKeys = map[string]bool{
	"FREE_PROXY_ENVIRONMENT": true, "FREE_PROXY_DATA_DIR": true,
	"FREE_PROXY_DATABASE_URL": true, "FREE_PROXY_SQL_ECHO": true,
	"FREE_PROXY_ALLOW_PROCESS_RESTART": true, "FREE_PROXY_PREFLIGHT_STRICT": true,
	"FREE_PROXY_OPENVPN_COMMAND": true, "FREE_PROXY_OPENVPN_USERNAME": true,
	"FREE_PROXY_OPENVPN_PASSWORD": true, "FREE_PROXY_TUNNEL_INTERFACE": true,
	"FREE_PROXY_TEST_TUN_START": true, "FREE_PROXY_TEST_TUN_END": true,
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
	return os.WriteFile(EnvFile, []byte(defaultEnv), 0o600)
}

// PruneDatabaseSettingsEnv removes legacy settings after their one-time SQLite
// import. It keeps only machine/bootstrap values that are intentionally outside
// the web control plane.
func PruneDatabaseSettingsEnv() error {
	data, err := os.ReadFile(EnvFile)
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
	tmp := EnvFile + ".tmp"
	if err := os.WriteFile(tmp, []byte(out), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, EnvFile)
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
