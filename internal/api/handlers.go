package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/masteralanlab/free-proxy/internal/domain"
	"github.com/masteralanlab/free-proxy/internal/security"
	"github.com/masteralanlab/free-proxy/internal/store"
)

// Handlers implements the REST endpoints over the service layer.
type Handlers struct{ Deps *Deps }

// ---- auth -------------------------------------------------------------------

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handlers) Login(c *echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if !h.Deps.Auth.Verify(req.Username, req.Password) {
		return echo.NewHTTPError(http.StatusForbidden, "Incorrect username or password")
	}
	token, err := h.Deps.Auth.Sessions.Create()
	if err != nil {
		return err
	}
	c.SetCookie(&http.Cookie{
		Name: "session", Value: token, Path: h.cookiePath(),
		HttpOnly: true, SameSite: http.SameSiteLaxMode,
		MaxAge: int(h.Deps.Cfg.SessionTTL().Seconds()),
	})
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handlers) Logout(c *echo.Context) error {
	if token, ok := c.Get(ctxSession).(string); ok {
		h.Deps.Auth.Sessions.Remove(token)
	}
	c.SetCookie(&http.Cookie{Name: "session", Value: "", Path: h.cookiePath(), MaxAge: -1})
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handlers) AuthConfig(c *echo.Context) error {
	cfg := h.Deps.Auth.Store.Config()
	return c.JSON(http.StatusOK, map[string]any{
		"username":     cfg.Username,
		"secret_path":  cfg.SecretPath,
		"host":         cfg.Host,
		"port":         cfg.Port,
		"proxy_host":   cfg.ProxyHost,
		"proxy_port":   cfg.ProxyPort,
		"password_set": cfg.PasswordHash != "",
	})
}

type credentialsUpdate struct {
	Username   string `json:"username" validate:"required"`
	Password   string `json:"password"`
	SecretPath string `json:"secret_path" validate:"required,alphanum"`
	Host       string `json:"host" validate:"required"`
	Port       int    `json:"port" validate:"required,min=1,max=65535"`
	ProxyHost  string `json:"proxy_host"`
	ProxyPort  int    `json:"proxy_port"`
}

func (h *Handlers) UpdateCredentials(c *echo.Context) error {
	var req credentialsUpdate
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	prev := h.Deps.Auth.Store.Config()
	hash := prev.PasswordHash
	if req.Password != "" {
		var err error
		if hash, err = security.HashPassword(req.Password); err != nil {
			return err
		}
	}
	updated := security.AdminConfig{
		Username: req.Username, PasswordHash: hash, SecretPath: req.SecretPath,
		Host: req.Host, Port: req.Port,
		ProxyHost: firstNonEmpty(req.ProxyHost, prev.ProxyHost),
		ProxyPort: firstNonZero(req.ProxyPort, prev.ProxyPort),
	}
	if err := h.Deps.Auth.Store.Update(updated); err != nil {
		return err
	}
	reauth := updated.Username != prev.Username || updated.PasswordHash != prev.PasswordHash
	if reauth {
		h.Deps.Auth.Sessions.Clear()
	}
	restart := updated.Host != prev.Host || updated.Port != prev.Port ||
		updated.SecretPath != prev.SecretPath || updated.ProxyHost != prev.ProxyHost ||
		updated.ProxyPort != prev.ProxyPort
	if restart && h.Deps.Cfg.AllowProcessRestart {
		go func() { time.Sleep(2 * time.Second); restartProcess() }()
	}
	return c.JSON(http.StatusOK, map[string]any{"ok": true, "restart_needed": restart, "reauth_required": reauth})
}

func (h *Handlers) cookiePath() string {
	return "/" + h.Deps.Auth.Store.Config().SecretPath
}

// ---- proxies ----------------------------------------------------------------

func (h *Handlers) ListProxies(c *echo.Context) error {
	limit := clampInt(queryInt(c, "limit", 100), 1, 500)
	offset := maxInt(queryInt(c, "offset", 0), 0)
	includeHistory := c.QueryParam("include_history") == "true"
	filter := store.NodeFilter{
		IPType: c.QueryParam("ip_type"), Status: c.QueryParam("status"),
		Country: c.QueryParam("country"), Search: c.QueryParam("search"),
		CurrentOnly: !includeHistory,
	}
	ctx := c.Request().Context()
	items, err := h.Deps.Repos.Nodes.ListNodes(ctx, filter, limit, offset)
	if err != nil {
		return err
	}
	total, err := h.Deps.Repos.Nodes.CountNodes(ctx, filter)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, domain.ProxyNodePage{Items: items, Total: total, Limit: limit, Offset: offset})
}

func (h *Handlers) DiscoverProxies(c *echo.Context) error {
	job, err := h.Deps.Jobs.Submit(c.Request().Context(), "discover-proxies", h.Deps.Discovery.DiscoverJob)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusAccepted, job)
}

func (h *Handlers) RefreshProxies(c *echo.Context) error {
	if op := h.Deps.Coordinator.Current(); op != "" {
		return echo.NewHTTPError(http.StatusConflict, "Another network operation is running")
	}
	job, err := h.Deps.Jobs.Submit(c.Request().Context(), "refresh-proxies", h.Deps.Maintenance.RunJob)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusAccepted, job)
}

func (h *Handlers) ProbeMultiple(c *echo.Context) error {
	var req domain.ProbeManyRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if op := h.Deps.Coordinator.Current(); op != "" {
		return echo.NewHTTPError(http.StatusConflict, "Another network operation is running")
	}
	ids := dedupeNonEmpty(req.IDs)
	if len(ids) > h.Deps.Cfg.ManualTestNodeLimit {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("At most %d nodes can be tested at once", h.Deps.Cfg.ManualTestNodeLimit))
	}
	job, err := h.Deps.Jobs.Submit(c.Request().Context(), "probe-proxies", h.Deps.Probe.ProbeManyJob(ids))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusAccepted, job)
}

func (h *Handlers) ProbeOne(c *echo.Context) error {
	job, err := h.Deps.Jobs.Submit(c.Request().Context(), "probe-proxy", h.Deps.Probe.ProbeJob(c.Param("id")))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusAccepted, job)
}

func (h *Handlers) ProbeHistory(c *echo.Context) error {
	limit := clampInt(queryInt(c, "limit", 100), 1, 500)
	items, err := h.Deps.Repos.Probes.ListForNode(c.Request().Context(), c.Param("id"), limit)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, items)
}

func (h *Handlers) ActivateProxy(c *echo.Context) error {
	job, err := h.Deps.Jobs.Submit(c.Request().Context(), "activate-proxy", h.Deps.Gateway.ActivateJob(c.Param("id")))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusAccepted, job)
}

func (h *Handlers) ToggleFavorite(c *echo.Context) error {
	updated, err := h.Deps.Settings.ToggleFavorite(c.Request().Context(), c.Param("id"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"ok": true, "favorite_node_ids": updated.FavoriteNodeIDs})
}

func (h *Handlers) DownloadConfig(c *echo.Context) error {
	id := c.Param("id")
	target, err := h.Deps.Repos.Nodes.GetTarget(c.Request().Context(), id)
	if err != nil {
		return err
	}
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", id+".ovpn"))
	return c.Blob(http.StatusOK, "application/x-openvpn-profile", []byte(target.ConfigText))
}

// ---- gateway ----------------------------------------------------------------

func (h *Handlers) GatewayStatus(c *echo.Context) error {
	return c.JSON(http.StatusOK, h.Deps.Gateway.Status())
}

func (h *Handlers) GatewayDisconnect(c *echo.Context) error {
	h.Deps.Gateway.Disconnect(c.Request().Context())
	return c.NoContent(http.StatusNoContent)
}

func (h *Handlers) GatewayCheck(c *echo.Context) error {
	return c.JSON(http.StatusOK, h.Deps.Health.Check(c.Request().Context(), false))
}

func (h *Handlers) GatewayRotate(c *echo.Context) error {
	rotate := func(ctx context.Context) (map[string]any, error) {
		result, err := h.Deps.AutoSwitch.Switch(ctx)
		if err != nil {
			return nil, err
		}
		payload := map[string]any{"connected": result != nil && result.Success, "tunnel": nil}
		if result != nil {
			payload["tunnel"] = *result
		}
		return payload, nil
	}
	job, err := h.Deps.Jobs.Submit(c.Request().Context(), "rotate-gateway", rotate)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusAccepted, job)
}

// ---- pool / jobs ------------------------------------------------------------

func (h *Handlers) PoolStatistics(c *echo.Context) error {
	stats, err := h.Deps.Pool.Statistics(c.Request().Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, stats)
}

func (h *Handlers) GetJob(c *echo.Context) error {
	job, err := h.Deps.Jobs.Get(c.Request().Context(), c.Param("id"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, job)
}

// ---- settings ---------------------------------------------------------------

func (h *Handlers) GetSettings(c *echo.Context) error {
	s, err := h.Deps.Settings.Get(c.Request().Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, s)
}

func (h *Handlers) UpdateSettings(c *echo.Context) error {
	var req domain.ProxySettingsUpdate
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.RoutingMode == domain.PolicyCountry && req.ForceCountry == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "force_country is required for country mode")
	}
	if req.RoutingMode == domain.PolicyFixed && (req.FixedNodeID == nil || *req.FixedNodeID == "") {
		return echo.NewHTTPError(http.StatusBadRequest, "fixed_node_id is required for fixed mode")
	}
	updated, err := h.Deps.Settings.Update(c.Request().Context(), req)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, updated)
}

// ---- system -----------------------------------------------------------------

func (h *Handlers) SystemStatus(c *echo.Context) error {
	ctx := c.Request().Context()
	gw := h.Deps.Gateway.Status()
	nodes, _ := h.Deps.Repos.Nodes.CountNodes(ctx, store.NodeFilter{})
	monitors := map[string]any{
		"maintenance":    monitorPayload(&h.Deps.MaintenanceMon.State),
		"active_latency": monitorPayload(&h.Deps.ActiveLatencyMon.State),
		"health":         monitorPayload(&h.Deps.HealthMon.State),
	}
	running := map[string]any{}
	for k, v := range monitors {
		running[k] = v.(map[string]any)["running"]
	}
	op, waiting, _ := h.Deps.Coordinator.Snapshot()
	return c.JSON(http.StatusOK, map[string]any{
		"name":            h.Deps.Cfg.AppName,
		"version":         h.Deps.Version,
		"environment":     h.Deps.Cfg.Environment,
		"status":          "running",
		"nodes":           nodes,
		"gateway_running": gw.Running,
		"active_node_id":  gw.ActiveNodeID,
		"listeners": map[string]any{
			"web":    fmt.Sprintf("%s:%d", h.Deps.Cfg.WebHost, h.Deps.Cfg.WebPort),
			"socks5": gw.SocksListener,
			"http":   gw.HTTPListener,
		},
		"monitors":         running,
		"monitor_details":  monitors,
		"network_operation": map[string]any{"operation": nullString(op), "waiting": waiting},
	})
}

func (h *Handlers) SystemDiagnostics(c *echo.Context) error {
	return c.JSON(http.StatusOK, h.Deps.Diagnostics.Diagnose(c.Request().Context(), false))
}

func (h *Handlers) DNSRepair(c *echo.Context) error {
	res, err := h.Deps.Diagnostics.RepairDNS(c.Request().Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}

// ---- logs -------------------------------------------------------------------

func (h *Handlers) GetLogs(c *echo.Context) error {
	limit := clampInt(queryInt(c, "limit", 1000), 1, 5000)
	entries := h.Deps.Logs.Read(c.QueryParam("date"), c.QueryParam("level"), c.QueryParam("module"), limit)
	return c.JSON(http.StatusOK, map[string]any{"logs": entries})
}

func (h *Handlers) ExportLogs(c *echo.Context) error {
	entries := h.Deps.Logs.Read(c.QueryParam("date"), c.QueryParam("level"), c.QueryParam("module"), 5000)
	var b strings.Builder
	for _, e := range entries {
		b.WriteString(fmt.Sprintf(`{"timestamp":%q,"level":%q,"module":%q,"message":%q}`+"\n", e.Timestamp, e.Level, e.Module, e.Message))
	}
	date := c.QueryParam("date")
	if date == "" {
		date = "today"
	}
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", "free-proxy-"+date+".jsonl"))
	return c.Blob(http.StatusOK, "application/x-ndjson", []byte(b.String()))
}

// ---- helpers ----------------------------------------------------------------

func monitorPayload(state interface{ AsMap() map[string]any }) map[string]any {
	m := state.AsMap()
	healthy := m["last_heartbeat_at"] != nil && m["last_error"] == nil
	out := map[string]any{"running": true, "status": statusWord(healthy)}
	for k, v := range m {
		out[k] = v
	}
	return out
}

func statusWord(healthy bool) string {
	if healthy {
		return "healthy"
	}
	return "degraded"
}

func queryInt(c *echo.Context, name string, def int) int {
	v := c.QueryParam(name)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func dedupeNonEmpty(ids []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func firstNonZero(a, b int) int {
	if a != 0 {
		return a
	}
	return b
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
