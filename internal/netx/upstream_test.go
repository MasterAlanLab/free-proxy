package netx

import "testing"

func TestParseUpstreamProxy(t *testing.T) {
	t.Run("socks with auth", func(t *testing.T) {
		p := ParseUpstreamProxy("socks5://user:pass@10.0.0.1:1080", "")
		if p == nil {
			t.Fatal("nil")
		}
		if p.Kind != "socks" || p.Host != "10.0.0.1" || p.Port != 1080 {
			t.Fatalf("got %+v", p)
		}
		if !p.HasAuth || p.Username != "user" || p.Password != "pass" {
			t.Fatalf("auth = %+v", p)
		}
	})

	t.Run("bare host defaults to http", func(t *testing.T) {
		p := ParseUpstreamProxy("proxy.local:3128", "")
		if p == nil || p.Kind != "http" || p.Host != "proxy.local" || p.Port != 3128 {
			t.Fatalf("got %+v", p)
		}
	})

	t.Run("http default port", func(t *testing.T) {
		p := ParseUpstreamProxy("http://h", "")
		if p == nil || p.Port != 8080 {
			t.Fatalf("got %+v", p)
		}
	})

	t.Run("socks default port", func(t *testing.T) {
		p := ParseUpstreamProxy("socks5://h", "")
		if p == nil || p.Port != 1080 {
			t.Fatalf("got %+v", p)
		}
	})

	t.Run("empty", func(t *testing.T) {
		if ParseUpstreamProxy("   ", "") != nil {
			t.Fatal("expected nil for empty")
		}
	})
}

func TestUpstreamURL(t *testing.T) {
	p := &UpstreamProxy{Kind: "socks", Host: "1.2.3.4", Port: 1080, Username: "u", Password: "p", HasAuth: true}
	if got := p.URL(); got != "socks5://u:p@1.2.3.4:1080" {
		t.Fatalf("URL = %s", got)
	}
	noauth := &UpstreamProxy{Kind: "http", Host: "1.2.3.4", Port: 8080}
	if got := noauth.URL(); got != "http://1.2.3.4:8080" {
		t.Fatalf("URL = %s", got)
	}
}

func TestTunAllocator(t *testing.T) {
	a, err := NewTunAllocator(2, 3)
	if err != nil {
		t.Fatal(err)
	}
	d1, r1, err := a.Allocate()
	if err != nil || d1 != "tun2" {
		t.Fatalf("first = %s %v", d1, err)
	}
	d2, r2, err := a.Allocate()
	if err != nil || d2 != "tun3" {
		t.Fatalf("second = %s %v", d2, err)
	}
	if _, _, err := a.Allocate(); err == nil {
		t.Fatal("expected exhaustion error")
	}
	r1()
	d3, r3, err := a.Allocate()
	if err != nil || d3 != "tun2" {
		t.Fatalf("after release = %s %v", d3, err)
	}
	r2()
	r3()
}
