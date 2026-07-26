package platform

import "os/exec"

// PkgManager describes how to install packages on a distro family.
type PkgManager struct {
	Name    string
	update  []string          // optional index refresh
	install []string          // install command prefix
	pkgs    map[string]string // logical package -> distro package name
}

// managers is the ordered detection list (first available wins).
var managers = []PkgManager{
	{
		Name:    "apt-get",
		update:  []string{"apt-get", "update"},
		install: []string{"apt-get", "install", "-y"},
		pkgs:    map[string]string{"openvpn": "openvpn", "iproute2": "iproute2", "procps": "procps", "ca-certificates": "ca-certificates"},
	},
	{
		Name:    "apk",
		install: []string{"apk", "add", "--no-cache"},
		pkgs:    map[string]string{"openvpn": "openvpn", "iproute2": "iproute2", "procps": "procps", "ca-certificates": "ca-certificates"},
	},
	{
		Name:    "dnf",
		install: []string{"dnf", "install", "-y"},
		pkgs:    map[string]string{"openvpn": "openvpn", "iproute2": "iproute", "procps": "procps-ng", "ca-certificates": "ca-certificates"},
	},
	{
		Name:    "yum",
		install: []string{"yum", "install", "-y"},
		pkgs:    map[string]string{"openvpn": "openvpn", "iproute2": "iproute", "procps": "procps-ng", "ca-certificates": "ca-certificates"},
	},
}

// DetectPkgManager returns the first available package manager, or nil.
func DetectPkgManager() *PkgManager {
	for i := range managers {
		if _, err := exec.LookPath(managers[i].bin()); err == nil {
			return &managers[i]
		}
	}
	return nil
}

func (p *PkgManager) bin() string {
	if len(p.install) > 0 {
		return p.install[0]
	}
	return p.Name
}

// steps returns the command lines to install the given logical packages.
func (p *PkgManager) steps(logical []string) [][]string {
	var names []string
	for _, l := range logical {
		if actual, ok := p.pkgs[l]; ok {
			names = append(names, actual)
		} else {
			names = append(names, l)
		}
	}
	var out [][]string
	if len(p.update) > 0 {
		out = append(out, p.update)
	}
	out = append(out, append(append([]string{}, p.install...), names...))
	return out
}
