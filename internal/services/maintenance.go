package services

import (
	"context"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/masteralanlab/free-proxy/internal/config"
	"github.com/masteralanlab/free-proxy/internal/domain"
	"github.com/masteralanlab/free-proxy/internal/store"
)

// MaintenanceService runs the periodic discover→probe→(auto-connect) cycle.
type MaintenanceService struct {
	cfg          *config.Config
	nodes        *store.NodeRepository
	settingsRepo *store.SettingsRepository
	discovery    *DiscoveryService
	probe        *ProbeService
	pool         *ProxyPoolService
	gateway      *GatewayService
	autoSwitch   *AutoSwitchService
	coordinator  *Coordinator
	mu           sync.Mutex
}

// NewMaintenanceService constructs a MaintenanceService.
func NewMaintenanceService(cfg *config.Config, nodes *store.NodeRepository, settingsRepo *store.SettingsRepository,
	discovery *DiscoveryService, probe *ProbeService, pool *ProxyPoolService, gateway *GatewayService,
	autoSwitch *AutoSwitchService, coordinator *Coordinator) *MaintenanceService {
	return &MaintenanceService{
		cfg: cfg, nodes: nodes, settingsRepo: settingsRepo, discovery: discovery, probe: probe,
		pool: pool, gateway: gateway, autoSwitch: autoSwitch, coordinator: coordinator,
	}
}

// Run performs one maintenance cycle under the operation lock.
func (m *MaintenanceService) Run(ctx context.Context) (domain.MaintenanceResult, error) {
	var res domain.MaintenanceResult
	err := m.coordinator.Run(ctx, "maintenance", false, func(ctx context.Context) error {
		var e error
		res, e = m.run(ctx)
		return e
	})
	return res, err
}

// RunJob is the JobFunc form of Run.
func (m *MaintenanceService) RunJob(ctx context.Context) (map[string]any, error) {
	res, err := m.Run(ctx)
	if err != nil {
		return nil, err
	}
	return toMap(res)
}

func (m *MaintenanceService) run(ctx context.Context) (domain.MaintenanceResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	slog.Info("starting periodic maintenance", "module", "maintenance")
	_ = m.nodes.ClearExpiredBlacklist(ctx)
	// A provider fetch that fails must not cost the cycle its probe pass: probing
	// reads the stored pool and does not need the provider to answer. Aborting
	// here used to drop a whole 200-node pass over a transient network error.
	//
	// The purge is the one step that does depend on the fetch. last_seen_at only
	// advances on a successful discovery, so running it after a failure reads an
	// outage as the entire pool going stale — and once the outage outlasts the
	// grace window that deletes every node, all of them still working.
	discovery, err := m.discovery.Discover(ctx)
	if err != nil {
		slog.Warn("node discovery failed; probing the stored pool and skipping the stale purge",
			"module", "maintenance", "err", err)
	} else {
		_, _ = m.nodes.PurgeStaleNodes(ctx, m.cfg.StaleNodeGrace())
	}

	settings, err := m.settingsRepo.Get(ctx)
	if err != nil {
		return domain.MaintenanceResult{}, err
	}
	initialTested := map[string]bool{}
	probed := 0

	if settings.ConnectionEnabled && settings.RoutingMode != domain.PolicyFixed && m.gateway.Status().ActiveNodeID == nil {
		candidates, err := m.candidateNodes(ctx, false)
		if err != nil {
			return domain.MaintenanceResult{}, err
		}
		candidates = ApplyFilters(candidates, settings, true)
		if settings.RoutingMode == domain.PolicySmart {
			sortSmartCandidates(candidates, true)
		} else {
			sort.SliceStable(candidates, func(i, j int) bool { return probeLess(candidates[i], candidates[j], settings) })
		}
		limit := m.cfg.InitialConnectTestLimit
		if limit > len(candidates) {
			limit = len(candidates)
		}
		var fastIDs []string
		for _, n := range candidates[:limit] {
			fastIDs = append(fastIDs, n.ID)
			initialTested[n.ID] = true
		}
		if len(fastIDs) > 0 {
			results, _ := m.probe.ProbeMany(ctx, fastIDs)
			probed += len(results)
			if anyAvailable(results) {
				_, _ = m.autoSwitch.Switch(ctx)
				if m.gateway.Status().ActiveNodeID != nil {
					return m.result(ctx, discovery.Discovered, probed)
				}
			}
		}
	}

	all, err := m.candidateNodes(ctx, true)
	if err != nil {
		return domain.MaintenanceResult{}, err
	}
	activeID := ""
	if a := m.gateway.Status().ActiveNodeID; a != nil {
		activeID = *a
	}
	var eligible []domain.ProxyNodeRead
	for _, n := range all {
		if !initialTested[n.ID] && n.ID != activeID {
			eligible = append(eligible, n)
		}
	}
	remaining := probeSlice(eligible)
	if len(remaining) > 0 {
		results, _ := m.probe.ProbeMany(ctx, remaining)
		probed += len(results)
	}

	if settings.ConnectionEnabled && m.gateway.Status().ActiveNodeID == nil {
		if settings.RoutingMode == domain.PolicyFixed && settings.FixedNodeID != nil && *settings.FixedNodeID != "" {
			_, _ = m.gateway.Activate(ctx, *settings.FixedNodeID)
		} else {
			_, _ = m.autoSwitch.Switch(ctx)
		}
	}
	return m.result(ctx, discovery.Discovered, probed)
}

// probeBudget caps how many nodes one maintenance cycle hands to the OpenVPN
// probe. A cycle holds the operation lock from start to finish, so an uncapped
// pass would block manual node switching for as long as it ran — and that
// duration used to be bounded only by how big the pool happened to be. With
// maintenance running every few hours, this budget still walks the whole pool
// within a day.
const probeBudget = 200

// probeSlice picks this cycle's share of the candidates, least-recently-probed
// first so coverage rotates and nodes that have never been probed go first.
func probeSlice(nodes []domain.ProxyNodeRead) []string {
	sort.SliceStable(nodes, func(i, j int) bool {
		a, b := nodes[i].LastProbedAt, nodes[j].LastProbedAt
		if (a == nil) != (b == nil) {
			return a == nil
		}
		if a == nil {
			return false
		}
		return a.Before(*b)
	})
	if len(nodes) > probeBudget {
		nodes = nodes[:probeBudget]
	}
	out := make([]string, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, n.ID)
	}
	return out
}

// candidateNodes lists the nodes worth spending an OpenVPN handshake on. Hosts
// the last liveness sweep could not reach are excluded: probing them costs the
// full connect timeout each and cannot succeed. They come back on their own if a
// later sweep reaches them, which resets the counter this filter reads.
func (m *MaintenanceService) candidateNodes(ctx context.Context, includeUnavailable bool) ([]domain.ProxyNodeRead, error) {
	nodes, err := m.nodes.ListNodes(ctx, store.NodeFilter{ReachableOnly: true}, candidateFetchLimit, 0)
	if err != nil {
		return nil, err
	}
	out := nodes[:0]
	for _, n := range nodes {
		switch n.Status {
		case domain.NodeDiscovered, domain.NodeReady:
			out = append(out, n)
		case domain.NodeUnavailable:
			if includeUnavailable {
				out = append(out, n)
			}
		}
	}
	return out, nil
}

func (m *MaintenanceService) result(ctx context.Context, discovered, probed int) (domain.MaintenanceResult, error) {
	available, err := m.nodes.CountNodes(ctx, store.NodeFilter{Status: string(domain.NodeReady)})
	if err != nil {
		return domain.MaintenanceResult{}, err
	}
	res := domain.MaintenanceResult{Discovered: discovered, Probed: probed, Available: int(available)}
	if a := m.gateway.Status().ActiveNodeID; a != nil {
		res.ConnectedNodeID = a
	}
	return res, nil
}

func anyAvailable(results []domain.ProbeResult) bool {
	for _, r := range results {
		if r.Available {
			return true
		}
	}
	return false
}

func probeLess(a, b domain.ProxyNodeRead, settings domain.ProxySettings) bool {
	if settings.RoutingMode == domain.PolicySpeedFirst {
		if a.SourceSpeedBPS != b.SourceSpeedBPS {
			return a.SourceSpeedBPS > b.SourceSpeedBPS
		}
	}
	pa, pb := a.SourcePingMS, b.SourcePingMS
	if pa == 0 {
		pa = 999999
	}
	if pb == 0 {
		pb = 999999
	}
	if pa != pb {
		return pa < pb
	}
	if a.SourceScore != b.SourceScore {
		return a.SourceScore > b.SourceScore
	}
	if a.SourceSpeedBPS != b.SourceSpeedBPS {
		return a.SourceSpeedBPS > b.SourceSpeedBPS
	}
	return a.SourceSessions < b.SourceSessions
}

// MaintenanceMonitor runs maintenance on an interval, backing off when disconnected.
type MaintenanceMonitor struct {
	cfg         *config.Config
	maintenance *MaintenanceService
	gateway     *GatewayService
	State       MonitorState
}

// NewMaintenanceMonitor constructs a MaintenanceMonitor.
func NewMaintenanceMonitor(cfg *config.Config, maintenance *MaintenanceService, gateway *GatewayService) *MaintenanceMonitor {
	return &MaintenanceMonitor{cfg: cfg, maintenance: maintenance, gateway: gateway}
}

// Run loops until ctx is cancelled.
func (m *MaintenanceMonitor) Run(ctx context.Context) {
	for {
		success := false
		if _, err := m.maintenance.Run(ctx); err != nil {
			m.State.Heartbeat(false, err.Error())
			slog.Warn("maintenance cycle failed", "module", "maintenance", "err", err)
		} else {
			success = true
			m.State.Heartbeat(true, "")
		}
		delay := m.cfg.MaintenanceInterval()
		if !success && m.gateway.Status().ActiveNodeID == nil {
			delay = m.cfg.DisconnectedRetry()
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
	}
}
