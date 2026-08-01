package proxy

import (
	"context"
	"net"
	"strconv"
	"time"
)

// OutboundConnector dials destination targets on behalf of proxy clients. In
// production the concrete implementation binds sockets to the tunnel interface
// and resolves DNS through the tunnel; tests can substitute a plain dialer.
type OutboundConnector interface {
	Dial(ctx context.Context, host string, port int) (net.Conn, error)
}

// SocketConnector dials via a net.Dialer whose Control binds to the tunnel
// interface (Linux) and whose Resolver forces DNS through the tunnel.
type SocketConnector struct {
	dialer *net.Dialer
}

// NewSocketConnector builds a connector bound to iface, resolving names via
// dnsServer through the tunnel. An empty iface disables binding; an empty
// dnsServer uses the system resolver (useful for tests / dev on macOS).
func NewSocketConnector(iface, dnsServer string, connectTimeout time.Duration) *SocketConnector {
	return &SocketConnector{dialer: &net.Dialer{
		Timeout:  connectTimeout,
		Control:  bindControl(iface),
		Resolver: newResolver(iface, dnsServer),
	}}
}

// NewConnectorWithDialer wraps an explicit dialer, for tests.
func NewConnectorWithDialer(d *net.Dialer) *SocketConnector {
	return &SocketConnector{dialer: d}
}

// Dial connects to host:port.
func (c *SocketConnector) Dial(ctx context.Context, host string, port int) (net.Conn, error) {
	return c.dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(port)))
}
