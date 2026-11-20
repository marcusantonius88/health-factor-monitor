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
