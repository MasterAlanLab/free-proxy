package services

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/masteralanlab/free-proxy/internal/domain"
	"github.com/masteralanlab/free-proxy/internal/store"
)

type fakeProvider struct {
	nodes []domain.DiscoveredNode
	stats [5]int
}

func (f *fakeProvider) Name() string { return "vpngate" }
func (f *fakeProvider) Discover(context.Context) ([]domain.DiscoveredNode, error) {
	return f.nodes, nil
}
func (f *fakeProvider) ParseStats() (int, int, int, int, int) {
	return f.stats[0], f.stats[1], f.stats[2], f.stats[3], f.stats[4]
}

func newTestRepos(t *testing.T) *store.Repos {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "test.db") + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := store.Open(dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := store.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return store.NewRepos(db)
}

func disc(id, ip string) domain.DiscoveredNode {
	return domain.DiscoveredNode{
		ID: id, Provider: "vpngate", ProviderIdentity: "vpngate:" + ip,
		IPAddress: ip, RemoteHost: ip, RemotePort: 1194, Transport: domain.TransportUDP,
		ConfigText: "remote " + ip + " 1194 udp\n", FetchedAt: time.Now().UTC(),
	}
}

func TestDiscoveryStoresNodes(t *testing.T) {
	repos := newTestRepos(t)
	provider := &fakeProvider{nodes: []domain.DiscoveredNode{disc("jp-1", "1.1.1.1"), disc("us-1", "2.2.2.2")}}
	svc := NewDiscoveryService(provider, repos.Nodes)
	ctx := context.Background()

	res, err := svc.Discover(ctx)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if res.Discovered != 2 || res.Stored != 2 {
		t.Fatalf("result = %+v", res)
	}
	total, err := repos.Nodes.CountNodes(ctx, store.NodeFilter{})
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if total != 2 {
		t.Fatalf("stored total = %d, want 2", total)
	}
}

func TestDiscoveryExcludesBlacklisted(t *testing.T) {
	repos := newTestRepos(t)
	provider := &fakeProvider{nodes: []domain.DiscoveredNode{disc("jp-1", "1.1.1.1"), disc("us-1", "2.2.2.2")}}
	svc := NewDiscoveryService(provider, repos.Nodes)
	ctx := context.Background()

	if _, err := svc.Discover(ctx); err != nil {
		t.Fatalf("first discover: %v", err)
	}
	if err := repos.Nodes.Blacklist(ctx, "jp-1", "manual", time.Hour); err != nil {
		t.Fatalf("blacklist: %v", err)
	}
	res, err := svc.Discover(ctx)
	if err != nil {
		t.Fatalf("second discover: %v", err)
	}
	if res.Discovered != 1 {
		t.Fatalf("discovered = %d, want 1 (blacklisted excluded)", res.Discovered)
	}
}

func TestDiscoverySnapshotMarksAbsent(t *testing.T) {
	repos := newTestRepos(t)
	ctx := context.Background()

	first := &fakeProvider{nodes: []domain.DiscoveredNode{disc("jp-1", "1.1.1.1"), disc("us-1", "2.2.2.2")}}
	if _, err := NewDiscoveryService(first, repos.Nodes).Discover(ctx); err != nil {
		t.Fatalf("first: %v", err)
	}
	// Second snapshot only contains one node; the other should be marked absent.
	second := &fakeProvider{nodes: []domain.DiscoveredNode{disc("jp-1", "1.1.1.1")}}
	if _, err := NewDiscoveryService(second, repos.Nodes).Discover(ctx); err != nil {
		t.Fatalf("second: %v", err)
	}
	listed, err := repos.Nodes.CountNodes(ctx, store.NodeFilter{ListedOnly: true})
	if err != nil {
		t.Fatalf("count listed: %v", err)
	}
	if listed != 1 {
		t.Fatalf("listed nodes = %d, want 1 (us-1 marked absent)", listed)
	}
	// Dropping out of the provider listing must not remove the node: the
	// provider rotates which servers it publishes, so absence is not evidence
	// the node is gone. Only the liveness sweep deletes.
	all, err := repos.Nodes.CountNodes(ctx, store.NodeFilter{})
	if err != nil {
		t.Fatalf("count all: %v", err)
	}
	if all != 2 {
		t.Fatalf("retained rows = %d, want 2", all)
	}
}

func TestStatisticsCountWholePool(t *testing.T) {
	repos := newTestRepos(t)
	ctx := context.Background()

	first := &fakeProvider{nodes: []domain.DiscoveredNode{disc("jp-1", "1.1.1.1"), disc("us-1", "2.2.2.2")}}
	if _, err := NewDiscoveryService(first, repos.Nodes).Discover(ctx); err != nil {
		t.Fatalf("first: %v", err)
	}
	if _, err := repos.DB.ExecContext(ctx, `UPDATE proxy_nodes
		SET status = 'ready', ip_type = 'residential', country = 'Japan'`); err != nil {
		t.Fatalf("classify nodes: %v", err)
	}

	// The second listing publishes only one of the two nodes. Both stay counted:
	// statistics describe the retained pool, which is what the node list shows.
	second := &fakeProvider{
		nodes: []domain.DiscoveredNode{disc("jp-1", "1.1.1.1")},
		stats: [5]int{2, 1, 0, 1, 0},
	}
	second.nodes[0].Country = "Japan"
	if _, err := NewDiscoveryService(second, repos.Nodes).Discover(ctx); err != nil {
		t.Fatalf("second: %v", err)
	}

	stats, err := repos.Nodes.Statistics(ctx)
	if err != nil {
		t.Fatalf("statistics: %v", err)
	}
	if stats.Total != 2 || stats.Ready != 2 || stats.Residential != 2 || stats.Countries != 1 {
		t.Fatalf("statistics = %+v, want both retained ready residential nodes", stats)
	}
	all, err := repos.Nodes.CountNodes(ctx, store.NodeFilter{})
	if err != nil {
		t.Fatalf("count pool: %v", err)
	}
	if int64(stats.Total) != all {
		t.Fatalf("statistics total = %d but list holds %d rows; they must agree", stats.Total, all)
	}
}

func TestFavoriteFilterIncludesRetainedHistory(t *testing.T) {
	repos := newTestRepos(t)
	ctx := context.Background()
	first := &fakeProvider{nodes: []domain.DiscoveredNode{disc("jp-1", "1.1.1.1"), disc("us-1", "2.2.2.2")}}
	if _, err := NewDiscoveryService(first, repos.Nodes).Discover(ctx); err != nil {
		t.Fatalf("first: %v", err)
	}
	if _, err := repos.Settings.ToggleFavorite(ctx, "us-1"); err != nil {
		t.Fatalf("favorite: %v", err)
	}
	second := &fakeProvider{nodes: []domain.DiscoveredNode{disc("jp-1", "1.1.1.1")}}
	if _, err := NewDiscoveryService(second, repos.Nodes).Discover(ctx); err != nil {
		t.Fatalf("second: %v", err)
	}

	listed, err := repos.Nodes.CountNodes(ctx, store.NodeFilter{FavoriteOnly: true, ListedOnly: true})
	if err != nil {
		t.Fatalf("count listed favorites: %v", err)
	}
	all, err := repos.Nodes.ListNodes(ctx, store.NodeFilter{FavoriteOnly: true}, 20, 0)
	if err != nil {
		t.Fatalf("list all favorites: %v", err)
	}
	if listed != 0 || len(all) != 1 || all[0].ID != "us-1" || all[0].SourcePresent {
		t.Fatalf("favorites listed=%d all=%+v, want retained unlisted us-1", listed, all)
	}
}
