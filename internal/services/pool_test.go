package services

import (
	"testing"

	"github.com/masteralanlab/free-proxy/internal/domain"
)

func node(id string, latency, score int, ip domain.IpType, country string) domain.ProxyNodeRead {
	return domain.ProxyNodeRead{ID: id, LatencyMS: latency, SourceScore: score, IPType: ip, Country: country}
}

func TestApplyFiltersIPType(t *testing.T) {
	nodes := []domain.ProxyNodeRead{
		node("a", 10, 1, domain.IpResidential, "Japan"),
		node("b", 20, 1, domain.IpHosting, "Japan"),
		node("c", 30, 1, domain.IpMobile, "Japan"),
	}
	res := ApplyFilters(nodes, domain.ProxySettings{RoutingMode: domain.PolicyAuto, RoutingIPType: domain.RoutingResidential}, false)
	if len(res) != 2 {
		t.Fatalf("residential filter kept %d, want 2 (res+mobile)", len(res))
	}
	host := ApplyFilters(nodes, domain.ProxySettings{RoutingMode: domain.PolicyAuto, RoutingIPType: domain.RoutingHosting}, false)
	if len(host) != 1 || host[0].ID != "b" {
		t.Fatalf("hosting filter = %+v", host)
	}
}

func TestApplyFiltersCountryAndFixed(t *testing.T) {
	nodes := []domain.ProxyNodeRead{
		node("a", 10, 1, domain.IpHosting, "Japan"),
		node("b", 20, 1, domain.IpHosting, "United States"),
	}
	// Country mode normalizes English->Chinese so "United States" matches "美国".
	us := ApplyFilters(nodes, domain.ProxySettings{RoutingMode: domain.PolicyCountry, ForceCountry: "美国", RoutingIPType: domain.RoutingAll}, false)
	if len(us) != 1 || us[0].ID != "b" {
		t.Fatalf("country filter = %+v", us)
	}
	fixed := "a"
	fx := ApplyFilters(nodes, domain.ProxySettings{RoutingMode: domain.PolicyFixed, FixedNodeID: &fixed, RoutingIPType: domain.RoutingAll}, false)
	if len(fx) != 1 || fx[0].ID != "a" {
		t.Fatalf("fixed filter = %+v", fx)
	}
}

func TestSortCandidatesAuto(t *testing.T) {
	nodes := []domain.ProxyNodeRead{
		node("slow", 100, 50, domain.IpResidential, "JP"),
		node("fast", 10, 5, domain.IpHosting, "JP"),
		node("mid", 50, 90, domain.IpResidential, "JP"),
	}
	SortCandidates(nodes, domain.ProxySettings{RoutingMode: domain.PolicyAuto})
	if nodes[0].ID != "fast" {
		t.Fatalf("auto should pick lowest latency first, got %s", nodes[0].ID)
	}
}

func TestSortCandidatesResidentialFirst(t *testing.T) {
	nodes := []domain.ProxyNodeRead{
		node("hostfast", 5, 5, domain.IpHosting, "JP"),
		node("resslow", 80, 5, domain.IpResidential, "JP"),
	}
	SortCandidates(nodes, domain.ProxySettings{RoutingMode: domain.PolicyResidentialFirst})
	if nodes[0].ID != "resslow" {
		t.Fatalf("residential_first should rank residential above faster hosting, got %s", nodes[0].ID)
	}
}

func TestSortCandidatesSpeedFirst(t *testing.T) {
	nodes := []domain.ProxyNodeRead{
		{ID: "slow", LatencyMS: 10, SourceSpeedBPS: 1000},
		{ID: "fast", LatencyMS: 100, SourceSpeedBPS: 9000},
	}
	SortCandidates(nodes, domain.ProxySettings{RoutingMode: domain.PolicySpeedFirst})
	if nodes[0].ID != "fast" {
		t.Fatalf("speed_first should pick highest advertised speed first, got %s", nodes[0].ID)
	}
}

func TestSortCandidatesSmart(t *testing.T) {
	nodes := []domain.ProxyNodeRead{
		{ID: "balanced", LatencyMS: 20, SourceSpeedBPS: 8000, SourceSessions: 2},
		{ID: "busy", LatencyMS: 10, SourceSpeedBPS: 9000, SourceSessions: 80},
		{ID: "slow", LatencyMS: 100, SourceSpeedBPS: 1000, SourceSessions: 1},
	}
	SortCandidates(nodes, domain.ProxySettings{RoutingMode: domain.PolicySmart})
	if nodes[0].ID != "balanced" {
		t.Fatalf("smart policy should balance latency, speed, and sessions, got %s", nodes[0].ID)
	}
}
