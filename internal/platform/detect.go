// Package platform detects and installs the system dependencies the service
// needs at runtime (openvpn, iproute2, procps), replacing the dependency-install
// role of the former shell installer.
package platform

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"

	"github.com/masteralanlab/free-proxy/internal/naming"
	"github.com/masteralanlab/free-proxy/internal/netx"
)

// Check is the result of a single dependency probe.
type Check struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Detail  string `json:"detail"`
	Fixable bool   `json:"fixable"`
	Pkg     string `json:"pkg,omitempty"` // logical package that provides it
}

// RunChecks probes every runtime dependency.
func RunChecks() []Check {
	return []Check{
		commandCheck("openvpn", "openvpn", "openvpn"),
		commandCheck("ip", "ip", "iproute2"),
		commandCheck("sysctl", "sysctl", "procps"),
		tunCheck(),
		rootCheck(),
	}
}

// MissingPackages returns the logical packages for failed, fixable checks.
func MissingPackages(checks []Check) []string {
	seen := map[string]bool{}
	var out []string
	for _, c := range checks {
		if !c.OK && c.Fixable && c.Pkg != "" && !seen[c.Pkg] {
			seen[c.Pkg] = true
			out = append(out, c.Pkg)
		}
	}
	return out
}

// CriticalMissing reports whether a check that blocks operation failed.
func CriticalMissing(checks []Check) bool {
	for _, c := range checks {
		if !c.OK && (c.Name == "openvpn" || c.Name == "ip" || c.Name == "tun_device") {
			return true
		}
	}
	return false
}

func commandCheck(name, bin, pkg string) Check {
	path, err := exec.LookPath(bin)
	if err != nil {
		return Check{Name: name, OK: false, Detail: bin + " not found in PATH", Fixable: true, Pkg: pkg}
	}
	detail := path
	if bin == "openvpn" {
		if out, err := exec.Command(bin, "--version").CombinedOutput(); err == nil {
			if line := firstLine(string(out)); line != "" {
				detail = line
			}
		}
	}
	return Check{Name: name, OK: true, Detail: detail}
}

// NamingChecks probes the host-global identifiers this install claims — the
// tunnel device names and the policy routing table id. They are advisory (never
// part of CriticalMissing), but they turn issue #2's symptom into a diagnosis:
// the operator learns the name is taken *before* OpenVPN fails on it.
func NamingChecks(ctx context.Context, activeDevice, probePrefix string, table int) []Check {
	if runtime.GOOS != "linux" {
		return nil
	}
	if probePrefix == "" {
		probePrefix = naming.DevicePrefix
	}
	checks := []Check{deviceNameCheck(activeDevice, probePrefix), probePoolCheck(probePrefix)}

	router := netx.NewPolicyRouter(nil, netx.PolicyRouterConfig{
		Table: table, Interface: activeDevice, DevicePrefix: probePrefix,
	})
	if foreign := router.TableConflict(ctx); foreign > 0 {
		checks = append(checks, Check{Name: "routing_table", OK: false, Detail: fmt.Sprintf(
			"table %d already holds %d route(s) from another program; set FREE_PROXY_POLICY_ROUTING_TABLE to a free id",
			table, foreign)})
	} else {
		checks = append(checks, Check{Name: "routing_table", OK: true, Detail: fmt.Sprintf("table %d", table)})
	}
	return checks
}

// deviceNameCheck distinguishes a name that is merely *ours already* (our own
// running tunnel, or a leftover we will reclaim) from a name owned by a
// different program, which is the case the operator has to act on.
func deviceNameCheck(activeDevice, probePrefix string) Check {
	exists := netx.DeviceExists(activeDevice)
	ours := naming.HasDevicePrefix(activeDevice, probePrefix)
	switch {
	case exists && !ours:
		return Check{Name: "tun_name", OK: false, Detail: fmt.Sprintf(
			"%s already exists and is outside this project's %s* namespace — another program (3x-ui, WARP, another VPN) owns it; "+
				"set FREE_PROXY_TUNNEL_INTERFACE to an unused name", activeDevice, probePrefix)}
	case exists:
		return Check{Name: "tun_name", OK: true, Detail: activeDevice + " present (this service's own tunnel or a leftover to reclaim)"}
	case naming.HasDevicePrefix(activeDevice, naming.LegacyDevicePrefix):
		return Check{Name: "tun_name", OK: false, Detail: fmt.Sprintf(
			"%s is in the shared %s* pool every tunnel program allocates from; move to %s* by clearing FREE_PROXY_TUNNEL_INTERFACE",
			activeDevice, naming.LegacyDevicePrefix, naming.DevicePrefix)}
	default:
		return Check{Name: "tun_name", OK: true, Detail: activeDevice + " is free"}
	}
}

func probePoolCheck(probePrefix string) Check {
	stale := netx.StaleDevices(probePrefix)
	if len(stale) == 0 {
		return Check{Name: "probe_devices", OK: true, Detail: probePrefix + "* pool is clear"}
	}
	sort.Strings(stale)
	return Check{Name: "probe_devices", OK: true, Detail: fmt.Sprintf(
		"%d leftover device(s) in the %s* pool (%s); reclaimed on next start",
		len(stale), probePrefix, strings.Join(stale, ", "))}
}

func tunCheck() Check {
	if runtime.GOOS != "linux" {
		return Check{Name: "tun_device", OK: true, Detail: "not required outside Linux"}
	}
	if _, err := os.Stat("/dev/net/tun"); err != nil {
		return Check{Name: "tun_device", OK: false, Detail: "/dev/net/tun missing (enable TUN/TAP in the VPS panel)"}
	}
	return Check{Name: "tun_device", OK: true, Detail: "/dev/net/tun"}
}

func rootCheck() Check {
	if runtime.GOOS != "linux" {
		return Check{Name: "root", OK: true, Detail: "not required outside Linux"}
	}
	if os.Geteuid() != 0 {
		return Check{Name: "root", OK: false, Detail: "run as root for tunnel/routing operations"}
	}
	return Check{Name: "root", OK: true, Detail: "uid=0"}
}

// OSRelease returns the distro ID from /etc/os-release (best effort).
func OSRelease() string {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return runtime.GOOS
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "ID=") {
			return strings.Trim(strings.TrimPrefix(line, "ID="), `"`)
		}
	}
	return runtime.GOOS
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}
