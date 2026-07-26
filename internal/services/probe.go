package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/masteralanlab/free-proxy/internal/config"
	"github.com/masteralanlab/free-proxy/internal/domain"
	"github.com/masteralanlab/free-proxy/internal/netx"
	"github.com/masteralanlab/free-proxy/internal/store"
	"github.com/masteralanlab/free-proxy/internal/tunnel"
)

// ProbeService dials nodes to test connectivity and measure latency.
type ProbeService struct {
	cfg         *config.Config
	nodes       *store.NodeRepository
	tunnel      *tunnel.Manager
	tunAlloc    *netx.TunAllocator
	runner      netx.CommandRunner
	ipInfo      *IpInfoService
	history     *store.ProbeResultRepository
	coordinator *Coordinator
	sem         chan struct{}
}

// NewProbeService constructs a ProbeService.
func NewProbeService(cfg *config.Config, nodes *store.NodeRepository, mgr *tunnel.Manager, tunAlloc *netx.TunAllocator,
	runner netx.CommandRunner, ipInfo *IpInfoService, history *store.ProbeResultRepository, coordinator *Coordinator) *ProbeService {
	n := cfg.MaxProbeConcurrency
	if n < 1 {
		n = 1
	}
	return &ProbeService{
		cfg: cfg, nodes: nodes, tunnel: mgr, tunAlloc: tunAlloc, runner: runner,
		ipInfo: ipInfo, history: history, coordinator: coordinator, sem: make(chan struct{}, n),
	}
}

// Probe tests a single node, updating its state and (optionally) enriching IP info.
func (s *ProbeService) Probe(ctx context.Context, nodeID string, enrich bool) (domain.ProbeResult, error) {
	target, err := s.nodes.GetTarget(ctx, nodeID)
	if err != nil {
		return domain.ProbeResult{}, err
	}
	_ = s.nodes.MarkProbing(ctx, nodeID)

	var latency int
	var tun domain.TunnelStartResult
	func() {
		s.sem <- struct{}{}
		defer func() { <-s.sem }()
		device, release, allocErr := s.tunAlloc.Allocate()
		if allocErr != nil {
			tun = failureResult(allocErr)
			return
		}
		defer release()
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			latency = netx.MeasureNodeLatency(ctx, s.runner, target.RemoteHost, target.RemotePort, target.SourcePingMS)
		}()
		go func() {
			defer wg.Done()
			tun = s.tunnel.Probe(ctx, target.ConfigText, device)
		}()
		wg.Wait()
	}()

	probedAt := time.Now().UTC()
	result := domain.ProbeResult{
		NodeID: nodeID, Available: tun.Success, LatencyMS: latency, Tunnel: tun, ProbedAt: probedAt,
	}
	_ = s.nodes.UpdateProbeResult(ctx, nodeID, result.Available, result.LatencyMS, probedAt)
	if s.history != nil {
		_, _ = s.history.Insert(ctx, result)
	}
	if enrich && result.Available && s.ipInfo != nil {
		_ = s.ipInfo.Enrich(ctx, nodeID, target.IPAddress)
	}
	return result, nil
}

// ProbeMany tests several nodes concurrently (bounded by the semaphore).
func (s *ProbeService) ProbeMany(ctx context.Context, nodeIDs []string) ([]domain.ProbeResult, error) {
	var results []domain.ProbeResult
	err := s.coordinator.Run(ctx, "probe", false, func(ctx context.Context) error {
		results = s.probeMany(ctx, nodeIDs)
		return nil
	})
	return results, err
}

func (s *ProbeService) probeMany(ctx context.Context, nodeIDs []string) []domain.ProbeResult {
	seen := map[string]bool{}
	var unique []string
	for _, id := range nodeIDs {
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	results := make([]domain.ProbeResult, len(unique))
	var wg sync.WaitGroup
	for i, id := range unique {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			res, err := s.Probe(ctx, id, false)
			if err != nil {
				res = domain.ProbeResult{NodeID: id, Available: false, Tunnel: failureResult(err), ProbedAt: time.Now().UTC()}
				_ = s.nodes.UpdateProbeResult(ctx, id, false, 0, res.ProbedAt)
			}
			results[i] = res
		}(i, id)
	}
	wg.Wait()

	if s.ipInfo != nil {
		successful := map[string]string{}
		for _, r := range results {
			if !r.Available {
				continue
			}
			if target, err := s.nodes.GetTarget(ctx, r.NodeID); err == nil {
				successful[r.NodeID] = target.IPAddress
			}
		}
		if len(successful) > 0 {
			_ = s.ipInfo.EnrichMany(ctx, successful)
		}
	}
	return results
}

// ProbeJob is the JobFunc form of Probe.
func (s *ProbeService) ProbeJob(nodeID string) JobFunc {
	return func(ctx context.Context) (map[string]any, error) {
		res, err := s.Probe(ctx, nodeID, true)
		if err != nil {
			return nil, err
		}
		return toMap(res)
	}
}

// ProbeManyJob is the JobFunc form of ProbeMany.
func (s *ProbeService) ProbeManyJob(nodeIDs []string) JobFunc {
	return func(ctx context.Context) (map[string]any, error) {
		res, err := s.ProbeMany(ctx, nodeIDs)
		if err != nil {
			return nil, err
		}
		items := make([]map[string]any, 0, len(res))
		for _, r := range res {
			m, err := toMap(r)
			if err != nil {
				return nil, err
			}
			items = append(items, m)
		}
		return map[string]any{"nodes": items}, nil
	}
}

func failureResult(err error) domain.TunnelStartResult {
	code := domain.FailUnknown
	return domain.TunnelStartResult{
		Success:        false,
		Status:         domain.TunnelFailed,
		Message:        fmt.Sprintf("Probe failed: %v", err),
		FailureCode:    &code,
		HandshakeStage: "starting",
	}
}
