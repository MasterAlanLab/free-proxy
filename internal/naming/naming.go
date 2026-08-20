// Package naming is the single source of truth for every identifier this
// project places into a *system-global* namespace: network device names,
// policy routing table ids, service/unit names and filesystem paths.
//
// Why this package exists: a host may run several tunnel-managing programs at
// once (3x-ui, WARP, tun2socks, another OpenVPN). Anything we take from a
// shared namespace — `tun0`, routing table `100` — is something we can steal
// from a neighbour, or have stolen from us. See issue #2.
//
// The rules every global identifier must follow:
//
//  1. Namespaced. Derived from a project-owned prefix, never a generic name
//     such as `tun0` that every other tool also defaults to.
//  2. Configurable. The operator can always move us out of the way.
//  3. Verified. Checked as free at runtime instead of assumed free.
//  4. Cleaned up by exact match. We only ever remove what we can attribute to
//     ourselves.
//
// Nothing outside this package may hardcode such a name; naming_test.go
// enforces that for the network device names.
package naming

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	// Project is the prefix for filesystem paths, the service unit and the
	// routing table alias.
	Project = "free-proxy"

	// DevicePrefix namespaces our network interfaces. Deliberately NOT "tun":
	// the tunN pool is a free-for-all that every tunnel program allocates from,
	// so a name inside it is a name we may lose a race for. Kept short — the
	// kernel caps interface names at MaxDeviceNameLen.
	DevicePrefix = "fpx"

	// LegacyDevicePrefix is the shared pool this project used to allocate from.
	// Retained only so upgrades can recognise and rewrite stale configuration.
	LegacyDevicePrefix = "tun"

	// MaxDeviceNameLen is IFNAMSIZ-1: the kernel's limit on interface names.
	MaxDeviceNameLen = 15

	// DefaultRoutingTable is the policy routing table we install the tunnel
	// default route into. Chosen well clear of the crowded low ids (100, 200,
	// …) that other tools pick, and of the reserved 253-255. The kernel has
	// accepted 32-bit table ids since 2.6.19.
	DefaultRoutingTable = 9527

	// RoutingTableAlias is the symbolic name registered for that table in
	// rt_tables.d, so `ip rule show` and human operators can see who owns it.
	RoutingTableAlias = "free_proxy"
)

// LegacyDefaults are the values earlier releases shipped. An install that still
// carries them was never customised by the operator, so an upgrade may safely
// move it into the private namespace.
const (
	LegacyTunnelInterface = "tun0"
	LegacyRoutingTable    = 100
)

// ActiveDevice is the interface name for the live exit tunnel.
func ActiveDevice() string { return ProbeDevice(0) }

// ProbeDevice renders the device name for allocator index i. Index 0 is
// reserved for the active tunnel; probes use 1 and up.
func ProbeDevice(i int) string { return DeviceName(DevicePrefix, i) }

// DeviceName renders prefix+index, the only form of device name we create.
func DeviceName(prefix string, i int) string { return prefix + strconv.Itoa(i) }

// RoutingTableFile is the rt_tables.d fragment that reserves our table id.
func RoutingTableFile() string {
	return "/etc/iproute2/rt_tables.d/" + Project + ".conf"
}

// RoutingTableEntry is the content of that fragment.
func RoutingTableEntry(table int) string {
	return fmt.Sprintf("%d\t%s\n", table, RoutingTableAlias)
}

// HasDevicePrefix reports whether device belongs to the given prefix pool, i.e.
// prefix followed by a decimal index. Used to decide what we are allowed to
// tear down: anything that fails this test belongs to somebody else.
func HasDevicePrefix(device, prefix string) bool {
	if prefix == "" || !strings.HasPrefix(device, prefix) {
		return false
	}
	suffix := device[len(prefix):]
	if suffix == "" {
		return false
	}
	for _, r := range suffix {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// ValidateDeviceName rejects names the kernel cannot accept, so a misconfigured
// FREE_PROXY_TUNNEL_INTERFACE fails at startup rather than deep inside an
// OpenVPN log line.
func ValidateDeviceName(name string) error {
	if name == "" {
		return fmt.Errorf("interface name must not be empty")
	}
	if len(name) > MaxDeviceNameLen {
		return fmt.Errorf("interface name %q exceeds the kernel limit of %d characters", name, MaxDeviceNameLen)
	}
	if strings.ContainsAny(name, " /:\t\n") || name == "." || name == ".." {
		return fmt.Errorf("interface name %q contains characters the kernel rejects", name)
	}
	return nil
}

// ValidateDevicePrefix additionally reserves room for the allocator index.
func ValidateDevicePrefix(prefix string) error {
	if prefix == "" {
		return fmt.Errorf("interface prefix must not be empty")
	}
	// Leave room for a multi-digit index appended by DeviceName.
	if err := ValidateDeviceName(prefix + "000"); err != nil {
		return fmt.Errorf("interface prefix %q is too long", prefix)
	}
	return nil
}

// ValidateRoutingTable rejects the ids the kernel reserves.
func ValidateRoutingTable(table int) error {
	switch {
	case table <= 0:
		return fmt.Errorf("policy routing table must be positive, got %d", table)
	case table == 253 || table == 254 || table == 255:
		return fmt.Errorf("policy routing table %d is reserved by the kernel (default/main/local)", table)
	}
	return nil
}
