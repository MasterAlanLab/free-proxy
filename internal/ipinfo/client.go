// Package ipinfo classifies IP addresses (residential/mobile/hosting) via the
// ip-api.com batch endpoint.
package ipinfo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/masteralanlab/free-proxy/internal/domain"
)

// Client looks up IP classification in batches.
type Client struct {
	apiURL string
	http   *http.Client
}

// New creates a Client.
func New(apiURL string, timeout time.Duration) *Client {
	return &Client{apiURL: apiURL, http: &http.Client{
		Timeout:   timeout,
		Transport: &http.Transport{},
	}}
}

type apiItem struct {
	Status     string `json:"status"`
	Query      string `json:"query"`
	Country    string `json:"country"`
	RegionName string `json:"regionName"`
	City       string `json:"city"`
	ISP        string `json:"isp"`
	Org        string `json:"org"`
	AS         string `json:"as"`
	ASName     string `json:"asname"`
	Proxy      bool   `json:"proxy"`
	Hosting    bool   `json:"hosting"`
	Mobile     bool   `json:"mobile"`
}

// LookupMany classifies the given IPs, returning a map keyed by IP.
func (c *Client) LookupMany(ctx context.Context, ips []string) (map[string]domain.IpInfo, error) {
	out := map[string]domain.IpInfo{}
	for i := 0; i < len(ips); i += 100 {
		end := min(i+100, len(ips))
		batch := ips[i:end]
		body, err := json.Marshal(batch)
		if err != nil {
			return out, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL, bytes.NewReader(body))
		if err != nil {
			return out, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "free-proxy/0.1")
		resp, err := c.http.Do(req)
		if err != nil {
			return out, err
		}
		func() {
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				err = fmt.Errorf("ip-api returned status %d", resp.StatusCode)
				return
			}
			var items []apiItem
			if decErr := json.NewDecoder(resp.Body).Decode(&items); decErr != nil {
				err = decErr
				return
			}
			for _, it := range items {
				if info, ok := parseItem(it); ok {
					out[info.IPAddress] = info
				}
			}
		}()
		if err != nil {
			return out, err
		}
	}
	return out, nil
}

func parseItem(it apiItem) (domain.IpInfo, bool) {
	if it.Status != "success" || it.Query == "" {
		return domain.IpInfo{}, false
	}
	ipType := domain.IpResidential
	switch {
	case it.Mobile:
		ipType = domain.IpMobile
	case it.Hosting || it.Proxy:
		ipType = domain.IpHosting
	}
	quality := "normal"
	switch {
	case it.Proxy:
		quality = "proxy"
	case it.Hosting:
		quality = "datacenter"
	case it.Mobile:
		quality = "mobile"
	}
	var locParts []string
	for _, p := range []string{it.Country, it.RegionName, it.City} {
		if p != "" {
			locParts = append(locParts, p)
		}
	}
	owner := it.Org
	if owner == "" {
		owner = it.ISP
	}
	return domain.IpInfo{
		IPAddress: it.Query,
		Owner:     owner,
		ASN:       it.AS,
		ASName:    it.ASName,
		Location:  strings.Join(locParts, " "),
		IPType:    ipType,
		Quality:   quality,
	}, true
}
