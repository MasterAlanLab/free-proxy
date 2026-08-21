package api

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
	"github.com/masteralanlab/free-proxy/internal/domain"
)

type structValidator struct{ v *validator.Validate }

func (s *structValidator) Validate(i any) error { return s.v.Struct(i) }

// NewServer builds the Echo application with routes, validation, secret-path
// middleware, error mapping, and the SPA fallback.
func NewServer(deps *Deps) *echo.Echo {
	e := echo.New()
	e.Validator = &structValidator{v: validator.New()}
	e.HTTPErrorHandler = errorHandler
	e.Pre(ExternalAccessGuard(deps.Auth.Store))
	e.Pre(SecretPath(deps.Auth))

	h := &Handlers{Deps: deps}
	g := e.Group("/api/v1")

	g.POST("/auth/login", h.Login)
	g.POST("/auth/logout", h.Logout)
	g.GET("/auth/config", h.AuthConfig)
	g.PUT("/auth/credentials", h.UpdateCredentials)

	g.GET("/proxies", h.ListProxies)
	g.POST("/proxies/discover", h.DiscoverProxies)
	g.POST("/proxies/refresh", h.RefreshProxies)
	g.POST("/proxies/sweep", h.SweepProxies)
	g.POST("/proxies/probe", h.ProbeMultiple)
	g.POST("/proxies/:id/probe", h.ProbeOne)
	g.GET("/proxies/:id/probes", h.ProbeHistory)
	g.POST("/proxies/:id/activate", h.ActivateProxy)
	g.POST("/proxies/:id/favorite", h.ToggleFavorite)
	g.GET("/proxies/:id/config", h.DownloadConfig)

	g.GET("/gateway/status", h.GatewayStatus)
	g.DELETE("/gateway/current", h.GatewayDisconnect)
	g.POST("/gateway/check", h.GatewayCheck)
	g.POST("/gateway/rotate", h.GatewayRotate)

	g.GET("/pool/statistics", h.PoolStatistics)
	g.GET("/jobs/:id", h.GetJob)

	g.GET("/settings", h.GetSettings)
	g.PUT("/settings", h.UpdateSettings)

	g.GET("/system/status", h.SystemStatus)
	g.GET("/system/diagnostics", h.SystemDiagnostics)
	g.POST("/system/dns/repair", h.DNSRepair)
	g.GET("/system/access", h.GetAccess)
	g.PUT("/system/access", h.UpdateAccess)
	g.GET("/system/config", h.GetSystemConfig)
	g.PUT("/system/config", h.UpdateSystemConfig)

	g.GET("/logs", h.GetLogs)
	g.GET("/logs/export", h.ExportLogs)

	registerFrontend(e, deps)
	return e
}

func errorHandler(c *echo.Context, err error) {
	status := http.StatusInternalServerError
	message := "internal server error"

	var he *echo.HTTPError
	switch {
	case errors.As(err, &he):
		status = he.Code
		message = fmt.Sprint(he.Message)
	case errors.Is(err, domain.ErrNotFound):
		status, message = http.StatusNotFound, err.Error()
	case errors.Is(err, domain.ErrOperationConflict):
		status, message = http.StatusConflict, err.Error()
	case errors.Is(err, domain.ErrDisabled), errors.Is(err, domain.ErrRoutingMismatch),
		errors.Is(err, domain.ErrProvider), errors.Is(err, domain.ErrNetworkOperation):
		status, message = http.StatusBadRequest, err.Error()
	}
	_ = c.JSON(status, map[string]any{"detail": message})
}

func restartProcess() {
	// Non-zero exit so systemd (Restart=on-failure) restarts the service after a
	// listener/credential change.
	os.Exit(3)
}
