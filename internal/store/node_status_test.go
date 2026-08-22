package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/masteralanlab/free-proxy/internal/domain"
)

func newNodeRepos(t *testing.T) *Repos {
	t.Helper()
	db, err := Open("file:" + filepath.Join(t.TempDir(), "nodes.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	return NewRepos(db)
}

func node(id, ip string) domain.DiscoveredNode {
	return domain.DiscoveredNode{
		ID: id, Provider: "vpngate", ProviderIdentity: "vpngate:" + ip,
		IPAddress: ip, RemoteHost: ip, RemotePort: 1194, Transport: domain.TransportTCP,
		ConfigText: "remote " + ip + " 1194 tcp\n", FetchedAt: time.Now().UTC(),
	}
}

func statusOf(t *testing.T, repos *Repos, id string) string {
	t.Helper()
	var s string
	if err := repos.DB.QueryRow("SELECT status FROM proxy_nodes WHERE id = ?", id).Scan(&s); err != nil {
		t.Fatalf("read status for %s: %v", id, err)
	}
	return s
}

// A cooldown is a timeout, not a verdict: once it lapses the node must go back to
// what its last handshake proved, not to a blanket 'unavailable' that hides a
// working node from every selection path until some later probe stumbles on it.
func TestExpiredCooldownRestoresProbeVerdict(t *testing.T) {
	repos := newNodeRepos(t)
	ctx := context.Background()
	good, bad, fresh := node("good", "1.1.1.1"), node("bad", "2.2.2.2"), node("fresh", "3.3.3.3")
	if _, err := repos.Nodes.UpsertDiscovered(ctx, []domain.DiscoveredNode{good, bad, fresh}); err != nil {
		t.Fatal(err)
	}
	probedAt := time.Now().UTC()
	if err := repos.Nodes.UpdateProbeResult(ctx, good.ID, true, 20, probedAt); err != nil {
		t.Fatal(err)
	}
	if err := repos.Nodes.UpdateProbeResult(ctx, bad.ID, false, 0, probedAt); err != nil {
		t.Fatal(err)
	}
	// "fresh" is left unprobed, so it has no verdict to restore.

	for _, id := range []string{good.ID, bad.ID, fresh.ID} {
		if err := repos.Nodes.Blacklist(ctx, id, "health check failed", -time.Minute); err != nil {
			t.Fatal(err)
		}
		if got := statusOf(t, repos, id); got != string(domain.NodeCooldown) {
			t.Fatalf("%s status after blacklist = %q, want cooldown", id, got)
		}
	}

	if err := repos.Nodes.ClearExpiredBlacklist(ctx); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		good.ID:  string(domain.NodeReady),
		bad.ID:   string(domain.NodeUnavailable),
		fresh.ID: string(domain.NodeDiscovered),
	}
	for id, w := range want {
		if got := statusOf(t, repos, id); got != w {
			t.Errorf("%s status after cooldown expiry = %q, want %q", id, got, w)
		}
	}
}

// The sweep demotes a ready node whose endpoint stops answering. It must lift
// that demotion when the endpoint answers again, or one bad sweep strands a
// working node in 'unavailable' permanently — clearing the counter alone still
// leaves it out of selection.
func TestLivenessSweepRestoresNodeItDemoted(t *testing.T) {
	repos := newNodeRepos(t)
	ctx := context.Background()
	n := node("flappy", "4.4.4.4")
	if _, err := repos.Nodes.UpsertDiscovered(ctx, []domain.DiscoveredNode{n}); err != nil {
		t.Fatal(err)
	}
	if err := repos.Nodes.UpdateProbeResult(ctx, n.ID, true, 15, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	if err := repos.Nodes.RecordLiveness(ctx, nil, []string{n.ID}, "", now); err != nil {
		t.Fatal(err)
	}
	if got := statusOf(t, repos, n.ID); got != string(domain.NodeUnavailable) {
		t.Fatalf("status after failed dial = %q, want unavailable", got)
	}

	if err := repos.Nodes.RecordLiveness(ctx, []string{n.ID}, nil, "", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if got := statusOf(t, repos, n.ID); got != string(domain.NodeReady) {
		t.Errorf("status after the host answered again = %q, want ready", got)
	}
}

// A host answering a TCP dial says nothing about whether an OpenVPN handshake
// completes, so a node the probe rejected must stay rejected.
func TestLivenessSweepLeavesProbeFailuresAlone(t *testing.T) {
	repos := newNodeRepos(t)
	ctx := context.Background()
	n := node("handshake-fails", "5.5.5.5")
	if _, err := repos.Nodes.UpsertDiscovered(ctx, []domain.DiscoveredNode{n}); err != nil {
		t.Fatal(err)
	}
	if err := repos.Nodes.UpdateProbeResult(ctx, n.ID, false, 0, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := repos.Nodes.RecordLiveness(ctx, []string{n.ID}, nil, "", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if got := statusOf(t, repos, n.ID); got != string(domain.NodeUnavailable) {
		t.Errorf("status = %q, want unavailable: a dial is not a handshake", got)
	}
}
