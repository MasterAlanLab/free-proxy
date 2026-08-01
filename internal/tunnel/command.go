package tunnel

import (
	"os"
	"strings"
)

// Version is an OpenVPN major.minor version.
type Version struct{ Major, Minor int }

// GTE reports whether v >= major.minor.
func (v Version) GTE(major, minor int) bool {
	if v.Major != major {
		return v.Major > major
	}
	return v.Minor >= minor
}

// BuildParams configures an OpenVPN invocation.
type BuildParams struct {
	Executable  []string
	ConfigFile  string
	AuthFile    string
	Device      string
	RouteNoPull bool
	Version     Version
}

const dataCiphers = "AES-128-CBC:AES-256-GCM:AES-128-GCM:CHACHA20-POLY1305"

// BuildArgs assembles the full OpenVPN argument vector, mirroring the former
// OpenVpnCommandBuilder.
func BuildArgs(p BuildParams) []string {
	args := append([]string{}, p.Executable...)
	args = append(args,
		"--config", p.ConfigFile,
		"--dev", p.Device,
		"--dev-type", "tun",
		"--pull-filter", "ignore", "route-ipv6",
		"--pull-filter", "ignore", "ifconfig-ipv6",
		"--route-delay", "2",
		"--connect-retry-max", "1",
		"--connect-timeout", "15",
		"--auth-user-pass", p.AuthFile,
		"--auth-nocache",
		"--verb", "3",
	)
	if p.Version.GTE(2, 5) {
		args = append(args, "--data-ciphers", dataCiphers)
	} else {
		args = append(args, "--ncp-ciphers", dataCiphers)
	}
	if fileExists("/etc/ssl/certs") {
		args = append(args, "--capath", "/etc/ssl/certs")
	}
	if p.RouteNoPull {
		args = append(args, "--route-nopull")
	}
	return args
}

// ParseExecutable splits the configured openvpn command into argv.
func ParseExecutable(command string) []string {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return []string{"openvpn"}
	}
	return fields
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
