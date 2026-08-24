package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/marcus/health-factor-monitor/internal/application"
	"github.com/marcus/health-factor-monitor/internal/domain"
	"github.com/marcus/health-factor-monitor/internal/infrastructure/aave"
	"github.com/marcus/health-factor-monitor/internal/infrastructure/config"
	"github.com/marcus/health-factor-monitor/internal/infrastructure/kamino"
	"github.com/marcus/health-factor-monitor/internal/interfaces/cli"
)

func TestE2EHealthCheckFlow(t *testing.T) {
	cfg, err := config.NewReader("testdata/valid-config.json").Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	aaveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req["id"],
			"result":  buildAaveResultHex(2.5),
		}); err != nil {
			t.Fatalf("encode aave response: %v", err)
		}
	}))
	defer aaveServer.Close()

	kaminoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"wallet": "11111111111111111111111111111111",
			"lending": []map[string]string{{
				"market":            "mainnet",
				"obligation":        "obligation-1",
				"tag":               "borrow",
				"netValue":          "50.00",
				"totalDepositValue": "100.00",
				"totalBorrowValue":  "50.00",
				"ltv":               "0.5",
				"maxLtv":            "0.8",
				"liquidationLtv":    "1.5",
				"leverage":          "1.0",
			}},
		}); err != nil {
			t.Fatalf("encode kamino response: %v", err)
		}
	}))
	defer kaminoServer.Close()

	cfg.RPCEndpoints[domain.NetworkEthereum] = aaveServer.URL
	cfg.RPCEndpoints[domain.NetworkSolana] = kaminoServer.URL

	providers := map[string]domain.HealthFactorProvider{
		"aave:ethereum": aave.NewProvider(aaveServer.URL, domain.NetworkEthereum),
		"kamino:solana": kamino.NewProvider(kaminoServer.URL),
	}

	svc := application.NewCheckService(cfg, providers)
	results := svc.CheckAll(context.Background())

	if len(results) != len(cfg.Positions) {
		t.Fatalf("got %d results, want %d", len(results), len(cfg.Positions))
	}

	formatted := cli.FormatResults(results)
	// Novo formato: Ethereum HF: 2.50 / Solana HF: 3.00
	if !strings.Contains(formatted, "Ethereum:	🟩 2.50") {
		t.Fatalf("formatted output missing Ethereum HF: %s", formatted)
	}
	if !strings.Contains(formatted, "Solana:	🟩 3.00") {
		t.Fatalf("formatted output missing Solana HF: %s", formatted)
	}

	for _, result := range results {
		if result.Error != "" {
			t.Fatalf("unexpected error for %s: %s", result.Position.Wallet.Alias, result.Error)
		}
		if result.HealthFactor == nil {
			t.Fatalf("expected health factor for %s", result.Position.Wallet.Alias)
		}
	}
}

func buildAaveResultHex(value float64) string {
	scaled := new(big.Float).Mul(big.NewFloat(value), big.NewFloat(1e18))
	intValue, _ := scaled.Int(nil)

	words := make([]string, 6)
	for i := 0; i < 5; i++ {
		words[i] = strings.Repeat("0", 64)
	}
	words[5] = fmt.Sprintf("%064x", intValue)
	return "0x" + strings.Join(words, "")
}
