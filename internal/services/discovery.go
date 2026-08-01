package services

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/masteralanlab/free-proxy/internal/domain"
	"github.com/masteralanlab/free-proxy/internal/store"
)

// ProviderPort is the discovery source abstraction (satisfied by vpngate.Provider).
type ProviderPort interface {
	Name() string
	Discover(ctx context.Context) ([]domain.DiscoveredNode, error)
}

// parseStatsProvider is optionally implemented by providers to expose row stats.
type parseStatsProvider interface {
	ParseStats() (total, valid, dup, malformed, missing int)
}

// DiscoveryService fetches nodes from a provider and stores them.
type DiscoveryService struct {
	provider ProviderPort
	nodes    *store.NodeRepository
}

// NewDiscoveryService constructs a DiscoveryService.
func NewDiscoveryService(provider ProviderPort, nodes *store.NodeRepository) *DiscoveryService {
	return &DiscoveryService{provider: provider, nodes: nodes}
}

// Discover fetches, filters blacklisted, upserts, and snapshots provider presence.
func (s *DiscoveryService) Discover(ctx context.Context) (domain.DiscoveryResult, error) {
	slog.Info("starting node discovery", "module", "discovery", "provider", s.provider.Name())
	nodes, err := s.provider.Discover(ctx)
	if err != nil {
		return domain.DiscoveryResult{}, err
	}

	identities := make([]string, 0, len(nodes))
	for _, n := range nodes {
		id := n.ProviderIdentity
		if id == "" {
			id = n.Provider + ":" + n.IPAddress
		}
		identities = append(identities, id)
	}

	blacklist, err := s.nodes.ActiveBlacklistIDs(ctx)
	if err != nil {
		return domain.DiscoveryResult{}, err
	}
	kept := nodes[:0]
	for _, n := range nodes {
		if !blacklist[n.ID] {
			kept = append(kept, n)
		}
	}

	stored, err := s.nodes.UpsertDiscovered(ctx, kept)
	if err != nil {
		return domain.DiscoveryResult{}, err
	}

	result := domain.DiscoveryResult{
		Provider:   s.provider.Name(),
		Discovered: len(kept),
		Stored:     stored,
	}
	if sp, ok := s.provider.(parseStatsProvider); ok {
		total, valid, dup, malformed, missing := sp.ParseStats()
		result.TotalRows = &total
		result.ValidRows = &valid
		result.DuplicateRows = &dup
		result.MalformedRows = &malformed
		result.MissingFieldRows = &missing
	}
	// A successful provider response is an authoritative snapshot of the usable
	// nodes returned by that provider. Malformed or incomplete individual rows
	// are intentionally excluded from the current pool; keeping the previous
	// snapshot current would otherwise make the visible pool grow forever.
	if err := s.nodes.MarkProviderSnapshot(ctx, s.provider.Name(), identities); err != nil {
		return domain.DiscoveryResult{}, err
	}
	slog.Info("discovery complete", "module", "discovery", "discovered", result.Discovered, "stored", result.Stored)
	return result, nil
}

// DiscoverJob is the JobFunc form of Discover.
func (s *DiscoveryService) DiscoverJob(ctx context.Context) (map[string]any, error) {
	res, err := s.Discover(ctx)
	if err != nil {
		return nil, err
	}
	return toMap(res)
}

func toMap(v any) (map[string]any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}
