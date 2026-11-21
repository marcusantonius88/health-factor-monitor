package domain

import (
	"errors"
	"math"
	"strings"
)

const (
	ContributionStatusAvailable   = "available"
	ContributionStatusUnavailable = "unavailable"
	TargetHealthFactor            = 1.5
)

// ContributionAsset identifies the protocol asset used as additional
// collateral. It is protocol-specific to preserve the distinction between
// BTC representations and SOL.
type ContributionAsset struct {
	Symbol     string
	Identifier string
}

// Validate checks whether the contribution asset is one of the supported
// display assets and has a protocol-specific market identifier.
func (a ContributionAsset) Validate() error {
	if a.Symbol != "BTC" && a.Symbol != "SOL" {
		return errors.New("asset symbol must be BTC or SOL")
	}
	if strings.TrimSpace(a.Identifier) == "" {
		return errors.New("asset identifier must not be empty")
	}
	return nil
}

// ContributionBasis contains protocol-normalized inputs needed to estimate
// additional collateral. It contains no protocol transport or calculation code.
type ContributionBasis struct {
	CurrentHealthFactor          float64
	EffectiveDebtUSD             float64
	IncrementalLiquidationFactor float64
	Asset                        ContributionAsset
	PriceUSD                     float64
	AssetPrecision               int
	PriceFreshnessValidated      bool
}

// Validate checks that the normalized basis can represent a below-target
// contribution estimate. A zero Asset is allowed here because protocol
// adapters supply it only after live eligibility has been verified.
func (b ContributionBasis) Validate() error {
	var errs []string

	if math.IsNaN(b.CurrentHealthFactor) || math.IsInf(b.CurrentHealthFactor, 0) ||
		b.CurrentHealthFactor <= 0 {
		errs = append(errs, "current health factor must be positive and finite")
	} else if b.CurrentHealthFactor >= TargetHealthFactor {
		errs = append(errs, "current health factor must be below target")
	}

	if math.IsNaN(b.EffectiveDebtUSD) || math.IsInf(b.EffectiveDebtUSD, 0) ||
		b.EffectiveDebtUSD <= 0 {
		errs = append(errs, "effective debt USD must be positive and finite")
	}
	if math.IsNaN(b.IncrementalLiquidationFactor) || math.IsInf(b.IncrementalLiquidationFactor, 0) ||
		b.IncrementalLiquidationFactor <= 0 {
		errs = append(errs, "incremental liquidation factor must be positive and finite")
	}
	if math.IsNaN(b.PriceUSD) || math.IsInf(b.PriceUSD, 0) || b.PriceUSD <= 0 {
		errs = append(errs, "price USD must be positive and finite")
	}
	if b.AssetPrecision < 0 {
		errs = append(errs, "asset precision must not be negative")
	}
	if !b.PriceFreshnessValidated {
		errs = append(errs, "price freshness must be validated")
	}
	if b.Asset != (ContributionAsset{}) {
		if err := b.Asset.Validate(); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

// ContributionEstimate is the informational result exposed to application
// and interface consumers. Available estimates contain a positive USD amount;
// unavailable estimates contain only a human-readable reason.
type ContributionEstimate struct {
	Status             string
	AssetSymbol        string
	AmountUSD          float64
	Reason             string
	TargetHealthFactor float64
}

// Validate checks the estimate's status and prevents numeric fallback values
// from being attached to unavailable estimates.
func (e ContributionEstimate) Validate() error {
	var errs []string

	if e.TargetHealthFactor != TargetHealthFactor {
		errs = append(errs, "target health factor must be 1.50")
	}
	if e.AssetSymbol != "BTC" && e.AssetSymbol != "SOL" {
		errs = append(errs, "asset symbol must be BTC or SOL")
	}

	switch e.Status {
	case ContributionStatusAvailable:
		if math.IsNaN(e.AmountUSD) || math.IsInf(e.AmountUSD, 0) || e.AmountUSD <= 0 {
			errs = append(errs, "available amount USD must be positive and finite")
		}
		if strings.TrimSpace(e.Reason) != "" {
			errs = append(errs, "available estimate must not have a reason")
		}
	case ContributionStatusUnavailable:
		if e.AmountUSD != 0 {
			errs = append(errs, "unavailable estimate must not have an amount")
		}
		if strings.TrimSpace(e.Reason) == "" {
			errs = append(errs, "unavailable estimate must have a reason")
		}
	default:
		errs = append(errs, "contribution estimate status must be available or unavailable")
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

// PositionSnapshot is a coherent provider read for one lending position.
// Estimate-only data may be unavailable without invalidating a successful
// Health Factor read.
type PositionSnapshot struct {
	Position                      LendingPosition
	HealthFactor                  *HealthFactor
	ContributionBasis             *ContributionBasis
	ContributionUnavailableReason string
}

// Validate checks that a snapshot retains independent Health Factor and
// estimate availability invariants.
func (s PositionSnapshot) Validate() error {
	var errs []string

	if err := s.Position.Validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if s.HealthFactor == nil {
		errs = append(errs, "health factor must be present")
	} else if err := s.HealthFactor.Validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if s.ContributionBasis != nil {
		if err := s.ContributionBasis.Validate(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if s.ContributionBasis != nil && strings.TrimSpace(s.ContributionUnavailableReason) != "" {
		errs = append(errs, "snapshot must not have both contribution basis and unavailable reason")
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}
