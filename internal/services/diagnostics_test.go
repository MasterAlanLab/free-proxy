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
