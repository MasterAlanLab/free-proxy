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
	current, err := repos.Nodes.CountNodes(ctx, store.NodeFilter{CurrentOnly: true})
	if err != nil {
		t.Fatalf("count current: %v", err)
	}
	if current != 1 {
		t.Fatalf("current nodes = %d, want 1 (us-1 marked absent)", current)
	}
}

func TestStatisticsAndSnapshotOnlyUseCurrentNodes(t *testing.T) {
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

	// A rejected row is a row that is not usable in the latest snapshot. It must
	// not cause every node from the previous snapshot to remain current.
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
	if stats.Total != 1 || stats.Ready != 1 || stats.Residential != 1 || stats.Countries != 1 {
		t.Fatalf("current statistics = %+v, want one current ready residential node", stats)
	}
	all, err := repos.Nodes.CountNodes(ctx, store.NodeFilter{})
	if err != nil {
		t.Fatalf("count history: %v", err)
	}
	if all != 2 {
		t.Fatalf("retained rows = %d, want 2", all)
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

	current, err := repos.Nodes.CountNodes(ctx, store.NodeFilter{FavoriteOnly: true, CurrentOnly: true})
	if err != nil {
		t.Fatalf("count current favorites: %v", err)
	}
	all, err := repos.Nodes.ListNodes(ctx, store.NodeFilter{FavoriteOnly: true}, 20, 0)
	if err != nil {
		t.Fatalf("list all favorites: %v", err)
	}
	if current != 0 || len(all) != 1 || all[0].ID != "us-1" || all[0].SourcePresent {
		t.Fatalf("favorites current=%d all=%+v, want retained historical us-1", current, all)
	}
}
