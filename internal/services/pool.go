package services

import (
	"context"
	"sort"

	"github.com/masteralanlab/free-proxy/internal/domain"
	"github.com/masteralanlab/free-proxy/internal/store"
)

// ProxyPoolService selects and filters usable nodes per routing settings.
type ProxyPoolService struct {
	nodes    *store.NodeRepository
	settings *store.SettingsRepository
}

// NewProxyPoolService constructs a ProxyPoolService.
func NewProxyPoolService(nodes *store.NodeRepository, settings *store.SettingsRepository) *ProxyPoolService {
	return &ProxyPoolService{nodes: nodes, settings: settings}
}

// SelectBest returns the best eligible node, or nil when none/disabled.
func (s *ProxyPoolService) SelectBest(ctx context.Context, excludeNodeID string) (*domain.ProxyNodeRead, error) {
	settings, err := s.settings.Get(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.ConnectionEnabled {
		return nil, nil
	}
	candidates, err := s.nodes.ListNodes(ctx, store.NodeFilter{Status: string(domain.NodeReady), CurrentOnly: true}, 1000, 0)
	if err != nil {
		return nil, err
	}
	candidates = ApplyFilters(candidates, settings, false)
	if excludeNodeID != "" {
		filtered := candidates[:0]
		for _, n := range candidates {
			if n.ID != excludeNodeID {
				filtered = append(filtered, n)
			}
		}
		candidates = filtered
	}
	SortCandidates(candidates, settings)
	if len(candidates) == 0 {
		return nil, nil
	}
	best := candidates[0]
	return &best, nil
}

// ValidateAllowed errors if the node is disallowed by current routing settings.
func (s *ProxyPoolService) ValidateAllowed(ctx context.Context, node domain.ProxyNodeRead) error {
	settings, err := s.settings.Get(ctx)
	if err != nil {
		return err
	}
	if !settings.ConnectionEnabled {
		return domain.ErrDisabled
	}
	if len(ApplyFilters([]domain.ProxyNodeRead{node}, settings, false)) == 0 {
		return domain.ErrRoutingMismatch
	}
	return nil
}

// Statistics returns pool-wide counts.
func (s *ProxyPoolService) Statistics(ctx context.Context) (domain.PoolStatistics, error) {
	return s.nodes.Statistics(ctx)
}

// ApplyFilters narrows nodes by routing mode and IP-type policy.
func ApplyFilters(nodes []domain.ProxyNodeRead, settings domain.ProxySettings, includeUnknownIPType bool) []domain.ProxyNodeRead {
	out := make([]domain.ProxyNodeRead, 0, len(nodes))
	fixedID := ""
	if settings.FixedNodeID != nil {
		fixedID = *settings.FixedNodeID
	}
	favorites := map[string]bool{}
	for _, id := range settings.FavoriteNodeIDs {
		favorites[id] = true
	}
	for _, n := range nodes {
		switch settings.RoutingMode {
		case domain.PolicyFixed:
			if n.ID != fixedID {
				continue
			}
		case domain.PolicyCountry:
			if settings.ForceCountry != "" &&
				domain.NormalizeCountry(n.Country) != domain.NormalizeCountry(settings.ForceCountry) {
				continue
			}
		case domain.PolicyFavorites:
			if !favorites[n.ID] {
				continue
			}
		}
		switch settings.RoutingIPType {
		case domain.RoutingResidential:
			if !(n.IPType == domain.IpResidential || n.IPType == domain.IpMobile ||
				(includeUnknownIPType && n.IPType == domain.IpUnknown)) {
				continue
			}
		case domain.RoutingHosting:
			if !(n.IPType == domain.IpHosting || (includeUnknownIPType && n.IPType == domain.IpUnknown)) {
				continue
			}
		}
		out = append(out, n)
	}
	return out
}

// SortCandidates orders nodes by the effective selection key for settings.
func SortCandidates(nodes []domain.ProxyNodeRead, settings domain.ProxySettings) {
	sort.SliceStable(nodes, func(i, j int) bool {
		return lessFor(nodes[i], nodes[j], settings)
	})
}

func residentialRank(n domain.ProxyNodeRead) int {
	if n.IPType == domain.IpResidential || n.IPType == domain.IpMobile {
		return 0
	}
	return 1
}

func effLatency(n domain.ProxyNodeRead) int {
	if n.LatencyMS > 0 {
		return n.LatencyMS
	}
	return 999999
}

func lessFor(a, b domain.ProxyNodeRead, settings domain.ProxySettings) bool {
	if settings.RoutingMode == domain.PolicyResidentialFirst {
		if ra, rb := residentialRank(a), residentialRank(b); ra != rb {
			return ra < rb
		}
		if la, lb := effLatency(a), effLatency(b); la != lb {
			return la < lb
		}
		if a.SourceScore != b.SourceScore {
			return a.SourceScore > b.SourceScore
		}
		return a.SourceSpeedBPS > b.SourceSpeedBPS
	}
	if la, lb := effLatency(a), effLatency(b); la != lb {
		return la < lb
	}
	if a.SourceScore != b.SourceScore {
		return a.SourceScore > b.SourceScore
	}
	if a.SourceSpeedBPS != b.SourceSpeedBPS {
		return a.SourceSpeedBPS > b.SourceSpeedBPS
	}
	return residentialRank(a) < residentialRank(b)
}
