package services

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/masteralanlab/free-proxy/internal/config"
	"github.com/masteralanlab/free-proxy/internal/domain"
	"github.com/masteralanlab/free-proxy/internal/netx"
	"github.com/masteralanlab/free-proxy/internal/store"
)

// MonitorState tracks a background loop's heartbeat for diagnostics.
type MonitorState struct {
	mu                  sync.Mutex
	lastHeartbeatAt     time.Time
	lastSuccessAt       time.Time
	lastError           string
	consecutiveFailures int
}

// Heartbeat records a loop iteration outcome.
func (s *MonitorState) Heartbeat(success bool, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	s.lastHeartbeatAt = now
	if success {
		s.lastSuccessAt = now
		s.lastError = ""
		s.consecutiveFailures = 0
	} else {
		s.lastError = errMsg
		s.consecutiveFailures++
	}
}

// AsMap renders the state for the diagnostics payload.
func (s *MonitorState) AsMap() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	var hb, ok any
	if !s.lastHeartbeatAt.IsZero() {
		hb = s.lastHeartbeatAt.Format(time.RFC3339)
	}
	if !s.lastSuccessAt.IsZero() {
		ok = s.lastSuccessAt.Format(time.RFC3339)
	}
	var lastErr any
	if s.lastError != "" {
		lastErr = s.lastError
	}
	return map[string]any{
		"last_heartbeat_at":    hb,
		"last_success_at":      ok,
		"last_error":           lastErr,
		"consecutive_failures": s.consecutiveFailures,
	}
}

// HealthService checks the proxy exit and recovers by rotating on failure.
type HealthService struct {
	cfg          *config.Config
	checker      *netx.HealthChecker
	nodes        *store.NodeRepository
	settingsRepo *store.SettingsRepository
	gateway      *GatewayService
	autoSwitch   *AutoSwitchService
}

// NewHealthService constructs a HealthService.
func NewHealthService(cfg *config.Config, checker *netx.HealthChecker, nodes *store.NodeRepository, settingsRepo *store.SettingsRepository, gateway *GatewayService, autoSwitch *AutoSwitchService) *HealthService {
	return &HealthService{cfg: cfg, checker: checker, nodes: nodes, settingsRepo: settingsRepo, gateway: gateway, autoSwitch: autoSwitch}
}

// Check runs a health check; when recover is set, it rotates away from a failing exit.
func (h *HealthService) Check(ctx context.Context, recover bool) domain.ProxyHealthResult {
	result := h.checker.Check()
	if result.OK && result.ExitIP != nil {
		h.gateway.UpdateHealth(*result.ExitIP, result.LatencyMS)
	} else {
		h.gateway.UpdateHealth("", 0)
	}
	active := h.gateway.Status().ActiveNodeID
	if result.OK || active == nil || !recover {
		return result
	}
	settings, err := h.settingsRepo.Get(ctx)
	if err != nil || !settings.ConnectionEnabled {
		return result
	}
	if settings.RoutingMode == domain.PolicyFixed {
		slog.Warn("fixed node failed health check; retrying in place", "module", "health", "node", *active)
		_, _ = h.gateway.Activate(ctx, *active)
	} else {
		msg := "Proxy health check failed"
		if result.Error != nil {
			msg = *result.Error
		}
		_ = h.nodes.Blacklist(ctx, *active, msg, h.cfg.InvalidBackoff())
		slog.Warn("active node entered cooldown after health failure", "module", "health", "node", *active)
		_, _ = h.autoSwitch.Switch(ctx)
	}
	return result
}

// Recover restores an exit when the gateway has none. Check only rotates away
// from an *active* node, so once a rotation exhausts its candidates and leaves
// the gateway exitless, nothing else brings it back: that teardown is deliberate,
// which keeps the unexpected-exit handler silent too. Without this the proxy goes
// on listening and answering with no upstream, however healthy the pool becomes.
func (h *HealthService) Recover(ctx context.Context) {
	settings, err := h.settingsRepo.Get(ctx)
	if err != nil || !settings.ConnectionEnabled {
		return
	}
	slog.Warn("gateway has no exit node; attempting recovery", "module", "health")
	_, _ = h.autoSwitch.Switch(ctx)
}

// HealthMonitor periodically runs Check.
type HealthMonitor struct {
	cfg     *config.Config
	health  *HealthService
	gateway *GatewayService
	State   MonitorState
}

// NewHealthMonitor constructs a HealthMonitor.
func NewHealthMonitor(cfg *config.Config, health *HealthService, gateway *GatewayService) *HealthMonitor {
	return &HealthMonitor{cfg: cfg, health: health, gateway: gateway}
}

// Run loops until ctx is cancelled.
func (m *HealthMonitor) Run(ctx context.Context) {
	t := time.NewTicker(m.cfg.HealthCheckInterval())
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if m.gateway.Status().ActiveNodeID != nil {
				m.health.Check(ctx, true)
			} else {
				m.health.Recover(ctx)
			}
			m.State.Heartbeat(true, "")
		}
	}
}
