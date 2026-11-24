package cli

import (
	"math"
	"strings"
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

func TestFormatResultsContributionEstimateLines(t *testing.T) {
	results := []domain.ProviderResult{
		{
			Position: domain.LendingPosition{
				Protocol: domain.ProtocolAave,
				Network:  domain.NetworkBase,
			},
			HealthFactor: &domain.HealthFactor{
				Value:          1.32,
				Classification: domain.ClassificationWarning,
			},
			ContributionEstimate: &domain.ContributionEstimate{
				Status:             domain.ContributionStatusAvailable,
				AssetSymbol:        "BTC",
				AmountUSD:          850,
				TargetHealthFactor: domain.TargetHealthFactor,
			},
		},
		{
			Position: domain.LendingPosition{
				Protocol: domain.ProtocolKamino,
				Network:  domain.NetworkSolana,
			},
			HealthFactor: &domain.HealthFactor{
				Value:          1.08,
				Classification: domain.ClassificationCritical,
			},
			ContributionEstimate: &domain.ContributionEstimate{
				Status:             domain.ContributionStatusAvailable,
				AssetSymbol:        "SOL",
				AmountUSD:          620,
				TargetHealthFactor: domain.TargetHealthFactor,
			},
		},
	}

	got := FormatResults(results)
	want := "Health Factor\n-------------\n" +
		"Base:\t🟨 1.32\n" +
		"\tAporte necessário (BTC): US$ 850,00\n" +
		"Solana:\t🟥 1.08\n" +
		"\tAporte necessário (SOL): US$ 620,00\n"
	if got != want {
		t.Fatalf("FormatResults() = %q, want %q", got, want)
	}
}

func TestFormatResultsOmitsContributionLinesForSafeAndOutOfScope(t *testing.T) {
	results := []domain.ProviderResult{
		{
			Position: domain.LendingPosition{
				Protocol: domain.ProtocolAave,
				Network:  domain.NetworkBase,
			},
			HealthFactor: &domain.HealthFactor{
				Value:          1.5,
				Classification: domain.ClassificationSafe,
			},
		},
		{
			Position: domain.LendingPosition{
				Protocol: domain.ProtocolAave,
				Network:  domain.NetworkEthereum,
			},
			HealthFactor: &domain.HealthFactor{
				Value:          1.32,
				Classification: domain.ClassificationWarning,
			},
			ContributionEstimate: &domain.ContributionEstimate{
				Status:             domain.ContributionStatusAvailable,
				AssetSymbol:        "BTC",
				AmountUSD:          850,
				TargetHealthFactor: domain.TargetHealthFactor,
			},
		},
	}

	got := FormatResults(results)
	want := "Health Factor\n-------------\n" +
		"Base:\t🟩 1.50\n" +
		"Ethereum:\t🟨 1.32\n"
	if got != want {
		t.Fatalf("FormatResults() = %q, want %q", got, want)
	}
}

func TestFormatResultsUnavailableContributionEstimates(t *testing.T) {
	results := []domain.ProviderResult{
		{
			Position: domain.LendingPosition{
				Protocol: domain.ProtocolAave,
				Network:  domain.NetworkBase,
			},
			HealthFactor: &domain.HealthFactor{
				Value:          1.32,
				Classification: domain.ClassificationWarning,
			},
			ContributionEstimate: &domain.ContributionEstimate{
				Status:             domain.ContributionStatusUnavailable,
				AssetSymbol:        "BTC",
				Reason:             "dados de preço indisponíveis",
				TargetHealthFactor: domain.TargetHealthFactor,
			},
		},
		{
			Position: domain.LendingPosition{
				Protocol: domain.ProtocolKamino,
				Network:  domain.NetworkSolana,
			},
			HealthFactor: &domain.HealthFactor{
				Value:          1.08,
				Classification: domain.ClassificationCritical,
			},
			ContributionEstimate: &domain.ContributionEstimate{
				Status:             domain.ContributionStatusUnavailable,
				AssetSymbol:        "SOL",
				Reason:             "parâmetros de risco não confirmados",
				TargetHealthFactor: domain.TargetHealthFactor,
			},
		},
	}

	got := FormatResults(results)
	want := "Health Factor\n-------------\n" +
		"Base:\t🟨 1.32\n" +
		"\tAporte necessário (BTC): indisponível (dados de preço indisponíveis)\n" +
		"Solana:\t🟥 1.08\n" +
		"\tAporte necessário (SOL): indisponível (parâmetros de risco não confirmados)\n"
	if got != want {
		t.Fatalf("FormatResults() = %q, want %q", got, want)
	}

	for _, forbidden := range []string{"US$", "850", "620", "BTC quantity", "SOL quantity", "unit price"} {
		if strings.Contains(got, forbidden) {
			t.Errorf("unavailable output contains prohibited text %q: %q", forbidden, got)
		}
	}
}
