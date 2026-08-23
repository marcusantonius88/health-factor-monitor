package aave_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/marcus/health-factor-monitor/internal/domain"
	"github.com/marcus/health-factor-monitor/internal/infrastructure/aave"
)

const (
	v3PoolContract     = "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2"
	v2PoolContract     = "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9"
	baseV3PoolContract = "0xA238Dd80C259a72e81d7e4664a9801593F98d1c5"
	userAddress        = "0x0000000000000000000000000000000000000001"
	selector           = "0xbf92857c"
)

func hexUint256(t *testing.T, v int64) string {
	t.Helper()
	return fmt.Sprintf("%064x", v)
}

func rpcResult(result string) string {
	return fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"result":%q}`, result)
}

func rpcError(msg string) string {
	return fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":%q}}`, msg)
}

// getUserAccountDataResult builds the ABI-encoded result of the 6-tuple
// returned by getUserAccountData, with the given healthFactor scaled by 1e18.
func getUserAccountDataResult(t *testing.T, healthFactor int64) string {
	t.Helper()
	return "0x" +
		hexUint256(t, 1000000000000000000) + // totalCollateralBase (1e18)
		hexUint256(t, 0) + // totalDebtBase
		hexUint256(t, 1000000000000000000) + // availableBorrowsBase (1e18)
		hexUint256(t, 8000000000000000000) + // currentLiquidationThreshold (0.8e18)
		hexUint256(t, 5000000000000000000) + // ltv (0.5e18)
		hexUint256(t, healthFactor) // healthFactor
}

func getUserAccountDataResultFromWords(
	t *testing.T,
	totalCollateralBase, totalDebtBase, availableBorrowsBase,
	currentLiquidationThreshold, ltv, healthFactor *big.Int,
) string {
	t.Helper()

	words := []*big.Int{
		totalCollateralBase,
		totalDebtBase,
		availableBorrowsBase,
		currentLiquidationThreshold,
		ltv,
		healthFactor,
	}

	encoded := "0x"
	for _, word := range words {
		if word == nil {
			word = big.NewInt(0)
		}
		encoded += fmt.Sprintf("%064x", word)
	}
	return encoded
}

func maxUint256(t *testing.T) *big.Int {
	t.Helper()

	decoded, err := hex.DecodeString(strings.Repeat("f", 64))
	if err != nil {
		t.Fatalf("decode max uint256: %v", err)
	}
	return new(big.Int).SetBytes(decoded)
}

func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *aave.Provider) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv, aave.NewProvider(srv.URL, domain.NetworkEthereum)
}

func TestGetHealthFactor(t *testing.T) {
	tests := []struct {
		name         string
		healthFactor int64
		wantValue    float64
		wantClass    string
	}{
		{
			name:         "safe position",
			healthFactor: 2500000000000000000, // 2.5e18
			wantValue:    2.5,
			wantClass:    domain.ClassificationSafe,
		},
		{
			name:         "warning position",
			healthFactor: 1200000000000000000, // 1.2e18
			wantValue:    1.2,
			wantClass:    domain.ClassificationWarning,
		},
		{
			name:         "critical position",
			healthFactor: 900000000000000000, // 0.9e18
			wantValue:    0.9,
			wantClass:    domain.ClassificationCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, provider := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(rpcResult(getUserAccountDataResult(t, tt.healthFactor))))
			})

			got, err := provider.GetHealthFactor(context.Background(), userAddress)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got == nil {
				t.Fatal("expected health factor, got nil")
			}
			if got.Value != tt.wantValue {
				t.Errorf("Value = %v, want %v", got.Value, tt.wantValue)
			}
			if got.Classification != tt.wantClass {
				t.Errorf("Classification = %q, want %q", got.Classification, tt.wantClass)
			}
		})
	}
}

func TestGetHealthFactorTimeout(t *testing.T) {
	_, provider := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		_, _ = w.Write([]byte(rpcResult(getUserAccountDataResult(t, 2500000000000000000))))
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if _, err := provider.GetHealthFactor(ctx, userAddress); err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestGetHealthFactorMalformedResponses(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "invalid hex result",
			body: rpcResult("0xzzzz"),
		},
		{
			name: "empty result",
			body: rpcResult(""),
		},
		{
			name: "malformed json",
			body: "this is not json",
		},
		{
			name: "rpc error response",
			body: rpcError("execution reverted"),
		},
		{
			name: "result too short",
			body: rpcResult("0x" + strings.Repeat("0", 64)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, provider := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(tt.body))
			})

			if _, err := provider.GetHealthFactor(context.Background(), userAddress); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestProviderSendsEthCallToPool(t *testing.T) {
	var captured struct {
		method string
		to     string
		data   string
	}

	_, provider := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Method string `json:"method"`
			Params []struct {
				To   string `json:"to"`
				Data string `json:"data"`
			} `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request: %v", err)
			return
		}
		if len(body.Params) > 0 {
			captured.method = body.Method
			captured.to = body.Params[0].To
			captured.data = body.Params[0].Data
		}
		_, _ = w.Write([]byte(rpcResult(getUserAccountDataResult(t, 2500000000000000000))))
	})

	if _, err := provider.GetHealthFactor(context.Background(), userAddress); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if captured.method != "eth_call" {
		t.Errorf("method = %q, want %q", captured.method, "eth_call")
	}
	if !strings.EqualFold(captured.to, v3PoolContract) {
		t.Errorf("to = %q, want %q", captured.to, v3PoolContract)
	}
	if !strings.HasPrefix(captured.data, selector) {
		t.Errorf("data = %q, want prefix %q", captured.data, selector)
	}
	if !strings.Contains(captured.data, strings.TrimPrefix(userAddress, "0x")) {
		t.Errorf("data = %q, want to contain user address %q", captured.data, userAddress)
	}
}

func TestGetHealthFactorFallsBackToV2WhenV3IsInfinite(t *testing.T) {
	infiniteHF := maxUint256(t)
	v2HF := big.NewInt(1950000000000000000)

	_, provider := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Params []struct {
				To string `json:"to"`
			} `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if len(body.Params) == 0 {
			t.Fatalf("missing params in request")
		}

		result := ""
		switch {
		case strings.EqualFold(body.Params[0].To, v3PoolContract):
			result = getUserAccountDataResultFromWords(
				t,
				big.NewInt(0),
				big.NewInt(0),
				big.NewInt(0),
				big.NewInt(0),
				big.NewInt(0),
				infiniteHF,
			)
		case strings.EqualFold(body.Params[0].To, v2PoolContract):
			result = getUserAccountDataResultFromWords(
				t,
				big.NewInt(3000000000000000000),
				big.NewInt(1000000000000000000),
				big.NewInt(1000000000000000000),
				big.NewInt(8000000000000000000),
				big.NewInt(5000000000000000000),
				v2HF,
			)
		default:
			t.Fatalf("unexpected pool address: %s", body.Params[0].To)
		}

		_, _ = w.Write([]byte(rpcResult(result)))
	})

	got, err := provider.GetHealthFactor(context.Background(), userAddress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected health factor, got nil")
	}
	if got.Value != 1.95 {
		t.Fatalf("Value = %v, want 1.95", got.Value)
	}
	if got.Classification != domain.ClassificationSafe {
		t.Fatalf("Classification = %q, want %q", got.Classification, domain.ClassificationSafe)
	}
}

func TestProviderIdentity(t *testing.T) {
	provider := aave.NewProvider("http://localhost:8545", domain.NetworkEthereum)

	if got := provider.Protocol(); got != domain.ProtocolAave {
		t.Errorf("Protocol() = %q, want %q", got, domain.ProtocolAave)
	}
	if got := provider.Network(); got != domain.NetworkEthereum {
		t.Errorf("Network() = %q, want %q", got, domain.NetworkEthereum)
	}
}

func TestProviderUsesBasePoolForBaseNetwork(t *testing.T) {
	var capturedTo string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Params []struct {
				To string `json:"to"`
			} `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if len(body.Params) == 0 {
			t.Fatal("expected params in request")
		}
		capturedTo = body.Params[0].To
		_, _ = w.Write([]byte(rpcResult(getUserAccountDataResult(t, 1950000000000000000))))
	}))
	defer srv.Close()

	provider := aave.NewProvider(srv.URL, domain.NetworkBase)
	got, err := provider.GetHealthFactor(context.Background(), userAddress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || got.Value != 1.95 {
		t.Fatalf("Value = %+v, want 1.95", got)
	}
	if !strings.EqualFold(capturedTo, baseV3PoolContract) {
		t.Fatalf("to = %q, want %q", capturedTo, baseV3PoolContract)
	}
}

var _ domain.HealthFactorProvider = (*aave.Provider)(nil)
