package platform

import (
	"slices"
	"testing"
)

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
