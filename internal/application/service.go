package application

import (
	"context"
	"fmt"

	"github.com/marcus/health-factor-monitor/internal/domain"
)

// CheckService orchestrates health factor retrieval across all configured
// positions, tolerating individual provider failures.
type CheckService struct {
	config    *domain.Config
	providers map[string]domain.HealthFactorProvider
}

// NewCheckService builds a CheckService from the validated config and a map of
// providers keyed by "protocol:network" (e.g., "aave:ethereum").
func NewCheckService(config *domain.Config, providers map[string]domain.HealthFactorProvider) *CheckService {
	return &CheckService{config: config, providers: providers}
}

// CheckAll retrieves health factors for all configured positions. It returns
// exactly one ProviderResult per position, never panicking on unmatched
// protocols and never stopping on individual provider errors.
func (s *CheckService) CheckAll(ctx context.Context) []domain.ProviderResult {
	results := make([]domain.ProviderResult, 0, len(s.config.Positions))
	for _, pos := range s.config.Positions {
		results = append(results, s.checkPosition(ctx, pos))
	}
	return results
}

// CheckByProtocol retrieves health factors only for positions matching the
// given protocol identifier. It follows the same error-tolerance rules as
// CheckAll.
func (s *CheckService) CheckByProtocol(ctx context.Context, protocol string) []domain.ProviderResult {
	results := make([]domain.ProviderResult, 0, len(s.config.Positions))
	for _, pos := range s.config.Positions {
		if pos.Protocol != protocol {
			continue
		}
		results = append(results, s.checkPosition(ctx, pos))
	}
	return results
}

func (s *CheckService) checkPosition(ctx context.Context, pos domain.LendingPosition) domain.ProviderResult {
	provider, ok := s.providers[providerKey(pos)]
	if !ok {
		return domain.ProviderResult{
			Position: pos,
			Error:    fmt.Sprintf("no provider registered for %s", providerKey(pos)),
		}
	}

	hf, err := provider.GetHealthFactor(ctx, pos.Wallet.Address)
	if err != nil {
		return domain.ProviderResult{
			Position: pos,
			Error:    err.Error(),
		}
	}

	return domain.ProviderResult{
		Position:     pos,
		HealthFactor: hf,
	}
}

func providerKey(pos domain.LendingPosition) string {
	return pos.Protocol + ":" + pos.Network
}
