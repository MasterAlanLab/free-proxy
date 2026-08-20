package naming

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDevicePrefixIsPrivate is the core invariant behind issue #2: our device
// names must not be drawn from the shared tunN pool that every other tunnel
// program allocates from.
func TestDevicePrefixIsPrivate(t *testing.T) {
	if DevicePrefix == LegacyDevicePrefix {
		t.Fatalf("DevicePrefix %q is the shared pool; it must be project-private", DevicePrefix)
	}
	if HasDevicePrefix(ActiveDevice(), LegacyDevicePrefix) {
		t.Fatalf("active device %q falls inside the shared %s* pool", ActiveDevice(), LegacyDevicePrefix)
	}
	if err := ValidateDevicePrefix(DevicePrefix); err != nil {
		t.Fatalf("default prefix is invalid: %v", err)
	}
	if err := ValidateRoutingTable(DefaultRoutingTable); err != nil {
		t.Fatalf("default routing table is invalid: %v", err)
	}
	if DefaultRoutingTable == LegacyRoutingTable {
		t.Fatalf("default routing table %d is still the crowded legacy id", DefaultRoutingTable)
	}
}

func TestHasDevicePrefix(t *testing.T) {
	cases := []struct {
		device, prefix string
		want           bool
	}{
		{"fpx0", "fpx", true},
		{"fpx42", "fpx", true},
		{"fpx", "fpx", false},    // no index
		{"fpxa", "fpx", false},   // non-numeric suffix
		{"tun0", "fpx", false},   // someone else's
		{"fpx0x1", "fpx", false}, // not a plain index
		{"myfpx0", "fpx", false}, // must be a prefix, not a substring
		{"fpx0", "", false},
	}
	for _, c := range cases {
		if got := HasDevicePrefix(c.device, c.prefix); got != c.want {
			t.Errorf("HasDevicePrefix(%q, %q) = %v, want %v", c.device, c.prefix, got, c.want)
		}
	}
}

func TestValidateDeviceName(t *testing.T) {
	if err := ValidateDeviceName(strings.Repeat("a", MaxDeviceNameLen+1)); err == nil {
		t.Error("expected an over-long interface name to be rejected")
	}
	if err := ValidateDeviceName("bad name"); err == nil {
		t.Error("expected a name with a space to be rejected")
	}
	if err := ValidateDeviceName(ActiveDevice()); err != nil {
		t.Errorf("default device name rejected: %v", err)
	}
}

func TestValidateRoutingTableRejectsReserved(t *testing.T) {
	for _, table := range []int{0, -1, 253, 254, 255} {
		if err := ValidateRoutingTable(table); err == nil {
			t.Errorf("table %d should be rejected", table)
		}
	}
}

// TestNoHardcodedDeviceNames keeps the fix from eroding. Every identifier we
// place into a host-global namespace has to come from this package, so a new
// call site cannot quietly reintroduce a literal `tun0` the way config.go,
// setup.go and tunnel_test.go each had one before.
func TestNoHardcodedDeviceNames(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	banned := []string{`"tun0"`, `"tun1"`, `envDefault:"tun`}
	scanned := 0

	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "dist", "frontend", "docs":
				return filepath.SkipDir
			}
			return nil
		}
		// Production Go code only: tests legitimately name foreign devices to
		// assert we leave them alone, and this package owns the legacy names.
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if filepath.Dir(path) == filepath.Join(root, "internal", "naming") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		scanned++
		for _, needle := range banned {
			if strings.Contains(string(data), needle) {
				rel, _ := filepath.Rel(root, path)
				t.Errorf("%s hardcodes %s; take the name from internal/naming instead", rel, needle)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if scanned == 0 {
		t.Fatal("scanned no Go files; the guard is not actually running")
	}
}
