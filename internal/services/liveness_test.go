package services

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/masteralanlab/free-proxy/internal/domain"
	"github.com/masteralanlab/free-proxy/internal/store"
)

// discTCP builds a TCP node, the only kind the sweep dials.
func discTCP(id, ip string) domain.DiscoveredNode {
	n := disc(id, ip)
	n.Transport = domain.TransportTCP
	return n
}

// unseen is old enough to clear the staleness bar for both transports.
const unseen = udpDeleteAfterUnseen + time.Hour

// newSweeper wires a service whose dial verdict comes from reachable[addr]. It
// runs against the shipped thresholds, so these tests fail if those move.
func newSweeper(t *testing.T, nodes *store.NodeRepository, reachable map[string]bool) *LivenessService {
	t.Helper()
	s := NewLivenessService(nodes, nil)
	s.dial = func(_ context.Context, addr string, _ time.Duration) bool { return reachable[addr] }
	return s
}

// sweepToThreshold runs exactly the number of sweeps deletion requires.
func sweepToThreshold(t *testing.T, svc *LivenessService) domain.LivenessResult {
	t.Helper()
	var res domain.LivenessResult
	for i := 0; i < deleteAfterFailures; i++ {
		var err error
		if res, err = svc.Sweep(context.Background()); err != nil {
			t.Fatalf("sweep %d: %v", i+1, err)
		}
	}
	return res
}

func addrOf(ip string) string { return net.JoinHostPort(ip, strconv.Itoa(1194)) }

// seed stores nodes and pushes last_seen_at back by age so the staleness half of
// the deletion rule can be exercised without waiting.
func seed(t *testing.T, repos *store.Repos, age time.Duration, nodes ...domain.DiscoveredNode) {
	t.Helper()
	ctx := context.Background()
	if _, err := repos.Nodes.UpsertDiscovered(ctx, nodes); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if age == 0 {
		return
	}
	seenAt := time.Now().UTC().Add(-age).Format(time.RFC3339Nano)
	for _, n := range nodes {
		if _, err := repos.DB.ExecContext(ctx,
			"UPDATE proxy_nodes SET last_seen_at = ? WHERE id = ?", seenAt, n.ID); err != nil {
			t.Fatalf("age node %s: %v", n.ID, err)
		}
	}
}

func livenessFailures(t *testing.T, repos *store.Repos, id string) int {
	t.Helper()
	var n int
	if err := repos.DB.QueryRow("SELECT liveness_failures FROM proxy_nodes WHERE id = ?", id).Scan(&n); err != nil {
		t.Fatalf("read failures for %s: %v", id, err)
	}
	return n
}

func nodeExists(t *testing.T, repos *store.Repos, id string) bool {
	t.Helper()
	var n int
	if err := repos.DB.QueryRow("SELECT COUNT(*) FROM proxy_nodes WHERE id = ?", id).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", id, err)
	}
	return n == 1
}

func TestSweepCountsFailuresAndClearsOnRecovery(t *testing.T) {
	repos := newTestRepos(t)
	ctx := context.Background()
	seed(t, repos, 0, discTCP("jp-1", "1.1.1.1"), discTCP("us-1", "2.2.2.2"))

	reachable := map[string]bool{addrOf("1.1.1.1"): true}
	svc := newSweeper(t, repos.Nodes, reachable)

	res, err := svc.Sweep(ctx)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if res.Checked != 2 || res.Alive != 1 || res.Dead != 1 {
		t.Fatalf("sweep result = %+v, want 2 checked / 1 alive / 1 dead", res)
	}
	if got := livenessFailures(t, repos, "us-1"); got != 1 {
		t.Fatalf("us-1 failures = %d, want 1", got)
	}
	if got := livenessFailures(t, repos, "jp-1"); got != 0 {
		t.Fatalf("jp-1 failures = %d, want 0", got)
	}

	// A node that comes back must have its counter reset, otherwise a node that
	// flaps once every sweep would eventually cross the deletion threshold
	// despite being reachable most of the time.
	reachable[addrOf("2.2.2.2")] = true
	if _, err := svc.Sweep(ctx); err != nil {
		t.Fatalf("second sweep: %v", err)
	}
	if got := livenessFailures(t, repos, "us-1"); got != 0 {
		t.Fatalf("us-1 failures after recovery = %d, want 0", got)
	}
}

func TestSweepDemotesUnreachableReadyNodes(t *testing.T) {
	repos := newTestRepos(t)
	ctx := context.Background()
	seed(t, repos, 0, discTCP("jp-1", "1.1.1.1"), discTCP("us-1", "2.2.2.2"), discTCP("cool-1", "4.4.4.4"))
	if _, err := repos.DB.ExecContext(ctx,
		"UPDATE proxy_nodes SET status = 'ready' WHERE id IN ('jp-1','us-1')"); err != nil {
		t.Fatalf("mark ready: %v", err)
	}
	if _, err := repos.DB.ExecContext(ctx,
		"UPDATE proxy_nodes SET status = 'cooldown' WHERE id = 'cool-1'"); err != nil {
		t.Fatalf("mark cooldown: %v", err)
	}

	svc := newSweeper(t, repos.Nodes, map[string]bool{addrOf("1.1.1.1"): true})
	if _, err := svc.Sweep(ctx); err != nil {
		t.Fatalf("sweep: %v", err)
	}

	statusOf := func(id string) string {
		var s string
		if err := repos.DB.QueryRow("SELECT status FROM proxy_nodes WHERE id = ?", id).Scan(&s); err != nil {
			t.Fatalf("status %s: %v", id, err)
		}
		return s
	}
	if got := statusOf("us-1"); got != "unavailable" {
		t.Fatalf("us-1 status = %q, want unavailable: a node that refuses TCP cannot serve OpenVPN over TCP", got)
	}
	if got := statusOf("jp-1"); got != "ready" {
		t.Fatalf("jp-1 status = %q, want ready to survive a successful dial", got)
	}
	if got := statusOf("cool-1"); got != "cooldown" {
		t.Fatalf("cool-1 status = %q, want cooldown preserved; the sweep only demotes ready rows", got)
	}
}

func TestSweepResultGatesTheProbeBudget(t *testing.T) {
	repos := newTestRepos(t)
	ctx := context.Background()
	seed(t, repos, 0, discTCP("jp-1", "1.1.1.1"), discTCP("us-1", "2.2.2.2"))

	reachable := map[string]bool{addrOf("1.1.1.1"): true}
	svc := newSweeper(t, repos.Nodes, reachable)
	if _, err := svc.Sweep(ctx); err != nil {
		t.Fatalf("sweep: %v", err)
	}

	// Probing an unreachable host costs a full connect timeout and cannot
	// succeed, so it must drop out of the candidate set the probe pass reads.
	got, err := repos.Nodes.ListNodes(ctx, store.NodeFilter{ReachableOnly: true}, 100, 0)
	if err != nil {
		t.Fatalf("list reachable: %v", err)
	}
	if len(got) != 1 || got[0].ID != "jp-1" {
		t.Fatalf("reachable candidates = %+v, want only jp-1", got)
	}

	// Recovery must put it back without any manual intervention.
	reachable[addrOf("2.2.2.2")] = true
	if _, err := svc.Sweep(ctx); err != nil {
		t.Fatalf("second sweep: %v", err)
	}
	got, err = repos.Nodes.ListNodes(ctx, store.NodeFilter{ReachableOnly: true}, 100, 0)
	if err != nil {
		t.Fatalf("list reachable after recovery: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("reachable candidates after recovery = %d, want both nodes back", len(got))
	}
}

func TestSweepDeletesOnlyWhenBothSignalsAgree(t *testing.T) {
	repos := newTestRepos(t)
	ctx := context.Background()
	// stale-* dropped off the provider listing long ago; listed-* is still
	// published. Only stale-alive answers a dial.
	seed(t, repos, unseen, discTCP("stale-dead", "1.1.1.1"), discTCP("stale-alive", "3.3.3.3"))
	seed(t, repos, 0, discTCP("listed-dead", "2.2.2.2"))

	svc := newSweeper(t, repos.Nodes, map[string]bool{addrOf("3.3.3.3"): true})

	// Deletion needs deleteAfterFailures consecutive failures, so every sweep
	// before the last must delete nothing.
	for i := 1; i < deleteAfterFailures; i++ {
		res, err := svc.Sweep(ctx)
		if err != nil {
			t.Fatalf("sweep %d: %v", i, err)
		}
		if res.Deleted != 0 {
			t.Fatalf("deleted %d after %d failures, want 0 below the threshold of %d",
				res.Deleted, i, deleteAfterFailures)
		}
	}

	res, err := svc.Sweep(ctx)
	if err != nil {
		t.Fatalf("threshold sweep: %v", err)
	}
	if res.Deleted != 1 {
		t.Fatalf("deleted = %d, want exactly stale-dead", res.Deleted)
	}
	if nodeExists(t, repos, "stale-dead") {
		t.Fatal("stale-dead survived: unreachable and unlisted should be deleted")
	}
	if !nodeExists(t, repos, "listed-dead") {
		t.Fatal("listed-dead was deleted: the provider still publishes it, so failed dials alone are not enough")
	}
	if !nodeExists(t, repos, "stale-alive") {
		t.Fatal("stale-alive was deleted: it answers dials, so dropping off the listing alone is not enough")
	}
}

func TestSweepKeepsProtectedNodes(t *testing.T) {
	repos := newTestRepos(t)
	ctx := context.Background()
	seed(t, repos, unseen, discTCP("fav-1", "1.1.1.1"), discTCP("plain-1", "2.2.2.2"))
	if _, err := repos.Settings.ToggleFavorite(ctx, "fav-1"); err != nil {
		t.Fatalf("favorite: %v", err)
	}

	sweepToThreshold(t, newSweeper(t, repos.Nodes, nil))
	if !nodeExists(t, repos, "fav-1") {
		t.Fatal("favorite was deleted; favorites must outlive the deletion rule")
	}
	if nodeExists(t, repos, "plain-1") {
		t.Fatal("plain-1 survived: unreachable and unlisted should be deleted")
	}
}

func TestSweepSkipsUDPNodes(t *testing.T) {
	repos := newTestRepos(t)
	ctx := context.Background()
	// disc() builds UDP nodes. They have no cheap reachability check, so the
	// sweep must not dial them and must not count them as dead.
	seed(t, repos, unseen, disc("udp-1", "1.1.1.1"))

	svc := newSweeper(t, repos.Nodes, nil)
	res, err := svc.Sweep(ctx)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if res.Checked != 0 || res.Dead != 0 || res.Deleted != 0 {
		t.Fatalf("sweep result = %+v, want UDP node untouched", res)
	}
	if !nodeExists(t, repos, "udp-1") {
		t.Fatal("udp-1 was deleted by a sweep that cannot measure it")
	}
}
