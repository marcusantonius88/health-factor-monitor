package domain

import (
	"errors"
	"strings"
)

const (
	ClassificationSafe    = "safe"
	ClassificationWarning = "warning"
	ClassificationCritical = "critical"
)

func validClassifications() []string {
	return []string{ClassificationSafe, ClassificationWarning, ClassificationCritical}
}

func isValidClassification(c string) bool {
	for _, vc := range validClassifications() {
		if c == vc {
			return true
		}
	}
	return false
}

// HealthFactor represents the liquidation safety metric returned by a
// lending protocol for a specific position.
type HealthFactor struct {
	Value          float64
	Classification string
}

// Validate checks that the health factor fields meet the required constraints.
func (h HealthFactor) Validate() error {
	var errs []string
	if h.Value <= 0 {
		errs = append(errs, "value must be positive")
	}
	if !isValidClassification(h.Classification) {
		errs = append(errs, "classification must be one of: safe, warning, critical")
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

// ProviderResult represents the outcome of attempting to retrieve a Health
// Factor from a provider for a specific position. Exactly one of HealthFactor
// or Error is populated.
type ProviderResult struct {
	Position     LendingPosition
	HealthFactor *HealthFactor
	Error        string
}

// Validate checks that the provider result has exactly one of HealthFactor or
// Error populated.
func (r ProviderResult) Validate() error {
	hasHF := r.HealthFactor != nil
	hasErr := r.Error != ""
	if hasHF && hasErr {
		return errors.New("provider result must not have both health factor and error")
	}
	if !hasHF && !hasErr {
		return errors.New("provider result must have either health factor or error")
	}
	if hasHF {
		return r.HealthFactor.Validate()
	}
	return nil
}