package application_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/marcus/health-factor-monitor/internal/application"
	"github.com/marcus/health-factor-monitor/internal/domain"
)

type mockProvider struct {
	protocol string
	network  string
	hf       *domain.HealthFactor
	err      error
}

func (m *mockProvider) Protocol() string { return m.protocol }
func (m *mockProvider) Network() string  { return m.network }

func (m *mockProvider) GetHealthFactor(_ context.Context, _ string) (*domain.HealthFactor, error) {
	return m.hf, m.err
}

var _ domain.HealthFactorProvider = (*mockProvider)(nil)

func newTestConfig(positions ...domain.LendingPosition) *domain.Config {
	return &domain.Config{
		RPCEndpoints: map[string]string{
			domain.NetworkEthereum: "https://eth.example.com",
			domain.NetworkSolana:   "https://sol.example.com",
		},
		Positions: positions,
	}
}

func countResults(results []domain.ProviderResult) (hfCount, errCount int) {
	for _, r := range results {
		if r.HealthFactor != nil {
			hfCount++
		}
		if r.Error != "" {
			errCount++
		}
	}
	return hfCount, errCount
}

func TestCheckAll(t *testing.T) {
	ethPos := domain.LendingPosition{
		Wallet:   domain.Wallet{Address: "0x0000000000000000000000000000000000000001", Alias: "aave-main"},
		Protocol: domain.ProtocolAave,
		Network:  domain.NetworkEthereum,
	}
	solPos := domain.LendingPosition{
		Wallet:   domain.Wallet{Address: "11111111111111111111111111111111", Alias: "kamino-main"},
		Protocol: domain.ProtocolKamino,
		Network:  domain.NetworkSolana,
	}
	safeHF := &domain.HealthFactor{Value: 2.5, Classification: domain.ClassificationSafe}

	tests := []struct {
		name            string
		positions       []domain.LendingPosition
		providers       map[string]domain.HealthFactorProvider
		wantHF          int
		wantErrors      int
		wantErrContains string
	}{
		{
			name:      "all providers succeed",
			positions: []domain.LendingPosition{ethPos, solPos},
			providers: map[string]domain.HealthFactorProvider{
				"aave:ethereum": &mockProvider{protocol: domain.ProtocolAave, network: domain.NetworkEthereum, hf: safeHF},
				"kamino:solana": &mockProvider{protocol: domain.ProtocolKamino, network: domain.NetworkSolana, hf: safeHF},
			},
			wantHF:     2,
			wantErrors: 0,
		},
		{
			name:      "one provider fails",
			positions: []domain.LendingPosition{ethPos, solPos},
			providers: map[string]domain.HealthFactorProvider{
				"aave:ethereum": &mockProvider{protocol: domain.ProtocolAave, network: domain.NetworkEthereum, hf: safeHF},
				"kamino:solana": &mockProvider{protocol: domain.ProtocolKamino, network: domain.NetworkSolana, err: errors.New("connection failed")},
			},
			wantHF:          1,
			wantErrors:      1,
			wantErrContains: "connection failed",
		},
		{
			name:      "all providers fail",
			positions: []domain.LendingPosition{ethPos, solPos},
			providers: map[string]domain.HealthFactorProvider{
				"aave:ethereum": &mockProvider{protocol: domain.ProtocolAave, network: domain.NetworkEthereum, err: errors.New("rpc down")},
				"kamino:solana": &mockProvider{protocol: domain.ProtocolKamino, network: domain.NetworkSolana, err: errors.New("api down")},
			},
			wantHF:     0,
			wantErrors: 2,
		},
		{
			name:            "unknown protocol returns error",
			positions:       []domain.LendingPosition{ethPos},
			providers:       map[string]domain.HealthFactorProvider{},
			wantHF:          0,
			wantErrors:      1,
			wantErrContains: "no provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := application.NewCheckService(newTestConfig(tt.positions...), tt.providers)
			results := svc.CheckAll(context.Background())

			if len(results) != len(tt.positions) {
				t.Fatalf("got %d results, want %d", len(results), len(tt.positions))
			}

			for i, res := range results {
				if res.Position != tt.positions[i] {
					t.Errorf("result %d position mismatch: got %+v, want %+v", i, res.Position, tt.positions[i])
				}
				if err := res.Validate(); err != nil {
					t.Errorf("result %d invalid: %v", i, err)
				}
				if tt.wantErrContains != "" && res.Error != "" && !strings.Contains(res.Error, tt.wantErrContains) {
					t.Errorf("result %d error = %q, want containing %q", i, res.Error, tt.wantErrContains)
				}
			}

			hfCount, errCount := countResults(results)
			if hfCount != tt.wantHF {
				t.Errorf("health factors = %d, want %d", hfCount, tt.wantHF)
			}
			if errCount != tt.wantErrors {
				t.Errorf("errors = %d, want %d", errCount, tt.wantErrors)
			}
		})
	}
}

func TestCheckAllPreservesHealthFactor(t *testing.T) {
	ethPos := domain.LendingPosition{
		Wallet:   domain.Wallet{Address: "0x0000000000000000000000000000000000000001"},
		Protocol: domain.ProtocolAave,
		Network:  domain.NetworkEthereum,
	}
	safeHF := &domain.HealthFactor{Value: 2.5, Classification: domain.ClassificationSafe}
	providers := map[string]domain.HealthFactorProvider{
		"aave:ethereum": &mockProvider{protocol: domain.ProtocolAave, network: domain.NetworkEthereum, hf: safeHF},
	}

	svc := application.NewCheckService(newTestConfig(ethPos), providers)
	results := svc.CheckAll(context.Background())

	if len(results) != 1 || results[0].HealthFactor == nil {
		t.Fatalf("expected 1 success result, got %+v", results)
	}
	if got := results[0].HealthFactor.Value; got != 2.5 {
		t.Errorf("health factor value = %v, want 2.5", got)
	}
}

func TestCheckByProtocol(t *testing.T) {
	ethPos1 := domain.LendingPosition{
		Wallet:   domain.Wallet{Address: "0x0000000000000000000000000000000000000001"},
		Protocol: domain.ProtocolAave,
		Network:  domain.NetworkEthereum,
	}
	ethPos2 := domain.LendingPosition{
		Wallet:   domain.Wallet{Address: "0x0000000000000000000000000000000000000002"},
		Protocol: domain.ProtocolAave,
		Network:  domain.NetworkEthereum,
	}
	solPos := domain.LendingPosition{
		Wallet:   domain.Wallet{Address: "11111111111111111111111111111111"},
		Protocol: domain.ProtocolKamino,
		Network:  domain.NetworkSolana,
	}

	safeHF := &domain.HealthFactor{Value: 2.5, Classification: domain.ClassificationSafe}
	providers := map[string]domain.HealthFactorProvider{
		"aave:ethereum": &mockProvider{protocol: domain.ProtocolAave, network: domain.NetworkEthereum, hf: safeHF},
		"kamino:solana": &mockProvider{protocol: domain.ProtocolKamino, network: domain.NetworkSolana, err: errors.New("api down")},
	}

	svc := application.NewCheckService(newTestConfig(ethPos1, solPos, ethPos2), providers)

	t.Run("aave returns two success results", func(t *testing.T) {
		results := svc.CheckByProtocol(context.Background(), domain.ProtocolAave)
		if len(results) != 2 {
			t.Fatalf("got %d results, want 2", len(results))
		}
		for _, r := range results {
			if r.HealthFactor == nil {
				t.Errorf("expected health factor for %+v, got error: %q", r.Position, r.Error)
			}
		}
	})

	t.Run("kamino returns one error result", func(t *testing.T) {
		results := svc.CheckByProtocol(context.Background(), domain.ProtocolKamino)
		if len(results) != 1 {
			t.Fatalf("got %d results, want 1", len(results))
		}
		if results[0].Error == "" {
			t.Errorf("expected error, got success: %+v", results[0])
		}
	})

	t.Run("unknown protocol returns no results", func(t *testing.T) {
		results := svc.CheckByProtocol(context.Background(), "morpho")
		if len(results) != 0 {
			t.Fatalf("got %d results, want 0", len(results))
		}
	})
}
