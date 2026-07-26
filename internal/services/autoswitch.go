package services

import (
	"context"

	"github.com/masteralanlab/free-proxy/internal/config"
	"github.com/masteralanlab/free-proxy/internal/domain"
	"github.com/masteralanlab/free-proxy/internal/store"
)

// AutoSwitchService picks a replacement exit after a failure, blacklisting nodes
// that fail to activate.
type AutoSwitchService struct {
	cfg          *config.Config
	nodes        *store.NodeRepository
	settingsRepo *store.SettingsRepository
	pool         *ProxyPoolService
	gateway      *GatewayService
}

// NewAutoSwitchService constructs an AutoSwitchService.
func NewAutoSwitchService(cfg *config.Config, nodes *store.NodeRepository, settingsRepo *store.SettingsRepository, pool *ProxyPoolService, gateway *GatewayService) *AutoSwitchService {
	return &AutoSwitchService{cfg: cfg, nodes: nodes, settingsRepo: settingsRepo, pool: pool, gateway: gateway}
}

// Switch activates the best alternate node, or the fixed node in fixed mode.
func (s *AutoSwitchService) Switch(ctx context.Context) (*domain.TunnelStartResult, error) {
	settings, err := s.settingsRepo.Get(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.ConnectionEnabled {
		return nil, nil
	}
	if settings.RoutingMode == domain.PolicyFixed {
		if settings.FixedNodeID == nil || *settings.FixedNodeID == "" {
			return nil, nil
		}
		res, err := s.gateway.Activate(ctx, *settings.FixedNodeID)
		return &res, err
	}

	excluded := map[string]bool{}
	if active := s.gateway.Status().ActiveNodeID; active != nil {
		excluded[*active] = true
	}
	for i := 0; i < 3; i++ {
		candidate, err := s.selectExcluding(ctx, excluded)
		if err != nil {
			return nil, err
		}
		if candidate == nil {
			s.gateway.DisconnectOnly(ctx)
			return nil, nil
		}
		res, err := s.gateway.Activate(ctx, candidate.ID)
		if err == nil && res.Success {
			return &res, nil
		}
		excluded[candidate.ID] = true
		msg := "activation failed"
		if res.Message != "" {
			msg = res.Message
		} else if err != nil {
			msg = err.Error()
		}
		_ = s.nodes.Blacklist(ctx, candidate.ID, msg, s.cfg.InvalidBackoff())
	}
	return nil, nil
}

// HandleUnexpectedExit reconnects after an unexpected tunnel exit.
func (s *AutoSwitchService) HandleUnexpectedExit(ctx context.Context) {
	settings, err := s.settingsRepo.Get(ctx)
	if err != nil || !settings.ConnectionEnabled {
		return
	}
	if settings.RoutingMode == domain.PolicyFixed && settings.FixedNodeID != nil && *settings.FixedNodeID != "" {
		_, _ = s.gateway.Activate(ctx, *settings.FixedNodeID)
		return
	}
	_, _ = s.Switch(ctx)
}

func (s *AutoSwitchService) selectExcluding(ctx context.Context, excluded map[string]bool) (*domain.ProxyNodeRead, error) {
	candidates, err := s.nodes.ListNodes(ctx, store.NodeFilter{Status: string(domain.NodeReady), CurrentOnly: true}, 1000, 0)
	if err != nil {
		return nil, err
	}
	settings, err := s.settingsRepo.Get(ctx)
	if err != nil {
		return nil, err
	}
	candidates = ApplyFilters(candidates, settings, false)
	filtered := candidates[:0]
	for _, n := range candidates {
		if !excluded[n.ID] {
			filtered = append(filtered, n)
		}
	}
	SortCandidates(filtered, settings)
	if len(filtered) == 0 {
		return nil, nil
	}
	best := filtered[0]
	return &best, nil
}
