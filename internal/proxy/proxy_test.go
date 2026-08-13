package proxy

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/masteralanlab/free-proxy/internal/netx"
	xproxy "golang.org/x/net/proxy"
)

type fixedAddr string

func (a fixedAddr) Network() string { return "tcp" }
func (a fixedAddr) String() string  { return string(a) }

type remoteAddrConn struct {
	net.Conn
	remote net.Addr
}

func (c remoteAddrConn) RemoteAddr() net.Addr { return c.remote }

type healthCheckConnector struct{}

func (healthCheckConnector) Dial(_ context.Context, _ string, _ int) (net.Conn, error) {
	proxySide, originSide := net.Pipe()
	go func() {
		defer originSide.Close()
		reader := bufio.NewReader(originSide)
		for {
			line, err := reader.ReadString('\n')
			if err != nil || line == "\r\n" {
				break
			}
		}
		_, _ = io.WriteString(originSide, "HTTP/1.1 200 OK\r\nContent-Length: 12\r\nConnection: close\r\n\r\n203.0.113.9\n")
	}()
	return proxySide, nil
}

// startGateway launches a gateway on an ephemeral port backed by a direct
// (non-tunnel) connector, and returns its address.
func startGateway(t *testing.T, opts Options) (*Gateway, string) {
	t.Helper()
	opts.Host = "127.0.0.1"
	opts.Port = 0
	opts.ConnectTimeout = 5 * time.Second
	opts.IdleTimeout = 10 * time.Second
	g := New(opts, NewConnectorWithDialer(&net.Dialer{Timeout: 5 * time.Second}))
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	if err := g.Start(ctx); err != nil {
		t.Fatalf("start gateway: %v", err)
	}
	return g, g.Addr()
}

func newEchoServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestSocks5Forward(t *testing.T) {
	target := newEchoServer(t, "socks-ok")
	_, addr := startGateway(t, Options{})

	dialer, err := xproxy.SOCKS5("tcp", addr, nil, xproxy.Direct)
	if err != nil {
		t.Fatalf("socks dialer: %v", err)
	}
	client := &http.Client{Transport: &http.Transport{Dial: dialer.Dial}}
	resp, err := client.Get(target.URL)
	if err != nil {
		t.Fatalf("get via socks: %v", err)
	}
	defer resp.Body.Close()
	got, _ := io.ReadAll(resp.Body)
	if string(got) != "socks-ok" {
		t.Fatalf("body = %q, want socks-ok", got)
	}
}

func TestSocks5AuthRequired(t *testing.T) {
	target := newEchoServer(t, "secret")
	_, addr := startGateway(t, Options{Username: "u", Password: "p"})

	// Wrong/absent credentials -> dial must fail during handshake.
	badDialer, err := xproxy.SOCKS5("tcp", addr, nil, xproxy.Direct)
	if err != nil {
		t.Fatalf("socks dialer: %v", err)
	}
	if _, err := badDialer.Dial("tcp", "127.0.0.1:1"); err == nil {
		t.Fatal("expected auth failure without credentials")
	}

	// Correct credentials -> success.
	auth := &xproxy.Auth{User: "u", Password: "p"}
	dialer, err := xproxy.SOCKS5("tcp", addr, auth, xproxy.Direct)
	if err != nil {
		t.Fatalf("socks dialer: %v", err)
	}
	client := &http.Client{Transport: &http.Transport{Dial: dialer.Dial}}
	resp, err := client.Get(target.URL)
	if err != nil {
		t.Fatalf("authed get: %v", err)
	}
	defer resp.Body.Close()
	got, _ := io.ReadAll(resp.Body)
	if string(got) != "secret" {
		t.Fatalf("body = %q, want secret", got)
	}
}

func TestLoopbackDynamicSOCKS5RequiresAuth(t *testing.T) {
	target := newEchoServer(t, "dynamic-socks-auth-ok")
	_, addr := startGateway(t, Options{
		AuthRequired: func() bool { return true },
		Authenticate: func(username, password string) bool {
			return username == "u" && password == "p"
		},
	})

	// A loopback connection follows the same configured authentication policy as
	// every other client.
	unauthenticated, err := xproxy.SOCKS5("tcp", addr, nil, xproxy.Direct)
	if err != nil {
		t.Fatalf("unauthenticated SOCKS dialer: %v", err)
	}
	if _, err := unauthenticated.Dial("tcp", target.Listener.Addr().String()); err == nil {
		t.Fatal("expected loopback SOCKS authentication failure without credentials")
	}

	authenticated, err := xproxy.SOCKS5("tcp", addr, &xproxy.Auth{User: "u", Password: "p"}, xproxy.Direct)
	if err != nil {
		t.Fatalf("authenticated SOCKS dialer: %v", err)
	}
	client := &http.Client{Transport: &http.Transport{Dial: authenticated.Dial}}
	resp, err := client.Get(target.URL)
	if err != nil {
		t.Fatalf("authenticated loopback GET: %v", err)
	}
	defer resp.Body.Close()
	got, _ := io.ReadAll(resp.Body)
	if string(got) != "dynamic-socks-auth-ok" {
		t.Fatalf("body = %q, want dynamic-socks-auth-ok", got)
	}
}

func TestLoopbackDynamicSOCKS5AcceptsAuthOnlyGreeting(t *testing.T) {
	_, addr := startGateway(t, Options{
		AuthRequired: func() bool { return true },
		Authenticate: func(username, password string) bool {
			return username == "u" && password == "p"
		},
	})
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Advertise only RFC 1929 username/password authentication. This is the
	// greeting used by clients that previously failed specifically on loopback.
	if _, err := conn.Write([]byte{0x05, 0x01, 0x02}); err != nil {
		t.Fatal(err)
	}
	selection := make([]byte, 2)
	if _, err := io.ReadFull(conn, selection); err != nil {
		t.Fatal(err)
	}
	if selection[0] != 0x05 || selection[1] != 0x02 {
		t.Fatalf("method selection = % x, want 05 02", selection)
	}

	if _, err := conn.Write([]byte{0x01, 0x01, 'u', 0x01, 'p'}); err != nil {
		t.Fatal(err)
	}
	authReply := make([]byte, 2)
	if _, err := io.ReadFull(conn, authReply); err != nil {
		t.Fatal(err)
	}
	if authReply[0] != 0x01 || authReply[1] != 0x00 {
		t.Fatalf("auth reply = % x, want 01 00", authReply)
	}
}

func TestInternalAuthenticationIsRestrictedToLoopback(t *testing.T) {
	g := New(Options{
		AuthRequired: func() bool { return true },
		Authenticate: func(username, password string) bool {
			return username == "user" && password == "password"
		},
		InternalAuthenticate: func(username, password string) bool {
			return username == "health" && password == "secret"
		},
	}, NewConnectorWithDialer(&net.Dialer{}))

	loopback := remoteAddrConn{remote: fixedAddr("127.0.0.1:12345")}
	external := remoteAddrConn{remote: fixedAddr("192.0.2.10:12345")}
	if !g.authenticate(loopback, "health", "secret") {
		t.Fatal("internal health credential should authenticate on loopback")
	}
	if g.authenticate(external, "health", "secret") {
		t.Fatal("internal health credential must be rejected outside loopback")
	}
	if !g.authenticate(loopback, "user", "password") || !g.authenticate(external, "user", "password") {
		t.Fatal("configured user credential should authenticate on every source address")
	}
}

func TestAuthenticatedInternalHealthCheck(t *testing.T) {
	g := New(Options{
		Host:         "127.0.0.1",
		Port:         0,
		AuthRequired: func() bool { return true },
		Authenticate: func(username, password string) bool {
			return username == "user" && password == "password"
		},
		InternalAuthenticate: func(username, password string) bool {
			return username == "health" && password == "secret"
		},
		ConnectTimeout: time.Second,
		IdleTimeout:    time.Second,
	}, healthCheckConnector{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := g.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer g.Stop()

	host, portText, err := net.SplitHostPort(g.Addr())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatal(err)
	}
	checker := netx.NewHealthChecker(host, port, "health", "secret", time.Second)
	result := checker.Check()
	if !result.OK || result.ExitIP == nil || *result.ExitIP != "203.0.113.9" {
		t.Fatalf("health result = %+v, want authenticated success", result)
	}
}

func TestHTTPForward(t *testing.T) {
	target := newEchoServer(t, "http-ok")
	_, addr := startGateway(t, Options{})

	pu, _ := url.Parse("http://" + addr)
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(pu)}}
	resp, err := client.Get(target.URL)
	if err != nil {
		t.Fatalf("http forward: %v", err)
	}
	defer resp.Body.Close()
	got, _ := io.ReadAll(resp.Body)
	if string(got) != "http-ok" {
		t.Fatalf("body = %q, want http-ok", got)
	}
}

func TestHTTPConnect(t *testing.T) {
	// Raw CONNECT to a plain TCP echo target, then send bytes through the tunnel.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		buf := make([]byte, 4)
		if _, err := io.ReadFull(c, buf); err == nil {
			_, _ = c.Write(buf) // echo
		}
	}()

	_, addr := startGateway(t, Options{})
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial gateway: %v", err)
	}
	defer conn.Close()
	fmt.Fprintf(conn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", ln.Addr(), ln.Addr())
	r := bufio.NewReader(conn)
	status, err := r.ReadString('\n')
	if err != nil || !strings.Contains(status, "200") {
		t.Fatalf("connect status = %q err=%v", status, err)
	}
	for { // consume headers up to blank line
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatalf("read headers: %v", err)
		}
		if line == "\r\n" {
			break
		}
	}
	if _, err := conn.Write([]byte("ping")); err != nil {
		t.Fatalf("write tunnel: %v", err)
	}
	got := make([]byte, 4)
	if _, err := io.ReadFull(r, got); err != nil {
		t.Fatalf("read tunnel: %v", err)
	}
	if string(got) != "ping" {
		t.Fatalf("tunnel echo = %q, want ping", got)
	}
}

func TestHTTPAuthRequired(t *testing.T) {
	_, addr := startGateway(t, Options{Username: "u", Password: "p"})
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	fmt.Fprintf(conn, "GET http://example.com/ HTTP/1.1\r\nHost: example.com\r\n\r\n")
	status, _ := bufio.NewReader(conn).ReadString('\n')
	if !strings.Contains(status, "407") {
		t.Fatalf("status = %q, want 407", status)
	}
}

func TestLoopbackDynamicHTTPRequiresAuth(t *testing.T) {
	target := newEchoServer(t, "dynamic-http-auth-ok")
	_, addr := startGateway(t, Options{
		AuthRequired: func() bool { return true },
		Authenticate: func(username, password string) bool {
			return username == "u" && password == "p"
		},
	})

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(conn, "GET %s HTTP/1.1\r\nHost: %s\r\n\r\n", target.URL, target.Listener.Addr())
	status, _ := bufio.NewReader(conn).ReadString('\n')
	_ = conn.Close()
	if !strings.Contains(status, "407") {
		t.Fatalf("unauthenticated status = %q, want 407", status)
	}

	proxyURL, _ := url.Parse("http://" + addr)
	proxyURL.User = url.UserPassword("u", "p")
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
	resp, err := client.Get(target.URL)
	if err != nil {
		t.Fatalf("authenticated loopback HTTP GET: %v", err)
	}
	defer resp.Body.Close()
	got, _ := io.ReadAll(resp.Body)
	if string(got) != "dynamic-http-auth-ok" {
		t.Fatalf("body = %q, want dynamic-http-auth-ok", got)
	}
}
