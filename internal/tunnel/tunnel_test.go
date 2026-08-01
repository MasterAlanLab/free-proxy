package tunnel

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/masteralanlab/free-proxy/internal/config"
	"github.com/masteralanlab/free-proxy/internal/domain"
)

func TestFailureCodeClassification(t *testing.T) {
	cases := []struct {
		line string
		want domain.TunnelFailureCode
	}{
		{"Cannot allocate TUN/TAP dev", domain.FailTunUnavailable},
		{"AUTH_FAILED received", domain.FailAuthFailed},
		{"RESOLVE: Cannot resolve host address: x", domain.FailDNSFailed},
		{"TLS key negotiation failed to occur", domain.FailTLSFailed},
		{"ERROR: Operation not permitted", domain.FailPermissionDenied},
		{"TCP: connect connection refused", domain.FailConnectionRefused},
		{"Options error: unrecognized option", domain.FailConfigError},
		// Benign pushed-option warnings from VPNGate nodes (client uses
		// --route-nopull and rejects these) must NOT be a config failure.
		{"Options error: option 'redirect-gateway' cannot be used in this context ([PUSH-OPTIONS])", domain.FailUnknown},
		{"Options error: option 'dhcp-option' cannot be used in this context ([PUSH-OPTIONS])", domain.FailUnknown},
		{"something totally unrelated", domain.FailUnknown},
	}
	for _, c := range cases {
		if got := FailureCode([]string{c.line}); got != c.want {
			t.Errorf("FailureCode(%q) = %s, want %s", c.line, got, c.want)
		}
	}
}

func TestIsReadyAndTerminal(t *testing.T) {
	if !IsReady("2026-01-01 Initialization Sequence Completed") {
		t.Error("expected ready")
	}
	if IsReady("still connecting") {
		t.Error("unexpected ready")
	}
	if !IsTerminalFailure("AUTH_FAILED") {
		t.Error("auth failure should be terminal")
	}
	if IsTerminalFailure("connection refused") {
		t.Error("refused should not be terminal")
	}
	// A VPNGate node pushing options the client rejects (--route-nopull) must
	// not abort the handshake as a terminal failure.
	if IsTerminalFailure("Options error: option 'redirect-gateway' cannot be used in this context ([PUSH-OPTIONS])") {
		t.Error("pushed-option warning must not be terminal")
	}
}

func TestBuildArgsVersionBranch(t *testing.T) {
	base := BuildParams{Executable: []string{"openvpn"}, ConfigFile: "/c.ovpn", AuthFile: "/a.txt", Device: "tun0"}

	base.Version = Version{2, 5}
	if !slices.Contains(BuildArgs(base), "--data-ciphers") {
		t.Error("2.5 should use --data-ciphers")
	}
	base.Version = Version{2, 4}
	if !slices.Contains(BuildArgs(base), "--ncp-ciphers") {
		t.Error("2.4 should use --ncp-ciphers")
	}
}

// TestConnectDoesNotDeadlock guards against the regression where Connect held
// m.mu and then called detectVersion (which also locks m.mu), self-deadlocking
// before openvpn ever started. With a harmless openvpn command the call must
// return quickly; a hang means the deadlock is back.
func TestConnectDoesNotDeadlock(t *testing.T) {
	cfg := &config.Config{
		OpenVPNCommand:            "true", // exits immediately, no real tunnel
		TunnelInterface:           "tun0",
		DataDir:                   t.TempDir(),
		OpenVPNConnectTimeoutSecs: 2,
	}
	m := NewManager(cfg)
	done := make(chan struct{})
	go func() {
		_ = m.Connect(context.Background(), "n1", "remote 1.2.3.4 1194\n")
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(20 * time.Second):
		t.Fatal("Manager.Connect deadlocked (did not return)")
	}
}
