// Package platform detects and installs the system dependencies the service
// needs at runtime (openvpn, iproute2, procps), replacing the dependency-install
// role of the former shell installer.
package platform

import (
	"bufio"
	"os"
	"os/exec"
	"runtime"
	"strings"
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
