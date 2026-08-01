package api

import (
	"net"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/masteralanlab/free-proxy/internal/security"
)

// ExternalAccessGuard blocks non-loopback clients from the web admin when the
// admin toggle disables external web access. It returns 404 (like SecretPath) to
// avoid revealing the service. Registered before SecretPath. Uses the real TCP
// remote address (not X-Forwarded-For) so it cannot be spoofed by a header.
func ExternalAccessGuard(store *security.AdminConfigStore) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if store.Config().WebExternalAllowed() || isLoopbackRemote(c.Request().RemoteAddr) {
				return next(c)
			}
			return echo.NewHTTPError(http.StatusNotFound, "Not found")
		}
	}
}

func isLoopbackRemote(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

const (
	ctxAuthorized = "authorized"
	ctxSession    = "session_token"
)

// SecretPath is a pre-router middleware: requests must start with the secret
// path prefix (else 404, hiding the service), the prefix is stripped, and
// non-public API routes require a valid session (else 401).
func SecretPath(auth *security.AuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			prefix := "/" + auth.Store.Config().SecretPath
			req := c.Request()
			p := req.URL.Path
			if p == prefix {
				return c.Redirect(http.StatusTemporaryRedirect, prefix+"/")
			}
			if !strings.HasPrefix(p, prefix+"/") {
				return echo.NewHTTPError(http.StatusNotFound, "Not found")
			}
			stripped := strings.TrimPrefix(p, prefix)
			req.URL.Path = stripped

			token := sessionCookie(c)
			authed := auth.Sessions.Valid(token)
			c.Set(ctxAuthorized, authed)
			c.Set(ctxSession, token)

			if !authed && !isPublic(stripped) {
				return echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized")
			}
			return next(c)
		}
	}
}

// isPublic reports whether a stripped path bypasses session auth. All non-API
// paths (SPA + static) are public; only /api/* is protected, except login.
func isPublic(path string) bool {
	if path == "/api/v1/auth/login" {
		return true
	}
	return !strings.HasPrefix(path, "/api/")
}

func sessionCookie(c *echo.Context) string {
	cookie, err := c.Cookie("session")
	if err != nil || cookie == nil {
		return ""
	}
	return cookie.Value
}
