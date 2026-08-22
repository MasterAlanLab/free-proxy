package netx

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsPublicIP(t *testing.T) {
	cases := []struct {
		ip   string
		want bool
	}{
		{"1.2.3.4", true},
		{"203.0.113.9", true},
		{"2001:db8::1", true},
		{"10.0.0.5", false},
		{"172.16.4.1", false},
		{"192.168.1.10", false},
		{"127.0.0.1", false},
		{"169.254.10.1", false},
		{"100.64.0.1", false},    // carrier-grade NAT
		{"100.127.255.1", false}, // carrier-grade NAT, top of the range
		{"100.128.0.1", true},    // just outside 100.64.0.0/10
		{"fe80::1", false},
		{"::1", false},
	}
	for _, c := range cases {
		if got := isPublicIP(net.ParseIP(c.ip)); got != c.want {
			t.Errorf("isPublicIP(%s) = %v, want %v", c.ip, got, c.want)
		}
	}
	if isPublicIP(nil) {
		t.Error("isPublicIP(nil) = true, want false")
	}
}

// A concrete bind address is the operator's own choice and must survive
// untouched — no lookup, no substitution.
func TestResolveAdvertiseHostKeepsConcreteBind(t *testing.T) {
	cases := []struct {
		bind       string
		wantHost   string
		wantPublic bool
	}{
		{"203.0.113.7", "203.0.113.7", true},
		{" 203.0.113.7 ", "203.0.113.7", true},
		{"192.168.1.20", "192.168.1.20", false},
		{"[2001:db8::5]", "2001:db8::5", true},
		{"panel.example.com", "panel.example.com", true},
	}
	for _, c := range cases {
		got := ResolveAdvertiseHost(context.Background(), c.bind)
		if got.Host != c.wantHost || got.Public != c.wantPublic {
			t.Errorf("ResolveAdvertiseHost(%q) = %+v, want {%s %v}", c.bind, got, c.wantHost, c.wantPublic)
		}
	}
}

func TestLookupPublicIP(t *testing.T) {
	private := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// A NAT'd echo answer we must reject rather than print.
		w.Write([]byte("10.1.2.3\n"))
	}))
	defer private.Close()
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusBadGateway)
	}))
	defer broken.Close()
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(" 203.0.113.42\n"))
	}))
	defer good.Close()

	got := lookupPublicIP(context.Background(), http.DefaultClient, []string{broken.URL, private.URL, good.URL})
	if got != "203.0.113.42" {
		t.Errorf("lookupPublicIP = %q, want 203.0.113.42", got)
	}
	if got := lookupPublicIP(context.Background(), http.DefaultClient, []string{broken.URL}); got != "" {
		t.Errorf("lookupPublicIP with no working endpoint = %q, want empty", got)
	}
}

// An HTML error page must never be mistaken for an address.
func TestFetchIPRejectsJunkBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("<html><body>captive portal</body></html>"))
	}))
	defer srv.Close()
	if got := fetchIP(context.Background(), http.DefaultClient, srv.URL); got != "" {
		t.Errorf("fetchIP = %q, want empty", got)
	}
}

func TestIsTunnelInterface(t *testing.T) {
	for _, name := range []string{"fpx0", "tun0", "tap3", "wg0", "ppp0"} {
		if !isTunnelInterface(name) {
			t.Errorf("isTunnelInterface(%q) = false, want true", name)
		}
	}
	for _, name := range []string{"eth0", "ens5", "enp3s0", "wlan0"} {
		if isTunnelInterface(name) {
			t.Errorf("isTunnelInterface(%q) = true, want false", name)
		}
	}
}
