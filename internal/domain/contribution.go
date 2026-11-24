package domain

import (
	"errors"
	"math"
	"math/big"
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

// CalculateContributionEstimate calculates the minimum contribution that
// reaches the fixed 1.50 Health Factor target. It rounds upward to the asset's
// smallest supported unit and then to display cents, and rejects a result that
// does not meet the target after rounding.
func CalculateContributionEstimate(b ContributionBasis) (*ContributionEstimate, error) {
	if !isFinitePositive(b.CurrentHealthFactor) {
		return nil, errors.New("current health factor must be positive and finite")
	}
	if b.CurrentHealthFactor >= TargetHealthFactor {
		return nil, nil
	}
	if err := b.Validate(); err != nil {
		return nil, err
	}
	if b.Asset == (ContributionAsset{}) {
		return nil, errors.New("contribution asset must be specified")
	}
	if err := b.Asset.Validate(); err != nil {
		return nil, err
	}

	h := decimalRat(b.CurrentHealthFactor)
	debt := decimalRat(b.EffectiveDebtUSD)
	factor := decimalRat(b.IncrementalLiquidationFactor)
	price := decimalRat(b.PriceUSD)
	target := new(big.Rat).SetFloat64(TargetHealthFactor)

	if h == nil || debt == nil || factor == nil || price == nil ||
		h.Sign() <= 0 || debt.Sign() <= 0 || factor.Sign() <= 0 || price.Sign() <= 0 {
		return nil, errors.New("contribution basis contains invalid numeric values")
	}

	deficit := new(big.Rat).Sub(target, h)
	if deficit.Sign() <= 0 {
		return nil, nil
	}
	requiredUSD := new(big.Rat).Mul(deficit, debt)
	requiredUSD.Quo(requiredUSD, factor)

	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(b.AssetPrecision)), nil)
	scaleRat := new(big.Rat).SetInt(scale)
	requiredQuantity := new(big.Rat).Quo(requiredUSD, price)
	scaledQuantity := new(big.Rat).Mul(requiredQuantity, scaleRat)

	scaledUnits := ceilRat(scaledQuantity)
	if scaledUnits.Sign() <= 0 {
		scaledUnits.SetInt64(1)
	}

	unitUSD := new(big.Rat).Mul(new(big.Rat).SetInt(scaledUnits), price)
	unitUSD.Quo(unitUSD, scaleRat)
	centAmount := new(big.Rat).Mul(unitUSD, big.NewRat(100, 1))
	centUnits := ceilRat(centAmount)
	if centUnits.Sign() <= 0 {
		centUnits.SetInt64(1)
	}

	centUSD := new(big.Rat).SetInt(centUnits)
	centUSD.Quo(centUSD, big.NewRat(100, 1))
	amountUSD, _ := centUSD.Float64()

	projected := new(big.Rat).Set(h)
	addedCollateral := new(big.Rat).Mul(centUSD, factor)
	addedCapacity := new(big.Rat).Quo(addedCollateral, debt)
	projected.Add(projected, addedCapacity)
	if projected.Cmp(target) < 0 {
		return nil, errors.New("rounded contribution does not reach target health factor")
	}

	return &ContributionEstimate{
		Status:             ContributionStatusAvailable,
		AssetSymbol:        b.Asset.Symbol,
		AmountUSD:          amountUSD,
		TargetHealthFactor: TargetHealthFactor,
	}, nil
}

func isFinitePositive(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value > 0
}

func decimalRat(value float64) *big.Rat {
	text := big.NewFloat(value).Text('g', -1)
	r, ok := new(big.Rat).SetString(text)
	if !ok {
		return nil
	}
	return r
}

func ceilRat(value *big.Rat) *big.Int {
	quotient := new(big.Int)
	remainder := new(big.Int)
	quotient.QuoRem(value.Num(), value.Denom(), remainder)
	if remainder.Sign() != 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	return quotient
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
