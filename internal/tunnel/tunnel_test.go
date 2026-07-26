package tunnel

import (
	"slices"
	"testing"

	"github.com/masteralanlab/free-proxy/internal/domain"
	"github.com/masteralanlab/free-proxy/internal/netx"
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

func TestBuildArgsUpstream(t *testing.T) {
	p := BuildParams{
		Executable: []string{"openvpn"}, ConfigFile: "/c", AuthFile: "/a", Device: "tun0",
		Version:  Version{2, 6},
		Upstream: &netx.UpstreamProxy{Kind: "socks", Host: "127.0.0.1", Port: 1080},
	}
	args := BuildArgs(p)
	if !slices.Contains(args, "--socks-proxy") {
		t.Errorf("expected --socks-proxy, got %v", args)
	}
}

func TestIsTCPConfig(t *testing.T) {
	if !IsTCPConfig("proto tcp\nremote 1.2.3.4 443\n") {
		t.Error("proto tcp should be TCP")
	}
	if IsTCPConfig("proto udp\nremote 1.2.3.4 1194\n") {
		t.Error("proto udp should not be TCP")
	}
	if !IsTCPConfig("remote 1.2.3.4 443 tcp\n") {
		t.Error("remote ... tcp should be TCP")
	}
}
