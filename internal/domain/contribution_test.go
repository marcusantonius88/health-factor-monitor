package domain

import (
	"math"
	"testing"
)

func TestContributionBasisValidate(t *testing.T) {
	tests := []struct {
		name  string
		basis ContributionBasis
		valid bool
	}{
		{
			name: "valid basis",
			basis: ContributionBasis{
				CurrentHealthFactor:          1.32,
				EffectiveDebtUSD:             100000,
				IncrementalLiquidationFactor: 0.82,
				PriceUSD:                     85000,
				AssetPrecision:               8,
				PriceFreshnessValidated:      true,
			},
			valid: true,
		},
		{
			name: "missing current health factor",
			basis: ContributionBasis{
				EffectiveDebtUSD:             100000,
				IncrementalLiquidationFactor: 0.82,
				PriceUSD:                     85000,
				AssetPrecision:               8,
				PriceFreshnessValidated:      true,
			},
			valid: false,
		},
		{
			name: "missing effective debt",
			basis: ContributionBasis{
				CurrentHealthFactor:          1.32,
				IncrementalLiquidationFactor: 0.82,
				PriceUSD:                     85000,
				AssetPrecision:               8,
				PriceFreshnessValidated:      true,
			},
			valid: false,
		},
		{
			name: "missing incremental liquidation factor",
			basis: ContributionBasis{
				CurrentHealthFactor:     1.32,
				EffectiveDebtUSD:        100000,
				PriceUSD:                85000,
				AssetPrecision:          8,
				PriceFreshnessValidated: true,
			},
			valid: false,
		},
		{
			name: "missing price",
			basis: ContributionBasis{
				CurrentHealthFactor:          1.32,
				EffectiveDebtUSD:             100000,
				IncrementalLiquidationFactor: 0.82,
				AssetPrecision:               8,
				PriceFreshnessValidated:      true,
			},
			valid: false,
		},
		{
			name: "negative effective debt",
			basis: ContributionBasis{
				CurrentHealthFactor:          1.32,
				EffectiveDebtUSD:             -1,
				IncrementalLiquidationFactor: 0.82,
				PriceUSD:                     85000,
				AssetPrecision:               8,
				PriceFreshnessValidated:      true,
			},
			valid: false,
		},
		{
			name: "negative incremental liquidation factor",
			basis: ContributionBasis{
				CurrentHealthFactor:          1.32,
				EffectiveDebtUSD:             100000,
				IncrementalLiquidationFactor: -0.82,
				PriceUSD:                     85000,
				AssetPrecision:               8,
				PriceFreshnessValidated:      true,
			},
			valid: false,
		},
		{
			name: "negative price",
			basis: ContributionBasis{
				CurrentHealthFactor:          1.32,
				EffectiveDebtUSD:             100000,
				IncrementalLiquidationFactor: 0.82,
				PriceUSD:                     -85000,
				AssetPrecision:               8,
				PriceFreshnessValidated:      true,
			},
			valid: false,
		},
		{
			name: "non-finite current health factor",
			basis: ContributionBasis{
				CurrentHealthFactor:          math.Inf(1),
				EffectiveDebtUSD:             100000,
				IncrementalLiquidationFactor: 0.82,
				PriceUSD:                     85000,
				AssetPrecision:               8,
				PriceFreshnessValidated:      true,
			},
			valid: false,
		},
		{
			name: "non-finite effective debt",
			basis: ContributionBasis{
				CurrentHealthFactor:          1.32,
				EffectiveDebtUSD:             math.NaN(),
				IncrementalLiquidationFactor: 0.82,
				PriceUSD:                     85000,
				AssetPrecision:               8,
				PriceFreshnessValidated:      true,
			},
			valid: false,
		},
		{
			name: "negative asset precision",
			basis: ContributionBasis{
				CurrentHealthFactor:          1.32,
				EffectiveDebtUSD:             100000,
				IncrementalLiquidationFactor: 0.82,
				PriceUSD:                     85000,
				AssetPrecision:               -1,
				PriceFreshnessValidated:      true,
			},
			valid: false,
		},
		{
			name: "unvalidated price freshness",
			basis: ContributionBasis{
				CurrentHealthFactor:          1.32,
				EffectiveDebtUSD:             100000,
				IncrementalLiquidationFactor: 0.82,
				PriceUSD:                     85000,
				AssetPrecision:               8,
				PriceFreshnessValidated:      false,
			},
			valid: false,
		},
		{
			name: "health factor at target is not contribution basis",
			basis: ContributionBasis{
				CurrentHealthFactor:          1.5,
				EffectiveDebtUSD:             100000,
				IncrementalLiquidationFactor: 0.82,
				PriceUSD:                     85000,
				AssetPrecision:               8,
				PriceFreshnessValidated:      true,
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.basis.Validate()
			if tt.valid && err != nil {
				t.Fatalf("ContributionBasis.Validate() error = %v, want nil", err)
			}
			if !tt.valid && err == nil {
				t.Fatal("ContributionBasis.Validate() error = nil, want error")
			}
		})
	}
}

func TestContributionEstimateValidate(t *testing.T) {
	tests := []struct {
		name     string
		estimate ContributionEstimate
		valid    bool
	}{
		{
			name: "available BTC estimate",
			estimate: ContributionEstimate{
				Status:             "available",
				AssetSymbol:        "BTC",
				AmountUSD:          850.0,
				TargetHealthFactor: 1.5,
			},
			valid: true,
		},
		{
			name: "available SOL estimate",
			estimate: ContributionEstimate{
				Status:             "available",
				AssetSymbol:        "SOL",
				AmountUSD:          620.0,
				TargetHealthFactor: 1.5,
			},
			valid: true,
		},
		{
			name: "unavailable BTC estimate",
			estimate: ContributionEstimate{
				Status:             "unavailable",
				AssetSymbol:        "BTC",
				Reason:             "dados de preço indisponíveis",
				TargetHealthFactor: 1.5,
			},
			valid: true,
		},
		{
			name: "unavailable SOL estimate",
			estimate: ContributionEstimate{
				Status:             "unavailable",
				AssetSymbol:        "SOL",
				Reason:             "parâmetros de risco não confirmados",
				TargetHealthFactor: 1.5,
			},
			valid: true,
		},
		{
			name: "available estimate with missing symbol",
			estimate: ContributionEstimate{
				Status:             "available",
				AmountUSD:          850,
				TargetHealthFactor: 1.5,
			},
			valid: false,
		},
		{
			name: "available estimate with unsupported symbol",
			estimate: ContributionEstimate{
				Status:             "available",
				AssetSymbol:        "ETH",
				AmountUSD:          850,
				TargetHealthFactor: 1.5,
			},
			valid: false,
		},
		{
			name: "available estimate with zero amount",
			estimate: ContributionEstimate{
				Status:             "available",
				AssetSymbol:        "BTC",
				AmountUSD:          0,
				TargetHealthFactor: 1.5,
			},
			valid: false,
		},
		{
			name: "available estimate with negative amount",
			estimate: ContributionEstimate{
				Status:             "available",
				AssetSymbol:        "SOL",
				AmountUSD:          -620,
				TargetHealthFactor: 1.5,
			},
			valid: false,
		},
		{
			name: "available estimate with non-finite amount",
			estimate: ContributionEstimate{
				Status:             "available",
				AssetSymbol:        "BTC",
				AmountUSD:          math.Inf(1),
				TargetHealthFactor: 1.5,
			},
			valid: false,
		},
		{
			name: "available estimate with reason",
			estimate: ContributionEstimate{
				Status:             "available",
				AssetSymbol:        "BTC",
				AmountUSD:          850,
				Reason:             "not allowed for available estimate",
				TargetHealthFactor: 1.5,
			},
			valid: false,
		},
		{
			name: "unavailable estimate with no reason",
			estimate: ContributionEstimate{
				Status:             "unavailable",
				AssetSymbol:        "SOL",
				TargetHealthFactor: 1.5,
			},
			valid: false,
		},
		{
			name: "unavailable estimate with numeric amount",
			estimate: ContributionEstimate{
				Status:             "unavailable",
				AssetSymbol:        "BTC",
				AmountUSD:          850,
				Reason:             "dados incompletos",
				TargetHealthFactor: 1.5,
			},
			valid: false,
		},
		{
			name: "invalid status",
			estimate: ContributionEstimate{
				Status:             "unknown",
				AssetSymbol:        "BTC",
				TargetHealthFactor: 1.5,
			},
			valid: false,
		},
		{
			name: "wrong target health factor",
			estimate: ContributionEstimate{
				Status:             "available",
				AssetSymbol:        "BTC",
				AmountUSD:          850,
				TargetHealthFactor: 1.4,
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.estimate.Validate()
			if tt.valid && err != nil {
				t.Fatalf("ContributionEstimate.Validate() error = %v, want nil", err)
			}
			if !tt.valid && err == nil {
				t.Fatal("ContributionEstimate.Validate() error = nil, want error")
			}
		})
	}
}

func TestPositionSnapshotValidate(t *testing.T) {
	validPosition := LendingPosition{
		Wallet:   Wallet{Address: validEthAddress},
		Protocol: ProtocolAave,
		Network:  NetworkBase,
	}
	validHF := &HealthFactor{Value: 1.32, Classification: ClassificationWarning}
	validBasis := &ContributionBasis{
		CurrentHealthFactor:          1.32,
		EffectiveDebtUSD:             100000,
		IncrementalLiquidationFactor: 0.82,
		PriceUSD:                     85000,
		AssetPrecision:               8,
		PriceFreshnessValidated:      true,
	}

	tests := []struct {
		name     string
		snapshot PositionSnapshot
		valid    bool
	}{
		{
			name: "available basis snapshot",
			snapshot: PositionSnapshot{
				Position:          validPosition,
				HealthFactor:      validHF,
				ContributionBasis: validBasis,
			},
			valid: true,
		},
		{
			name: "unavailable estimate snapshot",
			snapshot: PositionSnapshot{
				Position:                      validPosition,
				HealthFactor:                  validHF,
				ContributionUnavailableReason: "dados de preço indisponíveis",
			},
			valid: true,
		},
		{
			name: "safe snapshot has no estimate data",
			snapshot: PositionSnapshot{
				Position:     validPosition,
				HealthFactor: &HealthFactor{Value: 1.5, Classification: ClassificationSafe},
			},
			valid: true,
		},
		{
			name: "basis and unavailable reason are mutually exclusive",
			snapshot: PositionSnapshot{
				Position:                      validPosition,
				HealthFactor:                  validHF,
				ContributionBasis:             validBasis,
				ContributionUnavailableReason: "dados incompletos",
			},
			valid: false,
		},
		{
			name: "missing health factor",
			snapshot: PositionSnapshot{
				Position: validPosition,
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.snapshot.Validate()
			if tt.valid && err != nil {
				t.Fatalf("PositionSnapshot.Validate() error = %v, want nil", err)
			}
			if !tt.valid && err == nil {
				t.Fatal("PositionSnapshot.Validate() error = nil, want error")
			}
		})
	}
}

func TestProviderResultContributionEstimate(t *testing.T) {
	position := LendingPosition{
		Wallet:   Wallet{Address: validEthAddress},
		Protocol: ProtocolAave,
		Network:  NetworkBase,
	}
	belowTarget := &HealthFactor{Value: 1.32, Classification: ClassificationWarning}

	tests := []struct {
		name    string
		result  ProviderResult
		wantErr bool
	}{
		{
			name: "optional available estimate preserves successful health factor",
			result: ProviderResult{
				Position:     position,
				HealthFactor: belowTarget,
				ContributionEstimate: &ContributionEstimate{
					Status:             ContributionStatusAvailable,
					AssetSymbol:        "BTC",
					AmountUSD:          850,
					TargetHealthFactor: TargetHealthFactor,
				},
			},
			wantErr: false,
		},
		{
			name: "optional unavailable estimate preserves successful health factor",
			result: ProviderResult{
				Position:     position,
				HealthFactor: belowTarget,
				ContributionEstimate: &ContributionEstimate{
					Status:             ContributionStatusUnavailable,
					AssetSymbol:        "BTC",
					Reason:             "dados de preço indisponíveis",
					TargetHealthFactor: TargetHealthFactor,
				},
			},
			wantErr: false,
		},
		{
			name: "estimate is not allowed at target",
			result: ProviderResult{
				Position:     position,
				HealthFactor: &HealthFactor{Value: 1.5, Classification: ClassificationSafe},
				ContributionEstimate: &ContributionEstimate{
					Status:             ContributionStatusAvailable,
					AssetSymbol:        "BTC",
					AmountUSD:          850,
					TargetHealthFactor: TargetHealthFactor,
				},
			},
			wantErr: true,
		},
		{
			name: "estimate is not allowed with provider error",
			result: ProviderResult{
				Position: position,
				Error:    "connection failed",
				ContributionEstimate: &ContributionEstimate{
					Status:             ContributionStatusUnavailable,
					AssetSymbol:        "BTC",
					Reason:             "dados incompletos",
					TargetHealthFactor: TargetHealthFactor,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.result.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ProviderResult.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
