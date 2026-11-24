package application_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/marcus/health-factor-monitor/internal/application"
	"github.com/marcus/health-factor-monitor/internal/domain"
	"github.com/marcus/health-factor-monitor/internal/interfaces/cli"
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

type mockSnapshotProvider struct {
	mockProvider
	snapshot      *domain.PositionSnapshot
	snapshotErr   error
	snapshotCalls int
}

func (m *mockSnapshotProvider) GetPositionSnapshot(ctx context.Context, address string) (*domain.PositionSnapshot, error) {
	m.snapshotCalls++
	if m.snapshotErr != nil {
		return nil, m.snapshotErr
	}
	return m.snapshot, nil
}

var _ domain.ContributionSnapshotProvider = (*mockSnapshotProvider)(nil)

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

func TestCheckPositionAttachesBelowTargetContributionEstimate(t *testing.T) {
	position := domain.LendingPosition{
		Wallet:   domain.Wallet{Address: "0x0000000000000000000000000000000000000001"},
		Protocol: domain.ProtocolAave,
		Network:  domain.NetworkBase,
	}
	hf := &domain.HealthFactor{Value: 1.32, Classification: domain.ClassificationWarning}
	snapshot := &domain.PositionSnapshot{
		Position:     position,
		HealthFactor: hf,
		ContributionBasis: &domain.ContributionBasis{
			CurrentHealthFactor:          1.32,
			EffectiveDebtUSD:             100000,
			IncrementalLiquidationFactor: 0.82,
			Asset:                        domain.ContributionAsset{Symbol: "BTC", Identifier: "0xbtc"},
			PriceUSD:                     85000,
			AssetPrecision:               8,
			PriceFreshnessValidated:      true,
		},
	}
	provider := &mockSnapshotProvider{
		mockProvider: mockProvider{
			protocol: domain.ProtocolAave,
			network:  domain.NetworkBase,
			hf:       hf,
		},
		snapshot: snapshot,
	}
	providers := map[string]domain.HealthFactorProvider{
		"aave:base": provider,
	}

	svc := application.NewCheckService(newTestConfig(position), providers)
	results := svc.CheckAll(context.Background())

	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if provider.snapshotCalls != 1 {
		t.Errorf("snapshot calls = %d, want 1", provider.snapshotCalls)
	}
	if results[0].HealthFactor == nil || results[0].HealthFactor.Value != 1.32 {
		t.Fatalf("HealthFactor = %+v, want 1.32", results[0].HealthFactor)
	}
	if results[0].Error != "" {
		t.Fatalf("estimate path changed Health Factor result into provider error: %q", results[0].Error)
	}
	if results[0].ContributionEstimate == nil {
		t.Fatal("expected contribution estimate for below-target position")
	}
	if err := results[0].Validate(); err != nil {
		t.Fatalf("ProviderResult.Validate() error = %v", err)
	}
}

func TestCheckPositionSafeSkipsSnapshotEstimateReads(t *testing.T) {
	position := domain.LendingPosition{
		Wallet:   domain.Wallet{Address: "11111111111111111111111111111111"},
		Protocol: domain.ProtocolKamino,
		Network:  domain.NetworkSolana,
	}
	hf := &domain.HealthFactor{Value: 1.5, Classification: domain.ClassificationSafe}
	provider := &mockSnapshotProvider{
		mockProvider: mockProvider{
			protocol: domain.ProtocolKamino,
			network:  domain.NetworkSolana,
			hf:       hf,
		},
		snapshot: &domain.PositionSnapshot{
			Position:     position,
			HealthFactor: hf,
		},
	}
	providers := map[string]domain.HealthFactorProvider{
		"kamino:solana": provider,
	}

	svc := application.NewCheckService(newTestConfig(position), providers)
	results := svc.CheckAll(context.Background())

	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if provider.snapshotCalls != 0 {
		t.Errorf("snapshot calls = %d, want 0 for safe position", provider.snapshotCalls)
	}
	if results[0].HealthFactor == nil || results[0].HealthFactor.Value != 1.5 {
		t.Fatalf("HealthFactor = %+v, want 1.5", results[0].HealthFactor)
	}
	if results[0].ContributionEstimate != nil {
		t.Fatalf("safe position must not have contribution estimate: %+v", results[0].ContributionEstimate)
	}
}

func TestEstimateOnlySnapshotErrorPreservesHealthFactor(t *testing.T) {
	position := domain.LendingPosition{
		Wallet:   domain.Wallet{Address: "0x0000000000000000000000000000000000000001"},
		Protocol: domain.ProtocolAave,
		Network:  domain.NetworkBase,
	}
	hf := &domain.HealthFactor{Value: 1.32, Classification: domain.ClassificationWarning}
	provider := &mockSnapshotProvider{
		mockProvider: mockProvider{
			protocol: domain.ProtocolAave,
			network:  domain.NetworkBase,
			hf:       hf,
		},
		snapshotErr: errors.New("oracle auxiliary request failed"),
	}

	svc := application.NewCheckService(
		newTestConfig(position),
		map[string]domain.HealthFactorProvider{"aave:base": provider},
	)
	results := svc.CheckAll(context.Background())

	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	result := results[0]
	if result.Error != "" {
		t.Errorf("estimate-only error populated Health Factor provider error: %q", result.Error)
	}
	if result.HealthFactor == nil || result.HealthFactor.Value != 1.32 {
		t.Fatalf("HealthFactor = %+v, want preserved 1.32", result.HealthFactor)
	}
	estimate := result.ContributionEstimate
	if estimate == nil {
		t.Fatal("snapshot read error must create an unavailable contribution estimate")
	}
	if estimate.Status != domain.ContributionStatusUnavailable {
		t.Errorf("Status = %q, want unavailable", estimate.Status)
	}
	if estimate.AmountUSD != 0 {
		t.Errorf("AmountUSD = %v, want 0", estimate.AmountUSD)
	}
	if estimate.Reason == "" {
		t.Fatal("snapshot read error must include an unavailable reason")
	}
	if !cli.HasSuccess(results) {
		t.Fatal("estimate-only failure changed CLI success/exit-status behavior")
	}
}

func TestEstimateUnavailableReasonPreservesHealthFactorAndHasNoAmount(t *testing.T) {
	position := domain.LendingPosition{
		Wallet:   domain.Wallet{Address: "11111111111111111111111111111111"},
		Protocol: domain.ProtocolKamino,
		Network:  domain.NetworkSolana,
	}
	hf := &domain.HealthFactor{Value: 1.08, Classification: domain.ClassificationCritical}
	provider := &mockSnapshotProvider{
		mockProvider: mockProvider{
			protocol: domain.ProtocolKamino,
			network:  domain.NetworkSolana,
			hf:       hf,
		},
		snapshot: &domain.PositionSnapshot{
			Position:                      position,
			HealthFactor:                  hf,
			ContributionUnavailableReason: "parâmetros de elevação não confirmados",
		},
	}

	svc := application.NewCheckService(
		newTestConfig(position),
		map[string]domain.HealthFactorProvider{"kamino:solana": provider},
	)
	results := svc.CheckAll(context.Background())

	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	result := results[0]
	if result.Error != "" {
		t.Errorf("estimate unavailability populated provider error: %q", result.Error)
	}
	if result.HealthFactor == nil || result.HealthFactor.Value != 1.08 {
		t.Fatalf("HealthFactor = %+v, want preserved 1.08", result.HealthFactor)
	}
	estimate := result.ContributionEstimate
	if estimate == nil {
		t.Fatal("expected unavailable contribution estimate")
	}
	if estimate.Status != domain.ContributionStatusUnavailable {
		t.Errorf("Status = %q, want unavailable", estimate.Status)
	}
	if estimate.AmountUSD != 0 {
		t.Errorf("AmountUSD = %v, want 0", estimate.AmountUSD)
	}
	if estimate.Reason == "" {
		t.Fatal("unavailable estimate reason must be explicit")
	}
	if err := result.Validate(); err != nil {
		t.Fatalf("ProviderResult.Validate() error = %v", err)
	}
}

func TestEstimateFailureDoesNotSuppressOtherPositions(t *testing.T) {
	aavePosition := domain.LendingPosition{
		Wallet:   domain.Wallet{Address: "0x0000000000000000000000000000000000000001"},
		Protocol: domain.ProtocolAave,
		Network:  domain.NetworkBase,
	}
	kaminoPosition := domain.LendingPosition{
		Wallet:   domain.Wallet{Address: "11111111111111111111111111111111"},
		Protocol: domain.ProtocolKamino,
		Network:  domain.NetworkSolana,
	}

	aaveHF := &domain.HealthFactor{Value: 1.32, Classification: domain.ClassificationWarning}
	kaminoHF := &domain.HealthFactor{Value: 1.08, Classification: domain.ClassificationCritical}
	aaveProvider := &mockSnapshotProvider{
		mockProvider: mockProvider{
			protocol: domain.ProtocolAave,
			network:  domain.NetworkBase,
			hf:       aaveHF,
		},
		snapshotErr: errors.New("Aave reserve data unavailable"),
	}
	kaminoProvider := &mockSnapshotProvider{
		mockProvider: mockProvider{
			protocol: domain.ProtocolKamino,
			network:  domain.NetworkSolana,
			hf:       kaminoHF,
		},
		snapshot: &domain.PositionSnapshot{
			Position:                      kaminoPosition,
			HealthFactor:                  kaminoHF,
			ContributionUnavailableReason: "dados SOL incompletos",
		},
	}

	svc := application.NewCheckService(
		newTestConfig(aavePosition, kaminoPosition),
		map[string]domain.HealthFactorProvider{
			"aave:base":     aaveProvider,
			"kamino:solana": kaminoProvider,
		},
	)
	results := svc.CheckAll(context.Background())

	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	if results[0].Error != "" || results[0].HealthFactor == nil {
		t.Errorf("Aave result was suppressed: %+v", results[0])
	}
	if results[1].Error != "" || results[1].HealthFactor == nil {
		t.Errorf("Kamino result was suppressed: %+v", results[1])
	}
	if results[1].ContributionEstimate == nil ||
		results[1].ContributionEstimate.Status != domain.ContributionStatusUnavailable {
		t.Errorf("Kamino unavailable estimate = %+v, want unavailable", results[1].ContributionEstimate)
	}
	if !cli.HasSuccess(results) {
		t.Fatal("valid Health Factors should preserve CLI success")
	}
}
