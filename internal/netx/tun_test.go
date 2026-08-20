package netx

import (
	"strings"
	"testing"
)

func newTestAllocator(t *testing.T, start, end int, present ...string) *TunAllocator {
	t.Helper()
	a, err := NewTunAllocator("fpx", start, end)
	if err != nil {
		t.Fatalf("NewTunAllocator: %v", err)
	}
	set := map[string]bool{}
	for _, name := range present {
		set[name] = true
	}
	a.SetDeviceLister(func() map[string]bool { return set })
	return a
}

// TestAllocateSkipsDevicesPresentOnHost is the regression test for issue #2:
// the allocator used to track only its own bookkeeping, so it handed out a name
// another program already owned and OpenVPN died with "Cannot allocate TUN/TAP
// dev". A name that exists on the host is never ours to take.
func TestAllocateSkipsDevicesPresentOnHost(t *testing.T) {
	a := newTestAllocator(t, 1, 4, "fpx1", "fpx2")

	device, release, err := a.Allocate()
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	defer release()
	if device != "fpx3" {
		t.Fatalf("Allocate() = %q, want fpx3 (fpx1/fpx2 already exist)", device)
	}
}

func TestAllocateIsExclusive(t *testing.T) {
	a := newTestAllocator(t, 1, 2)

	first, releaseFirst, err := a.Allocate()
	if err != nil {
		t.Fatalf("first Allocate: %v", err)
	}
	second, releaseSecond, err := a.Allocate()
	if err != nil {
		t.Fatalf("second Allocate: %v", err)
	}
	if first == second {
		t.Fatalf("allocator handed out %q twice", first)
	}
	if _, _, err = a.Allocate(); err == nil {
		t.Fatal("expected exhaustion once the whole range is allocated")
	}

	releaseFirst()
	reused, releaseReused, err := a.Allocate()
	if err != nil {
		t.Fatalf("Allocate after release: %v", err)
	}
	if reused != first {
		t.Fatalf("released device %q was not reused, got %q", first, reused)
	}
	releaseSecond()
	releaseReused()
}

// A range fully occupied by foreign devices must say so, and say which knob
// moves us out of the way — the bare "no test TUN devices are available" gave
// the operator nothing to act on.
func TestAllocateExhaustedByHostDevicesExplainsWhy(t *testing.T) {
	a := newTestAllocator(t, 1, 2, "fpx1", "fpx2")

	_, _, err := a.Allocate()
	if err == nil {
		t.Fatal("expected allocation to fail")
	}
	for _, want := range []string{"already exist", "FREE_PROXY_PROBE_DEVICE_PREFIX"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

// A release func called twice must not free whatever allocation holds its index
// by then — that would hand two concurrent probes the same device.
func TestReleaseIsIdempotent(t *testing.T) {
	a := newTestAllocator(t, 1, 1)

	first, release, err := a.Allocate()
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	release()

	second, releaseSecond, err := a.Allocate()
	if err != nil {
		t.Fatalf("re-Allocate: %v", err)
	}
	defer releaseSecond()
	if second != first {
		t.Fatalf("expected the released index to be reused, got %q", second)
	}

	release() // stale call: must not free the live allocation above
	if _, _, err := a.Allocate(); err == nil {
		t.Fatal("a stale release freed a device that is still in use")
	}
}

func TestNewTunAllocatorValidatesInput(t *testing.T) {
	if _, err := NewTunAllocator("fpx", 5, 4); err == nil {
		t.Error("expected an inverted range to be rejected")
	}
	if _, err := NewTunAllocator(strings.Repeat("x", 20), 1, 2); err == nil {
		t.Error("expected an over-long prefix to be rejected")
	}
	a, err := NewTunAllocator("", 1, 2)
	if err != nil {
		t.Fatalf("empty prefix should fall back to the default: %v", err)
	}
	if a.Prefix() != "fpx" {
		t.Errorf("Prefix() = %q, want the project default", a.Prefix())
	}
}
