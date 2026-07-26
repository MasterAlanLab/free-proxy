package services

import (
	"context"
	"errors"

	"github.com/masteralanlab/free-proxy/internal/domain"
	"github.com/masteralanlab/free-proxy/internal/store"
)

// SettingsService applies routing-settings changes and keeps the active exit
// consistent with them.
type SettingsService struct {
	nodes        *store.NodeRepository
	settingsRepo *store.SettingsRepository
	pool         *ProxyPoolService
	gateway      *GatewayService
	autoSwitch   *AutoSwitchService
}

// NewSettingsService constructs a SettingsService.
func NewSettingsService(nodes *store.NodeRepository, settingsRepo *store.SettingsRepository, pool *ProxyPoolService, gateway *GatewayService, autoSwitch *AutoSwitchService) *SettingsService {
	return &SettingsService{nodes: nodes, settingsRepo: settingsRepo, pool: pool, gateway: gateway, autoSwitch: autoSwitch}
}

// Get returns the current settings.
func (s *SettingsService) Get(ctx context.Context) (domain.ProxySettings, error) {
	return s.settingsRepo.Get(ctx)
}

// Update applies a settings change and reconciles the active exit.
func (s *SettingsService) Update(ctx context.Context, payload domain.ProxySettingsUpdate) (domain.ProxySettings, error) {
	if err := s.settingsRepo.Update(ctx, payload); err != nil {
		return domain.ProxySettings{}, err
	}
	updated, err := s.settingsRepo.Get(ctx)
	if err != nil {
		return domain.ProxySettings{}, err
	}
	s.gateway.SetConnectionEnabled(updated.ConnectionEnabled)
	if !updated.ConnectionEnabled {
		s.gateway.DisconnectOnly(ctx)
		return updated, nil
	}
	if err := s.enforceActiveNode(ctx, updated); err != nil {
		return updated, err
	}
	if s.gateway.Status().ActiveNodeID == nil {
		if updated.RoutingMode == domain.PolicyFixed && updated.FixedNodeID != nil && *updated.FixedNodeID != "" {
			_, _ = s.gateway.Activate(ctx, *updated.FixedNodeID)
		} else {
			_, _ = s.autoSwitch.Switch(ctx)
		}
	}
	return updated, nil
}

// ToggleFavorite flips a node's favorite membership.
func (s *SettingsService) ToggleFavorite(ctx context.Context, nodeID string) (domain.ProxySettings, error) {
	if _, err := s.nodes.Get(ctx, nodeID); err != nil {
		return domain.ProxySettings{}, err
	}
	if _, err := s.settingsRepo.ToggleFavorite(ctx, nodeID); err != nil {
		return domain.ProxySettings{}, err
	}
	updated, err := s.settingsRepo.Get(ctx)
	if err != nil {
		return domain.ProxySettings{}, err
	}
	if updated.RoutingMode == domain.PolicyFavorites {
		if err := s.enforceActiveNode(ctx, updated); err != nil {
			return updated, err
		}
	}
	return updated, nil
}

func (s *SettingsService) enforceActiveNode(ctx context.Context, settings domain.ProxySettings) error {
	active := s.gateway.Status().ActiveNodeID
	if active == nil {
		return nil
	}
	node, err := s.nodes.Get(ctx, *active)
	allowed := err == nil && len(ApplyFilters([]domain.ProxyNodeRead{node}, settings, false)) > 0
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return err
	}
	if allowed {
		return nil
	}
	s.gateway.DisconnectOnly(ctx)
	if settings.ConnectionEnabled && settings.RoutingMode != domain.PolicyFixed {
		_, _ = s.autoSwitch.Switch(ctx)
	}
	return nil
}
