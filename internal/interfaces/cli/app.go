package cli

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/marcus/health-factor-monitor/internal/domain"
)

// FormatResults formata os resultados dos providers com classificação visual em emojis.
//
// Saída esperada:
//
//	Health Factor
//	-------------
//	Base:	🟩 1.97
//	Solana:	🟩 2.22
//
// ou com "unavailable" em caso de erro do provider.
func FormatResults(results []domain.ProviderResult) string {
	var buf strings.Builder

	// Header
	buf.WriteString("Health Factor\n")
	buf.WriteString("-------------\n")
	buf.WriteString("HF mínimo desejável: 2,00\n")

	for _, r := range results {
		if r.Error != "" {
			fmt.Fprintf(&buf, "%s HF: unavailable\n", networkName(r.Position.Network))
		} else if r.HealthFactor == nil {
			fmt.Fprintf(&buf, "%s:\t%s\n", networkName(r.Position.Network), domain.EmojiForHealthFactor(0))
		} else if r.HealthFactor.Value > 1e50 {
			fmt.Fprintf(&buf, "%s:\t%s no active debt\n", networkName(r.Position.Network), domain.EmojiForHealthFactor(r.HealthFactor.Value))
		} else {
			fmt.Fprintf(&buf, "%s:\t%s %.2f\n", networkName(r.Position.Network), domain.EmojiForHealthFactor(r.HealthFactor.Value), r.HealthFactor.Value)
			if r.ContributionEstimate != nil &&
				r.HealthFactor.Value < domain.TargetHealthFactor &&
				isSupportedContributionPosition(r.Position, r.ContributionEstimate.AssetSymbol) {
				switch r.ContributionEstimate.Status {
				case domain.ContributionStatusAvailable:
					fmt.Fprintf(
						&buf,
						"\tAporte necessário (%s): US$ %s\n",
						r.ContributionEstimate.AssetSymbol,
						formatUSD(r.ContributionEstimate.AmountUSD),
					)
				case domain.ContributionStatusUnavailable:
					reason := strings.TrimSpace(r.ContributionEstimate.Reason)
					if reason == "" {
						reason = "dados insuficientes"
					}
					fmt.Fprintf(
						&buf,
						"\tAporte necessário (%s): indisponível (%s)\n",
						r.ContributionEstimate.AssetSymbol,
						reason,
					)
				}
			}
		}
	}
	return buf.String()
}

func isSupportedContributionPosition(position domain.LendingPosition, assetSymbol string) bool {
	return (position.Protocol == domain.ProtocolAave &&
		position.Network == domain.NetworkBase &&
		assetSymbol == "BTC") ||
		(position.Protocol == domain.ProtocolKamino &&
			position.Network == domain.NetworkSolana &&
			assetSymbol == "SOL")
}

func formatUSD(amount float64) string {
	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		return "0,00"
	}
	cents := int64(math.Round(amount * 100))
	whole := cents / 100
	fraction := cents % 100

	digits := strconv.FormatInt(whole, 10)
	var formatted strings.Builder
	firstGroup := len(digits) % 3
	if firstGroup == 0 {
		firstGroup = 3
	}
	formatted.WriteString(digits[:firstGroup])
	for i := firstGroup; i < len(digits); i += 3 {
		formatted.WriteByte('.')
		formatted.WriteString(digits[i : i+3])
	}
	return fmt.Sprintf("%s,%02d", formatted.String(), fraction)
}

// networkName retorna o nome da rede com base no valor do campo Network da
// posição, capitalizando o primeiro letra (ethereum -> Ethereum, solana -> Solana).
func networkName(net string) string {
	switch net {
	case "ethereum":
		return "Ethereum"
	case "base":
		return "Base"
	case "solana":
		return "Solana"
	default:
		return net
	}
}

// HasSuccess reporta se pelo menos um health factor foi recuperado com sucesso.
func HasSuccess(results []domain.ProviderResult) bool {
	for _, r := range results {
		if r.HealthFactor != nil {
			return true
		}
	}
	return false
}
