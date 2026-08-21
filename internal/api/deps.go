// Package api exposes the Echo HTTP server: the secret-path + session
// middleware, request handlers over the service layer, and the SPA static mount.
package api

import (
	"github.com/masteralanlab/free-proxy/internal/config"
	"github.com/masteralanlab/free-proxy/internal/logging"
	"github.com/masteralanlab/free-proxy/internal/security"
	"github.com/masteralanlab/free-proxy/internal/services"
	"github.com/masteralanlab/free-proxy/internal/store"
)

// Deps is the explicitly-wired dependency container (replaces app.state).
type Deps struct {
	Cfg     *config.Config
	Version string
	Repos   *store.Repos
	Auth    *security.AuthService
	Logs    *logging.Store

	Coordinator *services.Coordinator
	Jobs        *services.JobService
	Discovery   *services.DiscoveryService
	Probe       *services.ProbeService
	Gateway     *services.GatewayService
	Pool        *services.ProxyPoolService
	Settings    *services.SettingsService
	Health      *services.HealthService
	Diagnostics *services.DiagnosticsService
	Maintenance *services.MaintenanceService
	AutoSwitch  *services.AutoSwitchService
	Liveness    *services.LivenessService

	MaintenanceMon   *services.MaintenanceMonitor
	ActiveLatencyMon *services.ActiveLatencyMonitor
	HealthMon        *services.HealthMonitor
	LivenessMon      *services.LivenessMonitor
}
