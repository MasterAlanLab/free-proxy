package proxy

import (
	"bufio"
	"context"
	"net"
	"time"
)

// handle sniffs the first byte of a connection and dispatches to SOCKS5 (0x05)
// or HTTP (ASCII method letter). It enforces the connection cap first.
func (g *Gateway) handle(ctx context.Context, conn net.Conn) {
	if !g.allowClient(conn) {
		_ = conn.Close() // external client blocked by access policy
		return
	}
	select {
	case g.sem <- struct{}{}:
		defer func() { <-g.sem }()
	default:
		_ = conn.Close() // over capacity
		return
	}
	defer conn.Close()

	br := bufio.NewReader(conn)
	_ = conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	first, err := br.Peek(1)
	if err != nil {
		return
	}
	_ = conn.SetReadDeadline(time.Time{})

	switch {
	case first[0] == 0x05:
		g.serveSOCKS5(ctx, conn, br, g.requireProtocolAuth())
	case isHTTPStart(first[0]):
		g.serveHTTP(ctx, conn, br, g.requireProtocolAuth())
	default:
		// Unknown protocol: drop.
	}
}

func isHTTPStart(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}
