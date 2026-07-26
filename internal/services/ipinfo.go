package services

import (
	"context"
	"errors"
	"time"

	"github.com/masteralanlab/free-proxy/internal/config"
	"github.com/masteralanlab/free-proxy/internal/domain"
	"github.com/masteralanlab/free-proxy/internal/ipinfo"
	"github.com/masteralanlab/free-proxy/internal/store"
)

// IpInfoService enriches nodes with IP classification, using the cache first.
type IpInfoService struct {
	cfg    *config.Config
	client *ipinfo.Client
	nodes  *store.NodeRepository
	cache  *store.IPCacheRepository
}

// NewIpInfoService constructs an IpInfoService.
func NewIpInfoService(cfg *config.Config, client *ipinfo.Client, nodes *store.NodeRepository, cache *store.IPCacheRepository) *IpInfoService {
	return &IpInfoService{cfg: cfg, client: client, nodes: nodes, cache: cache}
}

// Enrich classifies a single node's IP.
func (s *IpInfoService) Enrich(ctx context.Context, nodeID, ip string) error {
	return s.EnrichMany(ctx, map[string]string{nodeID: ip})
}

// EnrichMany classifies several nodes, refreshing only stale/absent cache entries.
func (s *IpInfoService) EnrichMany(ctx context.Context, nodes map[string]string) error {
	now := time.Now().UTC()
	cutoff := now.Add(-s.cfg.IPInfoCacheTTL())
	stale := map[string]string{}
	cached := map[string]domain.IpInfo{}
	cachedAt := map[string]time.Time{}

	for nodeID, ip := range nodes {
		info, updatedAt, err := s.cache.Get(ctx, ip)
		if err == nil {
			cached[ip] = info
			cachedAt[ip] = updatedAt
			if updatedAt.After(cutoff) {
				_ = s.nodes.UpdateIPInfo(ctx, nodeID, info, updatedAt)
				continue
			}
		} else if !errors.Is(err, store.ErrNotFound) {
			return err
		}
		stale[nodeID] = ip
	}
	if len(stale) == 0 {
		return nil
	}

	seen := map[string]bool{}
	var uniqueIPs []string
	for _, ip := range stale {
		if !seen[ip] {
			seen[ip] = true
			uniqueIPs = append(uniqueIPs, ip)
		}
	}
	results, err := s.client.LookupMany(ctx, uniqueIPs)
	if err != nil {
		return err
	}
	for nodeID, ip := range stale {
		if info, ok := results[ip]; ok {
			_ = s.cache.Upsert(ctx, info, now)
			_ = s.nodes.UpdateIPInfo(ctx, nodeID, info, now)
		} else if info, ok := cached[ip]; ok {
			_ = s.nodes.UpdateIPInfo(ctx, nodeID, info, cachedAt[ip])
		}
	}
	return nil
}
