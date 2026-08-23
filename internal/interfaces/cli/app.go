package cli

import (
	"fmt"
	"strings"

	"github.com/marcus/health-factor-monitor/internal/domain"
)

// FormatResults formata os resultados dos providers como linhas simples:
//
//	Ethereum HF: 1.96
//	Solana HF:   2.21
//
// ou com "unavailable" em caso de erro do provider.
func FormatResults(results []domain.ProviderResult) string {
	var buf strings.Builder
	for _, r := range results {
		if r.Error != "" {
			fmt.Fprintf(&buf, "%s HF: unavailable\n", networkName(r.Position.Network))
		} else if r.HealthFactor != nil && r.HealthFactor.Value > 1e50 {
			fmt.Fprintf(&buf, "%s HF: no active debt\n", networkName(r.Position.Network))
		} else {
			fmt.Fprintf(&buf, "%s HF: %.2f\n", networkName(r.Position.Network), r.HealthFactor.Value)
		}
	}
	return buf.String()
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
