package services

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/masteralanlab/free-proxy/internal/config"
	"github.com/masteralanlab/free-proxy/internal/domain"
	"github.com/masteralanlab/free-proxy/internal/netx"
	"github.com/masteralanlab/free-proxy/internal/proxy"
	"github.com/masteralanlab/free-proxy/internal/store"
	"github.com/masteralanlab/free-proxy/internal/tunnel"
)

// GatewayService owns the active exit: it activates/disconnects tunnels, installs
// policy routing, runs the local proxy, and reports gateway status.
type GatewayService struct {
	cfg          *config.Config
	nodes        *store.NodeRepository
	settingsRepo *store.SettingsRepository
	tunnel       *tunnel.Manager
	router       *netx.PolicyRouter
	proxy        *proxy.Gateway
	pool         *ProxyPoolService
	coordinator  *Coordinator
	runner       netx.CommandRunner

	opMu              sync.Mutex
	mu                sync.Mutex
	lastError         string
	activeLatencyMS   int
	exitIP            string
	exitLatencyMS     int
	connectionEnabled bool
	onUnexpectedExit  func(context.Context)
}

// NewGatewayService constructs a GatewayService.
func NewGatewayService(cfg *config.Config, nodes *store.NodeRepository, settingsRepo *store.SettingsRepository,
	mgr *tunnel.Manager, router *netx.PolicyRouter, pg *proxy.Gateway, pool *ProxyPoolService,
	coordinator *Coordinator, runner netx.CommandRunner) *GatewayService {
	if runner == nil {
		runner = netx.SystemCommandRunner{}
	}
	return &GatewayService{
		cfg: cfg, nodes: nodes, settingsRepo: settingsRepo, tunnel: mgr, router: router,
		proxy: pg, pool: pool, coordinator: coordinator, runner: runner, connectionEnabled: true,
	}
}

// Start cleans up stale tunnels/routes and launches the local proxy listener.
func (g *GatewayService) Start(ctx context.Context) error {
	g.tunnel.CleanupStaleProcesses()
	_ = g.router.Cleanup(ctx)
	// Devices left behind by a crashed run would otherwise make the allocator
	// (which now refuses names already present on the host) skip them forever.
	if removed := netx.ReclaimStaleDevices(ctx, g.runner, g.cfg.ProbeDevicePrefix); len(removed) > 0 {
		slog.Info("reclaimed leftover tunnel devices", "module", "gateway", "devices", removed)
	}
	// A shared routing table is not fatal — cleanup is attribution-based — but
	// the operator should know the id collides with another program.
	if foreign := g.router.TableConflict(ctx); foreign > 0 {
		slog.Warn("policy routing table is shared with another program; set FREE_PROXY_POLICY_ROUTING_TABLE to a free id",
			"module", "gateway", "table", g.router.Table(), "foreign_routes", foreign)
	}
	if g.cfg.ProxyEnabled {
		return g.proxy.Start(ctx)
	}
	return nil
}

// Activate brings up a node as the exit, serialized via the coordinator.
func (g *GatewayService) Activate(ctx context.Context, nodeID string) (domain.TunnelStartResult, error) {
	var res domain.TunnelStartResult
	err := g.coordinator.Run(ctx, "activate", false, func(ctx context.Context) error {
		g.opMu.Lock()
		defer g.opMu.Unlock()
		var e error
		res, e = g.activate(ctx, nodeID)
		return e
	})
	return res, err
}

func (g *GatewayService) activate(ctx context.Context, nodeID string) (domain.TunnelStartResult, error) {
	slog.Info("activating exit node", "module", "gateway", "node", nodeID)
	target, err := g.nodes.GetTarget(ctx, nodeID)
	if err != nil {
		return domain.TunnelStartResult{}, err
	}
	node, err := g.nodes.Get(ctx, nodeID)
	if err != nil {
		return domain.TunnelStartResult{}, err
	}
	_ = g.settingsRepo.SetConnectionEnabled(ctx, true)
	g.setConnectionEnabled(true)
	if err := g.pool.ValidateAllowed(ctx, node); err != nil {
		return domain.TunnelStartResult{}, err
	}

	// Release our own tunnel before testing the device name. The availability
	// check reads the device off running processes' command lines, so it cannot
	// tell our outgoing OpenVPN from a stranger's: with the incumbent still up,
	// every rotation failed as "in use by another running tunnel process" and
	// the caller blacklisted a healthy node for it. Connect tears the old tunnel
	// down regardless — doing it here only lets the check see the truth. The
	// validation above still runs first, so a rejected node costs no exit.
	g.tunnel.Disconnect()

	// Verify the active device name is actually free before OpenVPN gets it —
	// the same guarantee the probe allocator gives, which the active tunnel
	// previously went without.
	if err := netx.EnsureDeviceAvailable(ctx, g.runner, g.cfg.ProbeDevicePrefix, g.cfg.TunnelInterface); err != nil {
		g.setLastError(err.Error())
		slog.Warn("tunnel device unavailable", "module", "gateway", "device", g.cfg.TunnelInterface, "err", err)
		return domain.TunnelStartResult{}, err
	}

	result := g.tunnel.Connect(ctx, nodeID, target.ConfigText)
	if !result.Success {
		_ = g.nodes.MarkUnavailable(ctx, nodeID)
		g.setLastError(result.Message)
		slog.Warn("activation failed", "module", "gateway", "node", nodeID, "msg", result.Message)
		return result, nil
	}
	// Reaching here means the node completed its handshake, so a routing failure
	// is ours alone — marking the node unavailable for it would retire a working
	// exit over a local fault, and every candidate after it in turn.
	if err := g.router.Setup(ctx, g.cfg.TunnelInterface); err != nil {
		g.tunnel.Disconnect()
		g.setLastError(err.Error())
		slog.Warn("policy routing setup failed", "module", "gateway", "node", nodeID, "err", err)
		return domain.TunnelStartResult{}, err
	}
	g.setLastError("")
	g.mu.Lock()
	g.activeLatencyMS = 0
	g.mu.Unlock()
	slog.Info("exit node activated", "module", "gateway", "node", nodeID)
	return result, nil
}

// ActivateJob is the JobFunc form of Activate.
func (g *GatewayService) ActivateJob(nodeID string) JobFunc {
	return func(ctx context.Context) (map[string]any, error) {
		res, err := g.Activate(ctx, nodeID)
		if err != nil {
			return nil, err
		}
		return toMap(res)
	}
}

// Disconnect disables connections and tears down the exit.
func (g *GatewayService) Disconnect(ctx context.Context) {
	g.opMu.Lock()
	defer g.opMu.Unlock()
	_ = g.settingsRepo.SetConnectionEnabled(ctx, false)
	g.setConnectionEnabled(false)
	g.disconnectOnly(ctx)
}

// DisconnectOnly tears down the exit without changing the enabled flag.
func (g *GatewayService) DisconnectOnly(ctx context.Context) {
	g.disconnectOnly(ctx)
}

func (g *GatewayService) disconnectOnly(ctx context.Context) {
	activeID := g.tunnel.ActiveNodeID()
	_ = g.router.Cleanup(ctx)
	g.tunnel.Disconnect()
	g.mu.Lock()
	g.activeLatencyMS = 0
	g.exitIP = ""
	g.exitLatencyMS = 0
	g.mu.Unlock()
	if activeID != "" {
		slog.Info("exit node disconnected", "module", "gateway", "node", activeID)
	}
}

// SetConnectionEnabled records the master switch state.
func (g *GatewayService) SetConnectionEnabled(enabled bool) { g.setConnectionEnabled(enabled) }

func (g *GatewayService) setConnectionEnabled(enabled bool) {
	g.mu.Lock()
	g.connectionEnabled = enabled
	g.mu.Unlock()
}

func (g *GatewayService) setLastError(msg string) {
	g.mu.Lock()
	g.lastError = msg
	g.mu.Unlock()
}

// UpdateActiveLatency records the last measured active-node latency.
func (g *GatewayService) UpdateActiveLatency(ms int) {
	g.mu.Lock()
	g.activeLatencyMS = ms
	g.mu.Unlock()
}

// UpdateHealth records the last health-check exit IP and latency.
func (g *GatewayService) UpdateHealth(exitIP string, latencyMS int) {
	g.mu.Lock()
	g.exitIP = exitIP
	g.exitLatencyMS = latencyMS
	g.mu.Unlock()
}

// SetUnexpectedExitHandler wires the callback for unexpected tunnel exits.
func (g *GatewayService) SetUnexpectedExitHandler(h func(context.Context)) {
	g.mu.Lock()
	g.onUnexpectedExit = h
	g.mu.Unlock()
	g.tunnel.SetExitHandler(g.handleUnexpectedExit)
}

func (g *GatewayService) handleUnexpectedExit(code int) {
	if g.tunnel.ActiveNodeID() == "" {
		return
	}
	ctx := context.Background()
	g.setLastError(fmt.Sprintf("OpenVPN exited unexpectedly (code=%d)", code))
	_ = g.router.Cleanup(ctx)
	g.mu.Lock()
	g.activeLatencyMS = 0
	g.exitIP = ""
	g.exitLatencyMS = 0
	h := g.onUnexpectedExit
	g.mu.Unlock()
	g.tunnel.ClearExitedProcess()
	if h != nil {
		h(ctx)
	}
}

// Status assembles the current gateway status.
func (g *GatewayService) Status() domain.GatewayStatus {
	g.mu.Lock()
	lastError := g.lastError
	activeLatency := g.activeLatencyMS
	exitIP := g.exitIP
	exitLatency := g.exitLatencyMS
	enabled := g.connectionEnabled
	g.mu.Unlock()

	activeID := g.tunnel.ActiveNodeID()
	tunnelStatus := domain.TunnelIdle
	switch {
	case g.tunnel.ActiveRunning():
		tunnelStatus = domain.TunnelConnected
	case activeID != "":
		tunnelStatus = domain.TunnelFailed
	}
	listener := g.proxy.Addr()
	status := domain.GatewayStatus{
		Running:           g.proxy.Running(),
		TunnelStatus:      tunnelStatus,
		ProxyListener:     listener,
		SocksListener:     listener,
		HTTPListener:      listener,
		ActiveLatencyMS:   activeLatency,
		ExitLatencyMS:     exitLatency,
		ConnectionEnabled: enabled,
		MonitorStatus:     map[string]any{},
	}
	if activeID != "" {
		status.ActiveNodeID = &activeID
	}
	if lastError != "" {
		status.LastError = &lastError
	}
	if exitIP != "" {
		status.ExitIP = &exitIP
	}
	return status
}
