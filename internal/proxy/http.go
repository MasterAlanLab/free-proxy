package proxy

import (
	"bufio"
	"context"
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"strings"
)

func (g *Gateway) serveHTTP(ctx context.Context, conn net.Conn, br *bufio.Reader, requireAuth bool) {
	req, err := http.ReadRequest(br)
	if err != nil {
		return
	}

	if requireAuth && !g.httpAuthOK(conn, req) {
		_, _ = conn.Write([]byte(
			"HTTP/1.1 407 Proxy Authentication Required\r\n" +
				"Proxy-Authenticate: Basic realm=\"free-proxy\"\r\n" +
				"Content-Length: 0\r\n\r\n"))
		return
	}

	if req.Method == http.MethodConnect {
		g.httpConnect(ctx, conn, req.Host)
		return
	}
	g.httpForward(ctx, conn, req)
}

func (g *Gateway) httpConnect(ctx context.Context, conn net.Conn, hostport string) {
	host, port := splitHostPort(hostport, 443)
	targetConn, err := g.connector.Dial(ctx, host, port)
	if err != nil {
		_, _ = conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\nContent-Length: 0\r\n\r\n"))
		return
	}
	if _, err := conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
		_ = targetConn.Close()
		return
	}
	relay(conn, targetConn, g.opts.IdleTimeout)
}

func (g *Gateway) httpForward(ctx context.Context, conn net.Conn, req *http.Request) {
	host := req.URL.Hostname()
	if host == "" {
		host, _ = splitHostPortRaw(req.Host)
	}
	if host == "" {
		return
	}
	port := 80
	if p := req.URL.Port(); p != "" {
		port = atoiDefault(p, 80)
	}
	targetConn, err := g.connector.Dial(ctx, host, port)
	if err != nil {
		_, _ = conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\nContent-Length: 0\r\n\r\n"))
		return
	}
	defer targetConn.Close()

	req.Header.Del("Proxy-Authorization")
	req.Header.Del("Proxy-Connection")
	req.Close = true
	// Write in origin form (URL.RequestURI) with Host header to the origin.
	if err := req.Write(targetConn); err != nil {
		return
	}
	_, _ = io.Copy(conn, targetConn)
}

func (g *Gateway) httpAuthOK(conn net.Conn, req *http.Request) bool {
	const prefix = "Basic "
	h := req.Header.Get("Proxy-Authorization")
	if !strings.HasPrefix(h, prefix) {
		return false
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(h, prefix))
	if err != nil {
		return false
	}
	user, pass, ok := strings.Cut(string(raw), ":")
	if !ok {
		return false
	}
	return g.authenticate(conn, user, pass)
}

func splitHostPort(hostport string, defPort int) (string, int) {
	host, portStr, err := net.SplitHostPort(hostport)
	if err != nil {
		return hostport, defPort
	}
	return host, atoiDefault(portStr, defPort)
}

func splitHostPortRaw(hostport string) (string, string) {
	host, port, err := net.SplitHostPort(hostport)
	if err != nil {
		return hostport, ""
	}
	return host, port
}

func atoiDefault(s string, def int) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return def
		}
		n = n*10 + int(r-'0')
	}
	if s == "" {
		return def
	}
	return n
}
