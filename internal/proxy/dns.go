package proxy

import (
	"context"
	"net"
)

// newResolver returns a pure-Go resolver that dials dnsServer through the
// tunnel interface, preventing DNS leaks. It returns nil (system resolver) when
// dnsServer is empty, which is the common case in tests.
func newResolver(iface, dnsServer string) *net.Resolver {
	if dnsServer == "" {
		return nil
	}
	d := &net.Dialer{Control: bindControl(iface)}
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return d.DialContext(ctx, "udp", net.JoinHostPort(dnsServer, "53"))
		},
	}
}
