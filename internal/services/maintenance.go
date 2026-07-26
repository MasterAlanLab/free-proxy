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
	discovery, err := m.discovery.Discover(ctx)
	if err != nil {
		return domain.MaintenanceResult{}, err
	}
	_, _ = m.nodes.PurgeStaleNodes(ctx, m.cfg.StaleNodeGrace())

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
		sort.SliceStable(candidates, func(i, j int) bool { return probeLess(candidates[i], candidates[j]) })
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
	var remaining []string
	for _, n := range all {
		if !initialTested[n.ID] && n.ID != activeID {
			remaining = append(remaining, n.ID)
		}
	}
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

func (m *MaintenanceService) candidateNodes(ctx context.Context, includeUnavailable bool) ([]domain.ProxyNodeRead, error) {
	nodes, err := m.nodes.ListNodes(ctx, store.NodeFilter{CurrentOnly: true}, 1000, 0)
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

func probeLess(a, b domain.ProxyNodeRead) bool {
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
