package services

import (
	"errors"
	"testing"
)

func TestParseDefaultInterface(t *testing.T) {
	out := "default via 10.0.0.1 dev eth0 proto dhcp metric 100"
	if got := ParseDefaultInterface(out); got != "eth0" {
		t.Fatalf("iface = %q, want eth0", got)
	}
	if got := ParseDefaultInterface("no route here"); got != "" {
		t.Fatalf("iface = %q, want empty", got)
	}
}

func TestIsDNSError(t *testing.T) {
	if !IsDNSError(errors.New("lookup vpngate.net: no such host")) {
		t.Error("expected DNS error")
	}
	if !IsDNSError(errors.New("Temporary failure in name resolution")) {
		t.Error("expected DNS error")
	}
	if IsDNSError(errors.New("connection refused")) {
		t.Error("refused is not a DNS error")
	}
	if IsDNSError(nil) {
		t.Error("nil is not a DNS error")
	}
}

func TestProviderHost(t *testing.T) {
	cases := map[string]string{
		"https://www.vpngate.net/api/iphone/": "www.vpngate.net",
		"http://ip-api.com/batch?x=1":         "ip-api.com",
		"https://host:8443/path":              "host",
	}
	for in, want := range cases {
		if got := providerHost(in); got != want {
			t.Errorf("providerHost(%q) = %q, want %q", in, got, want)
		}
	}
}

// The rp_filter check must read the effective value, max(conf.all, conf.<dev>),
// not conf.all alone. Once the router stopped forcing conf.all=2 (so it would
// stop weakening every other interface on the host), an all-alone check turned
// every conf.all=1 host — the default on RHEL-family distributions — into a
// permanent, unclearable FAIL despite the tunnel working correctly.
func TestRPFilterUsesEffectiveValue(t *testing.T) {
	cases := []struct {
		name     string
		all, dev int
		want     bool
	}{
		{"strict host, our device loose", 1, 2, true}, // the regression
		{"disabled host, our device loose", 0, 2, true},
		{"loose host, our device loose", 2, 2, true},
		{"both disabled", 0, 0, true},
		{"strict everywhere", 1, 1, false},
		{"strict host, device untouched", 1, 0, false},
		{"device strict, host disabled", 0, 1, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := rpFilterOK(c.all, c.dev); got != c.want {
				t.Errorf("rpFilterOK(all=%d, dev=%d) = %v, want %v", c.all, c.dev, got, c.want)
			}
		})
	}
}
