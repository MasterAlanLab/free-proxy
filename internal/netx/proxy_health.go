package netx

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/masteralanlab/free-proxy/internal/domain"
)

// HealthChecker verifies the local SOCKS5 proxy reaches the internet and reports
// the exit IP, by performing a real SOCKS5 CONNECT + HTTP GET through it.
type HealthChecker struct {
	proxyHost      string
	proxyPort      int
	username       string
	password       string
	connectTimeout time.Duration
}

// NewHealthChecker builds a HealthChecker.
func NewHealthChecker(proxyHost string, proxyPort int, username, password string, connectTimeout time.Duration) *HealthChecker {
	return &HealthChecker{proxyHost: proxyHost, proxyPort: proxyPort, username: username, password: password, connectTimeout: connectTimeout}
}

func (c *HealthChecker) authEnabled() bool { return c.username != "" || c.password != "" }

// Check tries each known IP echo endpoint through the proxy.
func (c *HealthChecker) Check() domain.ProxyHealthResult {
	lastErr := "Proxy exit check failed"
	for _, host := range []string{"ip.sb", "api.ipify.org"} {
		res, err := c.requestIP(host)
		if err == nil {
			return res
		}
		lastErr = err.Error()
	}
	return domain.ProxyHealthResult{OK: false, Error: &lastErr}
}

func (c *HealthChecker) requestIP(host string) (domain.ProxyHealthResult, error) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(c.proxyHost, fmt.Sprintf("%d", c.proxyPort)), c.connectTimeout)
	if err != nil {
		return domain.ProxyHealthResult{}, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(c.connectTimeout))

	// Greeting.
	methods := []byte{0x00}
	if c.authEnabled() {
		methods = []byte{0x00, 0x02}
	}
	if _, err := conn.Write(append([]byte{0x05, byte(len(methods))}, methods...)); err != nil {
		return domain.ProxyHealthResult{}, err
	}
	sel := make([]byte, 2)
	if _, err := io.ReadFull(conn, sel); err != nil {
		return domain.ProxyHealthResult{}, err
	}
	if sel[0] != 0x05 || sel[1] == 0xFF {
		return domain.ProxyHealthResult{}, fmt.Errorf("SOCKS5 proxy rejected health check auth")
	}
	if sel[1] == 0x02 {
		if err := c.authenticate(conn); err != nil {
			return domain.ProxyHealthResult{}, err
		}
	}

	// CONNECT host:80 (domain address type).
	hb := []byte(host)
	req := bytes.NewBuffer([]byte{0x05, 0x01, 0x00, 0x03, byte(len(hb))})
	req.Write(hb)
	_ = binary.Write(req, binary.BigEndian, uint16(80))
	if _, err := conn.Write(req.Bytes()); err != nil {
		return domain.ProxyHealthResult{}, err
	}
	reply := make([]byte, 4)
	if _, err := io.ReadFull(conn, reply); err != nil {
		return domain.ProxyHealthResult{}, err
	}
	if reply[1] != 0x00 {
		return domain.ProxyHealthResult{}, fmt.Errorf("SOCKS5 connect failed: %d", reply[1])
	}
	if err := consumeSocksAddr(conn, reply[3]); err != nil {
		return domain.ProxyHealthResult{}, err
	}

	// HTTP GET.
	if _, err := fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", host); err != nil {
		return domain.ProxyHealthResult{}, err
	}
	raw, err := io.ReadAll(io.LimitReader(bufio.NewReader(conn), 64*1024))
	if err != nil && len(raw) == 0 {
		return domain.ProxyHealthResult{}, err
	}
	head, body, ok := bytes.Cut(raw, []byte("\r\n\r\n"))
	statusLine, _, _ := bytes.Cut(head, []byte("\r\n"))
	if !ok || !bytes.Contains(statusLine, []byte(" 200 ")) {
		return domain.ProxyHealthResult{}, fmt.Errorf("proxy exit endpoint did not return HTTP 200")
	}
	exitIP := strings.TrimSpace(strings.SplitN(string(body), "\n", 2)[0])
	if net.ParseIP(exitIP) == nil {
		return domain.ProxyHealthResult{}, fmt.Errorf("invalid exit IP %q", exitIP)
	}
	ms := int(time.Since(start).Milliseconds())
	if ms < 1 {
		ms = 1
	}
	return domain.ProxyHealthResult{OK: true, ExitIP: &exitIP, LatencyMS: ms}, nil
}

func (c *HealthChecker) authenticate(conn net.Conn) error {
	u, p := []byte(c.username), []byte(c.password)
	msg := bytes.NewBuffer([]byte{0x01, byte(len(u))})
	msg.Write(u)
	msg.WriteByte(byte(len(p)))
	msg.Write(p)
	if _, err := conn.Write(msg.Bytes()); err != nil {
		return err
	}
	resp := make([]byte, 2)
	if _, err := io.ReadFull(conn, resp); err != nil {
		return err
	}
	if resp[0] != 0x01 || resp[1] != 0x00 {
		return fmt.Errorf("SOCKS5 health check authentication failed")
	}
	return nil
}

func consumeSocksAddr(conn net.Conn, atyp byte) error {
	switch atyp {
	case 0x01:
		_, err := io.ReadFull(conn, make([]byte, 4+2))
		return err
	case 0x04:
		_, err := io.ReadFull(conn, make([]byte, 16+2))
		return err
	case 0x03:
		l := make([]byte, 1)
		if _, err := io.ReadFull(conn, l); err != nil {
			return err
		}
		_, err := io.ReadFull(conn, make([]byte, int(l[0])+2))
		return err
	default:
		return fmt.Errorf("invalid SOCKS5 address type %d", atyp)
	}
}
