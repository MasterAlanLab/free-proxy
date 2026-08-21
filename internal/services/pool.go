package services

import (
	"context"
	"sort"

	"github.com/masteralanlab/free-proxy/internal/domain"
	"github.com/masteralanlab/free-proxy/internal/store"
)

// candidateFetchLimit bounds how many rows the selection and probe paths pull in
// one go. It sits far above the pool's steady-state size on purpose: the pool is
// no longer capped at one provider snapshot, so a limit near the expected size
// would silently truncate the candidate set instead of failing visibly.
const candidateFetchLimit = 5000

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
	candidates, err := s.nodes.ListNodes(ctx, store.NodeFilter{Status: string(domain.NodeReady)}, candidateFetchLimit, 0)
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
	if settings.RoutingMode == domain.PolicySmart {
		sortSmartCandidates(nodes, false)
		return
	}
	sort.SliceStable(nodes, func(i, j int) bool {
		return lessFor(nodes[i], nodes[j], settings)
	})
}

// sortSmartCandidates ranks the pool using relative scores so latency,
// advertised speed, and VPN Gate session count contribute without their raw
// units overwhelming one another. Latency and speed carry 40% each; lower
// session count carries the remaining 20%.
func sortSmartCandidates(nodes []domain.ProxyNodeRead, sourceLatency bool) {
	if len(nodes) < 2 {
		return
	}
	latencies := make([]float64, len(nodes))
	speeds := make([]float64, len(nodes))
	sessions := make([]float64, len(nodes))
	for i, n := range nodes {
		latencies[i] = smartLatency(n, sourceLatency)
		speeds[i] = float64(effSpeed(n))
		sessions[i] = float64(n.SourceSessions)
	}
	minMax := func(values []float64) (float64, float64) {
		min, max := values[0], values[0]
		for _, value := range values[1:] {
			if value < min {
				min = value
			}
			if value > max {
				max = value
			}
		}
		return min, max
	}
	latMin, latMax := minMax(latencies)
	speedMin, speedMax := minMax(speeds)
	sessionsMin, sessionsMax := minMax(sessions)
	normalize := func(value, min, max float64) float64 {
		if max == min {
			return 0.5
		}
		return (value - min) / (max - min)
	}
	scores := make(map[string]float64, len(nodes))
	for i, n := range nodes {
		latencyScore := 1 - normalize(latencies[i], latMin, latMax)
		speedScore := normalize(speeds[i], speedMin, speedMax)
		sessionScore := 1 - normalize(sessions[i], sessionsMin, sessionsMax)
		scores[n.ID] = 0.4*latencyScore + 0.4*speedScore + 0.2*sessionScore
	}
	sort.SliceStable(nodes, func(i, j int) bool {
		if scores[nodes[i].ID] != scores[nodes[j].ID] {
			return scores[nodes[i].ID] > scores[nodes[j].ID]
		}
		return nodes[i].ID < nodes[j].ID
	})
}

func smartLatency(n domain.ProxyNodeRead, sourceLatency bool) float64 {
	if n.LatencyMS > 0 {
		return float64(n.LatencyMS)
	}
	if sourceLatency && n.SourcePingMS > 0 {
		return float64(n.SourcePingMS)
	}
	return 999999
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

func effSpeed(n domain.ProxyNodeRead) int64 {
	if n.SourceSpeedBPS > 0 {
		return n.SourceSpeedBPS
	}
	return -1
}

func lessFor(a, b domain.ProxyNodeRead, settings domain.ProxySettings) bool {
	if settings.RoutingMode == domain.PolicySpeedFirst {
		if sa, sb := effSpeed(a), effSpeed(b); sa != sb {
			return sa > sb
		}
		if la, lb := effLatency(a), effLatency(b); la != lb {
			return la < lb
		}
		if a.SourceScore != b.SourceScore {
			return a.SourceScore > b.SourceScore
		}
		return residentialRank(a) < residentialRank(b)
	}
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
