package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/masteralanlab/free-proxy/internal/web"
)

// registerFrontend serves the embedded React build, falling back to index.html
// for client-side routes. The secret-path middleware has already stripped the
// prefix, so assets are addressed from the root.
func registerFrontend(e *echo.Echo, _ *Deps) {
	sub, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(sub))
	index, _ := fs.ReadFile(sub, "index.html")

	handler := func(c *echo.Context) error {
		p := strings.TrimPrefix(c.Request().URL.Path, "/")
		if p != "" {
			if f, err := sub.Open(p); err == nil {
				_ = f.Close()
				fileServer.ServeHTTP(c.Response(), c.Request())
				return nil
			}
		}
		return c.Blob(http.StatusOK, "text/html; charset=utf-8", index)
	}
	e.GET("/", handler)
	e.GET("/*", handler)
}
