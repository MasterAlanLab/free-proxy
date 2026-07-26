// Package proxy implements the local SOCKS5/HTTP proxy gateway served on a
// single port. It sniffs the first byte to dispatch protocols, forwards through
// an OutboundConnector (bound to the tunnel), and enforces optional auth and a
// connection cap. It depends only on config values passed via Options.
package proxy

import (
	"context"
	"net"
	"strconv"
	"sync"
	"time"
)

// Options configures a Gateway.
type Options struct {
	Host           string
	Port           int
	Username       string
	Password       string
	MaxConnections int
	ConnectTimeout time.Duration
	IdleTimeout    time.Duration
}

// Gateway is the unified SOCKS5/HTTP proxy server.
type Gateway struct {
	opts      Options
	connector OutboundConnector
	sem       chan struct{}

	mu sync.Mutex
	ln net.Listener
}

// New creates a Gateway. MaxConnections defaults to 256 when unset.
func New(opts Options, connector OutboundConnector) *Gateway {
	max := opts.MaxConnections
	if max <= 0 {
		max = 256
	}
	return &Gateway{opts: opts, connector: connector, sem: make(chan struct{}, max)}
}

func (g *Gateway) authEnabled() bool {
	return g.opts.Username != "" || g.opts.Password != ""
}

// Start binds the listener and serves accepted connections in the background.
// It returns once the listener is bound so callers can read Addr.
func (g *Gateway) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", net.JoinHostPort(g.opts.Host, strconv.Itoa(g.opts.Port)))
	if err != nil {
		return err
	}
	g.mu.Lock()
	g.ln = ln
	g.mu.Unlock()
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()
	go g.acceptLoop(ctx, ln)
	return nil
}

func (g *Gateway) acceptLoop(ctx context.Context, ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go g.handle(ctx, conn)
	}
}

// Addr returns the bound listener address (or the configured one before Start).
func (g *Gateway) Addr() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.ln != nil {
		return g.ln.Addr().String()
	}
	return net.JoinHostPort(g.opts.Host, strconv.Itoa(g.opts.Port))
}

// Running reports whether the listener is bound.
func (g *Gateway) Running() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.ln != nil
}

// Stop closes the listener.
func (g *Gateway) Stop() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.ln != nil {
		err := g.ln.Close()
		g.ln = nil
		return err
	}
	return nil
}
