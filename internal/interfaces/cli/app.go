package cli

import (
	"fmt"
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
	
	for _, r := range results {
		if r.Error != "" {
			fmt.Fprintf(&buf, "%s HF: unavailable\n", networkName(r.Position.Network))
		} else if r.HealthFactor == nil {
			fmt.Fprintf(&buf, "%s:\t%s\n", networkName(r.Position.Network), domain.EmojiForHealthFactor(0))
		} else if r.HealthFactor.Value > 1e50 {
			fmt.Fprintf(&buf, "%s:\t%s no active debt\n", networkName(r.Position.Network), domain.EmojiForHealthFactor(r.HealthFactor.Value))
		} else {
			fmt.Fprintf(&buf, "%s:\t%s %.2f\n", networkName(r.Position.Network), domain.EmojiForHealthFactor(r.HealthFactor.Value), r.HealthFactor.Value)
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
