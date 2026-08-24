package cli

import (
	"math"
	"testing"

	"github.com/marcus/health-factor-monitor/internal/domain"
)

func TestFormatResults(t *testing.T) {
	results := []domain.ProviderResult{
		{
			Position: domain.LendingPosition{Network: domain.NetworkEthereum},
			HealthFactor: &domain.HealthFactor{
				Value:          math.Inf(1),
				Classification: domain.ClassificationSafe,
			},
		},
		{
			Position: domain.LendingPosition{Network: domain.NetworkSolana},
			HealthFactor: &domain.HealthFactor{
				Value:          2.18,
				Classification: domain.ClassificationSafe,
			},
		},
	}

	got := FormatResults(results)
	// Novo formato: Health Factor header + alinhamento com tabulação
	want := "Health Factor\n-------------\nEthereum:\t🟩 no active debt\nSolana:\t🟩 2.18\n"
	if got != want {
		t.Fatalf("FormatResults() = %q, want %q", got, want)
	}
}
