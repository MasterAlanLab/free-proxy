package vpngate

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/masteralanlab/free-proxy/internal/config"
	"github.com/masteralanlab/free-proxy/internal/domain"
	"github.com/masteralanlab/free-proxy/internal/netx"
)

// Provider fetches and parses nodes from VPNGate, with a TLS/HTTP + upstream
// fallback chain matching the former Python client.
type Provider struct {
	apiURL         string
	limit          int
	timeout        time.Duration
	upstreamURL    string
	directFallback bool

	httpClient *http.Client // optional injected client for tests
	now        func() time.Time

	LastStats ParseStats
}

// NewProvider builds a Provider from config.
func NewProvider(cfg *config.Config) *Provider {
	return &Provider{
		apiURL:         cfg.VPNGateAPIURL,
		limit:          cfg.DiscoveryLimit,
		timeout:        cfg.RequestTimeout(),
		upstreamURL:    cfg.UpstreamProxyURL,
		directFallback: cfg.UpstreamDirectFallback,
		now:            time.Now,
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

// Discover fetches the node list, trying HTTPS, HTTPS-no-verify, and HTTP, each
// optionally through the configured upstream proxy then directly.
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
	upstream := netx.GetUpstreamProxy(p.upstreamURL,
		os.Getenv("http_proxy"), os.Getenv("HTTP_PROXY"),
		os.Getenv("https_proxy"), os.Getenv("HTTPS_PROXY"))

	var lastErr error
	attempt := func(proxyURL string, list []target) ([]domain.DiscoveredNode, bool) {
		for _, t := range list {
			client, err := buildClient(proxyURL, t.verify, p.timeout)
			if err != nil {
				lastErr = err
				continue
			}
			nodes, err := p.fetch(ctx, client, t.url)
			if err != nil {
				lastErr = err
				continue
			}
			return nodes, true
		}
		return nil, false
	}

	proxyStr := ""
	if upstream != nil {
		proxyStr = upstream.URL()
	}
	if nodes, ok := attempt(proxyStr, targets); ok {
		return nodes, nil
	}
	if upstream == nil {
		if nodes, ok := attempt("", targets[:1]); ok {
			return nodes, nil
		}
	} else if p.directFallback {
		if nodes, ok := attempt("", targets); ok {
			return nodes, nil
		}
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

func buildClient(proxyURL string, verify bool, timeout time.Duration) (*http.Client, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !verify}, //nolint:gosec // intentional TLS fallback
	}
	if proxyURL != "" {
		pu, err := url.Parse(proxyURL)
		if err != nil {
			return nil, err
		}
		tr.Proxy = http.ProxyURL(pu)
	}
	return &http.Client{Timeout: timeout, Transport: tr}, nil
}
