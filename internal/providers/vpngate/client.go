package vpngate

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/masteralanlab/free-proxy/internal/config"
	"github.com/masteralanlab/free-proxy/internal/domain"
)

// Provider fetches and parses nodes from VPNGate with a direct TLS/HTTP fallback chain.
type Provider struct {
	apiURL  string
	limit   int
	timeout time.Duration

	httpClient *http.Client // optional injected client for tests
	now        func() time.Time

	LastStats ParseStats
}

// NewProvider builds a Provider from config.
func NewProvider(cfg *config.Config) *Provider {
	return &Provider{
		apiURL:  cfg.VPNGateAPIURL,
		limit:   cfg.DiscoveryLimit,
		timeout: cfg.RequestTimeout(),
		now:     time.Now,
	}
}

// WithHTTPClient injects a client (tests); it bypasses the fallback chain.
func (p *Provider) WithHTTPClient(c *http.Client) *Provider {
	p.httpClient = c
	return p
}

// Name identifies the provider.
func (p *Provider) Name() string { return "vpngate" }

// ParseStats exposes the last parse counters (total, valid, dup, malformed, missing).
func (p *Provider) ParseStats() (int, int, int, int, int) {
	s := p.LastStats
	return s.TotalRows, s.ValidRows, s.DuplicateRows, s.MalformedRows, s.MissingFieldRows
}

type target struct {
	url    string
	verify bool
}

// Discover fetches the node list directly, trying HTTPS, HTTPS-no-verify, and HTTP.
func (p *Provider) Discover(ctx context.Context) ([]domain.DiscoveredNode, error) {
	if p.httpClient != nil {
		return p.fetch(ctx, p.httpClient, p.apiURL)
	}

	targets := []target{{p.apiURL, true}}
	if strings.HasPrefix(p.apiURL, "https://") {
		targets = append(targets,
			target{p.apiURL, false},
			target{strings.Replace(p.apiURL, "https://", "http://", 1), true},
		)
	}
	var lastErr error
	for _, t := range targets {
		client := buildClient(t.verify, p.timeout)
		nodes, err := p.fetch(ctx, client, t.url)
		if err != nil {
			lastErr = err
			continue
		}
		return nodes, nil
	}
	if lastErr == nil {
		lastErr = errors.New("no reachable endpoint")
	}
	return nil, fmt.Errorf("unable to fetch VPNGate nodes: %w", lastErr)
}

func (p *Provider) fetch(ctx context.Context, client *http.Client, endpoint string) ([]domain.DiscoveredNode, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "free-proxy/0.1")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("VPNGate returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	res, err := ParseResponse(string(body), p.limit, p.now())
	if err != nil {
		return nil, err
	}
	p.LastStats = res.Stats
	if res.Nodes == nil {
		res.Nodes = []domain.DiscoveredNode{}
	}
	return res.Nodes, nil
}

func buildClient(verify bool, timeout time.Duration) *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !verify}, //nolint:gosec // intentional TLS fallback
	}
	return &http.Client{Timeout: timeout, Transport: tr}
}
