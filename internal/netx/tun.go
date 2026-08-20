package netx

import (
	"errors"
	"fmt"
	"sync"

	"github.com/masteralanlab/free-proxy/internal/naming"
)

// TunAllocator hands out exclusive tunnel device names from a private,
// project-prefixed pool for concurrent probing.
//
// Two independent conditions must hold before a name is handed out: it must be
// free in our own bookkeeping, and it must not already exist on the host. The
// second check is what keeps us out of issue #2 — before it, the allocator
// happily returned a name another program was already using and OpenVPN died
// with "Cannot allocate TUN/TAP dev".
type TunAllocator struct {
	prefix     string
	start, end int
	devices    DeviceLister

	mu        sync.Mutex
	allocated map[int]bool
}

// NewTunAllocator creates an allocator over indices [start, end] of the prefix
// pool. An empty prefix falls back to the project default.
func NewTunAllocator(prefix string, start, end int) (*TunAllocator, error) {
	if prefix == "" {
		prefix = naming.DevicePrefix
	}
	if err := naming.ValidateDevicePrefix(prefix); err != nil {
		return nil, err
	}
	if start > end {
		return nil, errors.New("TUN allocation start must not exceed end")
	}
	return &TunAllocator{
		prefix: prefix, start: start, end: end,
		devices: SystemDevices, allocated: map[int]bool{},
	}, nil
}

// SetDeviceLister overrides host interface enumeration (tests).
func (a *TunAllocator) SetDeviceLister(l DeviceLister) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.devices = l
}

// Prefix returns the device-name pool this allocator draws from.
func (a *TunAllocator) Prefix() string { return a.prefix }

// Allocate reserves a device and returns its name plus a release func. The
// release func is safe to call once; typically deferred.
func (a *TunAllocator) Allocate() (device string, release func(), err error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	present := map[string]bool{}
	if a.devices != nil {
		present = a.devices()
	}
	occupied := 0
	for i := a.start; i <= a.end; i++ {
		if a.allocated[i] {
			continue
		}
		name := naming.DeviceName(a.prefix, i)
		// Already on the host: a leftover from a crashed run, a second
		// free-proxy instance, or an unrelated program. Either way it is not
		// ours to take.
		if present[name] {
			occupied++
			continue
		}
		a.allocated[i] = true
		idx := i
		// Released exactly once, whatever the caller does. A second call would
		// otherwise free whichever allocation holds the index by then, handing
		// two concurrent probes the same device.
		var once sync.Once
		return name, func() {
			once.Do(func() {
				a.mu.Lock()
				delete(a.allocated, idx)
				a.mu.Unlock()
			})
		}, nil
	}
	total := a.end - a.start + 1
	if occupied > 0 {
		return "", nil, fmt.Errorf(
			"no free %s* probe device in range %d-%d (%d of %d already exist on this host); "+
				"widen FREE_PROXY_TEST_TUN_START/END or change FREE_PROXY_PROBE_DEVICE_PREFIX",
			a.prefix, a.start, a.end, occupied, total)
	}
	return "", nil, fmt.Errorf("all %d %s* probe devices are in use; lower FREE_PROXY_MAX_PROBE_CONCURRENCY or widen the range",
		total, a.prefix)
}
