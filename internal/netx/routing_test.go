package netx

import (
	"context"
	"net"
	"runtime"
	"strings"
	"testing"
	"time"
)

// fakeRunner records every command and replies from canned output keyed by a
// substring of the joined command line.
type fakeRunner struct {
	replies map[string]string
	calls   [][]string
}

func (f *fakeRunner) Run(_ context.Context, args []string, _ time.Duration) (CommandResult, error) {
	f.calls = append(f.calls, args)
	joined := strings.Join(args, " ")
	for key, out := range f.replies {
		if strings.Contains(joined, key) {
			return CommandResult{Stdout: out}, nil
		}
	}
	return CommandResult{}, nil
}

func (f *fakeRunner) ran(substr string) bool {
	for _, call := range f.calls {
		if strings.Contains(strings.Join(call, " "), substr) {
			return true
		}
	}
	return false
}

func newTestRouter(runner CommandRunner) *PolicyRouter {
	r := NewPolicyRouter(runner, PolicyRouterConfig{Table: 9527, Interface: "fpx0", DevicePrefix: "fpx"})
	on := true
	r.supportedOverride = &on
	return r
}

// rp_filter is a host-global knob. We need loose mode on our own tunnel, and
// because the kernel takes max(conf.all, conf.<iface>), setting the device
// alone achieves that — while setting conf.all=2 would force every interface on
// the host into loose mode, disabling anti-spoofing for programs that never
// asked us to.
func TestSetupScopesRPFilterToOurDeviceOnly(t *testing.T) {
	runner := &fakeRunner{}
	router := newTestRouter(runner)

	if err := router.Setup(context.Background(), "fpx0"); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	if !runner.ran("sysctl -w net.ipv4.conf.fpx0.rp_filter=2") {
		t.Error("our own tunnel device was not set to loose mode")
	}
	for _, global := range []string{"conf.all.rp_filter=2", "conf.default.rp_filter=2"} {
		if runner.ran(global) {
			t.Errorf("must not touch the host-global %s", global)
		}
	}
}

// TestCleanupLeavesForeignEntriesAlone is the second half of the coexistence
// fix. Cleanup used to run a blanket `ip rule del table N` / `ip route flush
// table N`, so whenever another program shared the table id we deleted its
// rules and wiped its routes. Only entries pointing at one of our own devices
// may be touched.
func TestCleanupLeavesForeignEntriesAlone(t *testing.T) {
	runner := &fakeRunner{replies: map[string]string{
		"ip rule show": "0:\tfrom all lookup local\n" +
			"32764:\tfrom all oif wg0 lookup 9527\n" +
			"32765:\tfrom all oif fpx0 lookup 9527\n" +
			"32766:\tfrom all lookup main\n",
		"ip route show table 9527": "default dev fpx0 scope link\n10.0.0.0/24 dev wg0 scope link\n",
	}}
	router := newTestRouter(runner)

	if err := router.Cleanup(context.Background()); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}

	if !runner.ran("ip rule del oif fpx0 table 9527") {
		t.Error("our own policy rule was not removed")
	}
	if runner.ran("oif wg0") {
		t.Error("removed another program's policy rule")
	}
	if runner.ran("ip route flush") {
		t.Error("flushed a table shared with another program")
	}
	if !runner.ran("ip route del default dev fpx0 table 9527") {
		t.Error("our own route was not removed")
	}
	if runner.ran("dev wg0") {
		t.Error("removed another program's route")
	}
}

// With nothing foreign in the table a flush is still the right, thorough move —
// it also clears entries we could not parse.
func TestCleanupFlushesTableWeOwnOutright(t *testing.T) {
	runner := &fakeRunner{replies: map[string]string{
		"ip rule show":             "32765:\tfrom all oif fpx0 lookup 9527\n",
		"ip route show table 9527": "default dev fpx0 scope link\n",
	}}
	router := newTestRouter(runner)

	if err := router.Cleanup(context.Background()); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if !runner.ran("ip route flush table 9527") {
		t.Error("expected a flush when the table holds only our routes")
	}
}

// A crashed run leaves a rule behind pointing at a probe device rather than the
// configured active one; prefix attribution has to reclaim it.
func TestCleanupReclaimsStaleProbeRules(t *testing.T) {
	runner := &fakeRunner{replies: map[string]string{
		"ip rule show":             "32765:\tfrom all oif fpx7 lookup 9527\n",
		"ip route show table 9527": "default dev fpx7 scope link\n",
	}}
	router := newTestRouter(runner)

	if err := router.Cleanup(context.Background()); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if !runner.ran("ip rule del oif fpx7 table 9527") {
		t.Error("stale rule from a probe device was not reclaimed")
	}
}

func TestTableConflictCountsForeignRoutesOnly(t *testing.T) {
	runner := &fakeRunner{replies: map[string]string{
		"ip route show table 9527": "default dev fpx0 scope link\n10.0.0.0/24 dev wg0\n192.168.9.0/24 dev eth0\n",
	}}
	if got := newTestRouter(runner).TableConflict(context.Background()); got != 2 {
		t.Errorf("TableConflict() = %d, want 2", got)
	}

	clean := &fakeRunner{replies: map[string]string{
		"ip route show table 9527": "default dev fpx0 scope link\n",
	}}
	if got := newTestRouter(clean).TableConflict(context.Background()); got != 0 {
		t.Errorf("TableConflict() = %d, want 0", got)
	}
}

// iproute2 prints the rt_tables alias instead of the numeric id once the table
// is named, so rule matching has to accept both spellings.
func TestPolicyRuleMatchesTableAlias(t *testing.T) {
	rules := ParsePolicyRules("32765:\tfrom all oif fpx0 lookup free_proxy\n")
	if len(rules) != 1 {
		t.Fatalf("parsed %d rules, want 1", len(rules))
	}
	if !rules[0].MatchesTable(9527) {
		t.Error("alias form should match our table")
	}
	if rules[0].OIF != "fpx0" {
		t.Errorf("OIF = %q, want fpx0", rules[0].OIF)
	}
}

func TestParsePolicyRulesIgnoresRulesWithoutLookup(t *testing.T) {
	rules := ParsePolicyRules("0:\tfrom all lookup local\nnot a rule\n32766:\tfrom all lookup main\n")
	if len(rules) != 2 {
		t.Fatalf("parsed %d rules, want 2", len(rules))
	}
}

func TestParseTableRoutes(t *testing.T) {
	routes := ParseTableRoutes("default dev fpx0 scope link\n10.0.0.0/24 via 10.0.0.1 dev wg0\n")
	if len(routes) != 2 {
		t.Fatalf("parsed %d routes, want 2", len(routes))
	}
	if routes[0].Destination != "default" || routes[0].Device != "fpx0" {
		t.Errorf("route[0] = %+v", routes[0])
	}
	if routes[1].Destination != "10.0.0.0/24" || routes[1].Device != "wg0" {
		t.Errorf("route[1] = %+v", routes[1])
	}
}

// An upgrade has to clear the policy entries an abruptly-killed old release
// left pointing at tun0. They are inert while tun0 is absent, but would hijack
// a neighbour's traffic the day something else creates that device.
func TestCleanupOrphanedRulesRemovesEntriesForAbsentDevice(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("CleanupOrphanedRules is Linux-only")
	}
	const absent = "fpx-no-such-dev0" // guaranteed not to exist
	runner := &fakeRunner{replies: map[string]string{
		"ip rule show":            "32765:\tfrom all oif " + absent + " lookup 100\n32766:\tfrom all lookup main\n",
		"ip route show table 100": "default dev " + absent + " scope link\n10.0.0.0/24 dev wg0\n",
	}}

	removed := CleanupOrphanedRules(context.Background(), runner, absent)

	if len(removed) != 2 {
		t.Fatalf("removed %v, want the rule and its one route", removed)
	}
	if !runner.ran("ip rule del oif " + absent + " table 100") {
		t.Error("orphaned rule was not removed")
	}
	if !runner.ran("ip route del default dev " + absent + " table 100") {
		t.Error("orphaned route was not removed")
	}
	// The neighbour sharing table 100 is still none of our business.
	if runner.ran("dev wg0") {
		t.Error("removed another program's route from the shared table")
	}
}

// While the device is live we cannot tell our own leftover from a neighbour's
// active configuration, so we must not touch it.
func TestCleanupOrphanedRulesSkipsLiveDevice(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("CleanupOrphanedRules is Linux-only")
	}
	iface, err := net.Interfaces()
	if err != nil || len(iface) == 0 {
		t.Skip("no interfaces to test against")
	}
	runner := &fakeRunner{}

	if removed := CleanupOrphanedRules(context.Background(), runner, iface[0].Name); removed != nil {
		t.Errorf("removed %v for a live device", removed)
	}
	if len(runner.calls) != 0 {
		t.Errorf("no command should have run for a live device, got %v", runner.calls)
	}
}

// The active tunnel device gets the same pre-flight verification the probe
// allocator has. A name held by another program must be refused outright, not
// deleted and not handed to OpenVPN.
func TestEnsureDeviceAvailableRefusesForeignDevice(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("EnsureDeviceAvailable is Linux-only")
	}
	runner := &fakeRunner{}

	err := EnsureDeviceAvailable(context.Background(), runner, "fpx", "lo")
	if err == nil {
		t.Fatal("expected a device outside our namespace to be refused")
	}
	for _, want := range []string{"lo", "outside", "FREE_PROXY_TUNNEL_INTERFACE"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
	if len(runner.calls) != 0 {
		t.Errorf("must not try to delete a foreign device, ran %v", runner.calls)
	}
}

func TestEnsureDeviceAvailableAcceptsFreeName(t *testing.T) {
	runner := &fakeRunner{}
	if err := EnsureDeviceAvailable(context.Background(), runner, "fpx", "fpx-no-such-dev0"); err != nil {
		t.Errorf("a name nothing holds should be accepted, got %v", err)
	}
	if len(runner.calls) != 0 {
		t.Errorf("no command should run for a free name, ran %v", runner.calls)
	}
}

func TestRemoveDeviceRefusesForeignNames(t *testing.T) {
	runner := &fakeRunner{}
	if RemoveDevice(context.Background(), runner, "fpx", "tun0") {
		t.Error("RemoveDevice must refuse a name outside our prefix pool")
	}
	if len(runner.calls) != 0 {
		t.Errorf("no command should have run, got %v", runner.calls)
	}
}
