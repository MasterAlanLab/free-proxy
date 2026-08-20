package netx

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/masteralanlab/free-proxy/internal/naming"
)

// DeviceLister reports the interface names currently present on the host. The
// allocator consults it so a name is verified free against the *kernel*, not
// merely against this process's bookkeeping — another program (or a crashed
// run of our own) may already own it.
type DeviceLister func() map[string]bool

// SystemDevices lists the host's interface names. It never fails hard: on an
// enumeration error it returns an empty set, degrading to the previous
// assume-free behaviour rather than blocking all allocation.
func SystemDevices() map[string]bool {
	present := map[string]bool{}
	ifaces, err := net.Interfaces()
	if err != nil {
		return present
	}
	for _, i := range ifaces {
		present[i.Name] = true
	}
	return present
}

// DeviceExists reports whether an interface with this name is present.
func DeviceExists(name string) bool {
	_, err := net.InterfaceByName(name)
	return err == nil
}

// StaleDevices returns the interfaces in our own prefix pool that are still
// present. They can only be leftovers from a crashed run — a live tunnel keeps
// its device only while its OpenVPN process is alive.
func StaleDevices(prefix string) []string {
	var out []string
	for name := range SystemDevices() {
		if naming.HasDevicePrefix(name, prefix) {
			out = append(out, name)
		}
	}
	return out
}

// RemoveDevice deletes an interface. It refuses any name outside our prefix
// pool, so a misconfiguration can never take down somebody else's tunnel.
func RemoveDevice(ctx context.Context, runner CommandRunner, prefix, name string) bool {
	if !naming.HasDevicePrefix(name, prefix) {
		return false
	}
	if runner == nil {
		runner = SystemCommandRunner{}
	}
	res, err := runner.Run(ctx, []string{"ip", "link", "del", name}, 5*time.Second)
	return err == nil && res.ReturnCode == 0
}

// ReclaimStaleDevices removes leftover interfaces from our own prefix pool and
// returns the names actually removed.
//
// This only ever touches names inside our prefix, which is the whole reason the
// project stopped allocating from the shared tunN pool: reclaiming leftovers
// there would have meant deleting devices that might belong to somebody else.
//
// The active device is deliberately NOT exempt. A leftover copy of the name we
// are about to connect on is exactly what has to go — and "don't cut a tunnel
// that is actually running" is already covered by the live-process filter, which
// is the accurate test rather than a name comparison.
func ReclaimStaleDevices(ctx context.Context, runner CommandRunner, prefix string) []string {
	if runtime.GOOS != "linux" {
		return nil
	}
	live := devicesHeldByLiveTunnels()
	var removed []string
	for _, name := range StaleDevices(prefix) {
		// A device still owned by a running tunnel is not stale — it belongs to
		// a second free-proxy instance on this host, and deleting it would cut
		// that instance's traffic.
		if live[name] {
			continue
		}
		if RemoveDevice(ctx, runner, prefix, name) {
			removed = append(removed, name)
		}
	}
	return removed
}

// EnsureDeviceAvailable makes the active tunnel device name usable, or explains
// why it is not. It gives the active device the same guarantee the probe
// allocator already had: verified against the kernel before OpenVPN is handed
// the name, instead of discovering the clash in a log line.
func EnsureDeviceAvailable(ctx context.Context, runner CommandRunner, prefix, device string) error {
	if runtime.GOOS != "linux" || !DeviceExists(device) {
		return nil
	}
	if devicesHeldByLiveTunnels()[device] {
		return fmt.Errorf("tunnel device %s is in use by another running tunnel process; "+
			"set FREE_PROXY_TUNNEL_INTERFACE to a different name", device)
	}
	if !naming.HasDevicePrefix(device, prefix) {
		return fmt.Errorf("tunnel device %s already exists and is outside this project's %s* namespace, "+
			"so it belongs to another program; set FREE_PROXY_TUNNEL_INTERFACE to an unused name", device, prefix)
	}
	if !RemoveDevice(ctx, runner, prefix, device) {
		return fmt.Errorf("tunnel device %s is left over from an earlier run and could not be removed", device)
	}
	return nil
}

// devicesHeldByLiveTunnels returns the device names claimed by `--dev` on the
// command line of any running process.
//
// This filter guards a deletion, so its errors are asymmetric: naming one extra
// device costs nothing (we skip a reclaim), while missing one destroys a live
// tunnel. It therefore deliberately does NOT require the binary to be called
// "openvpn" — FREE_PROXY_OPENVPN_COMMAND can point at a wrapper or a renamed
// binary, and under the old check a second instance running one of those would
// have had its active device deleted out from under it.
func devicesHeldByLiveTunnels() map[string]bool {
	held := map[string]bool{}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return held
	}
	for _, e := range entries {
		if _, err := strconv.Atoi(e.Name()); err != nil {
			continue
		}
		raw, err := os.ReadFile(filepath.Join("/proc", e.Name(), "cmdline"))
		if err != nil {
			continue
		}
		args := strings.Split(strings.TrimRight(string(raw), "\x00"), "\x00")
		for i, a := range args {
			if (a == "--dev" || a == "--dev-node") && i+1 < len(args) {
				held[args[i+1]] = true
			}
		}
	}
	return held
}
