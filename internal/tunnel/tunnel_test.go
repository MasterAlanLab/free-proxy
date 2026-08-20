package tunnel

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/masteralanlab/free-proxy/internal/config"
	"github.com/masteralanlab/free-proxy/internal/domain"
	"github.com/masteralanlab/free-proxy/internal/naming"
)

// A TUN allocation failure is the symptom operators hit when another program
// owns the device name (issue #2). The message has to name the device and the
// setting that moves us, not just say "unavailable".
func TestFailureMessageNamesTheDevice(t *testing.T) {
	lines := []string{"ERROR: Cannot allocate TUN/TAP dev dynamically"}

	generic := FailureMessage(lines)
	if strings.Contains(generic, "fpx7") {
		t.Errorf("message without a device should stay generic, got %q", generic)
	}

	specific := FailureMessageFor(lines, "fpx7")
	for _, want := range []string{"fpx7", "already be taken by another program", "FREE_PROXY_TUNNEL_INTERFACE"} {
		if !strings.Contains(specific, want) {
			t.Errorf("message %q does not mention %q", specific, want)
		}
	}

	// Other failure codes must not be rewritten just because a device is known.
	authLines := []string{"AUTH_FAILED"}
	if FailureMessageFor(authLines, "fpx7") != FailureMessage(authLines) {
		t.Error("only TUN-unavailable failures should gain device context")
	}
}

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

// CleanupStaleProcesses signals processes, so "probably ours" is not good
// enough. Identification must be by exact argument, never by substring against
// a flattened command line.
func TestOwnsCommandLineRequiresExactMatch(t *testing.T) {
	dataDir := t.TempDir()
	m := NewManager(&config.Config{DataDir: dataDir})
	configsDir := m.cfg.ConfigsDir()

	ours := []string{
		"openvpn", "--config", configsDir + "/active-abc123.ovpn",
		"--dev", naming.ActiveDevice(), "--auth-user-pass", m.authFile,
	}
	if !m.ownsCommandLine(ours) {
		t.Error("our own tunnel was not recognised")
	}
	if !m.ownsCommandLine([]string{"openvpn", "--auth-user-pass", m.authFile}) {
		t.Error("the auth file alone should identify our process")
	}

	strangers := [][]string{
		// A sibling directory whose path contains ours as a substring: the old
		// strings.Contains check matched this and killed the process.
		{"openvpn", "--config", dataDir + "-backup/configs/other.ovpn"},
		// Our data dir mentioned in an unrelated argument.
		{"openvpn", "--log", dataDir + "/../elsewhere.log"},
		// A different program's tunnel entirely.
		{"openvpn", "--config", "/etc/openvpn/client.conf"},
		{},
	}
	for _, args := range strangers {
		if m.ownsCommandLine(args) {
			t.Errorf("claimed ownership of a stranger's process: %v", args)
		}
	}
}

func TestSplitCmdlinePreservesArgumentBoundaries(t *testing.T) {
	got := splitCmdline([]byte("openvpn\x00--config\x00/tmp/a b.ovpn\x00"))
	want := []string{"openvpn", "--config", "/tmp/a b.ovpn"}
	if !slices.Equal(got, want) {
		t.Errorf("splitCmdline = %v, want %v", got, want)
	}
	if got := splitCmdline(nil); got != nil {
		t.Errorf("splitCmdline(nil) = %v, want nil", got)
	}
}

// Stop must be a no-op on a process that already exited: after watch() reaps
// the child its pid can be recycled, and we signal a whole process group.
func TestStopAfterExitDoesNotSignal(t *testing.T) {
	cfg := &config.Config{
		OpenVPNCommand:            "true", // exits immediately
		TunnelInterface:           naming.ActiveDevice(),
		DataDir:                   t.TempDir(),
		OpenVPNConnectTimeoutSecs: 2,
	}
	m := NewManager(cfg)
	_ = m.Connect(context.Background(), "n1", "remote 1.2.3.4 1194\n")

	// Whether or not a Managed survived the failed start, Disconnect must
	// return promptly and without panicking on an already-reaped process.
	done := make(chan struct{})
	go func() { m.Disconnect(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Disconnect blocked on an already-exited process")
	}
}

func TestBuildArgsVersionBranch(t *testing.T) {
	base := BuildParams{Executable: []string{"openvpn"}, ConfigFile: "/c.ovpn", AuthFile: "/a.txt", Device: naming.ActiveDevice()}

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
		TunnelInterface:           naming.ActiveDevice(),
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
