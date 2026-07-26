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
	"strings"
	"testing"
	"time"

	xproxy "golang.org/x/net/proxy"
)

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
