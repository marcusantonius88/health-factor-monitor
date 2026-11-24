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

var _ domain.ContributionSnapshotProvider = (*aave.Provider)(nil)

func TestGetPositionSnapshotBaseBTC(t *testing.T) {
	const (
		cbBTCAddress = "0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf"
		lBTCAddress  = "0xecAc9C5f704e954931349Da37F60E39f515c11c1"
	)

	var ethCalls []struct {
		to       string
		blockTag string
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		switch body.Method {
		case "eth_blockNumber":
			_, _ = w.Write([]byte(rpcResult(hexUint256(t, 123456))))
		case "eth_call":
			if len(body.Params) == 0 {
				t.Fatal("missing eth_call params")
			}
			var callParam struct {
				To   string `json:"to"`
				Data string `json:"data"`
			}
			if err := json.Unmarshal(body.Params[0], &callParam); err != nil {
				t.Fatalf("decode eth_call param: %v", err)
			}
			blockTag := ""
			if len(body.Params) > 1 {
				if err := json.Unmarshal(body.Params[1], &blockTag); err != nil {
					t.Fatalf("decode eth_call block tag: %v", err)
				}
			}
			ethCalls = append(ethCalls, struct {
				to       string
				blockTag string
			}{to: callParam.To, blockTag: blockTag})
			switch {
			case strings.HasPrefix(callParam.Data, "0xd1946f7a"):
				offset := fmt.Sprintf("%064x", 32)
				length := fmt.Sprintf("%064x", 1)
				addressBytes, err := hex.DecodeString(strings.TrimPrefix(cbBTCAddress, "0x"))
				if err != nil {
					t.Fatalf("decode BTC address: %v", err)
				}
				addressWord := fmt.Sprintf("%064x", new(big.Int).SetBytes(addressBytes))
				_, _ = w.Write([]byte(rpcResult("0x" + offset + length + addressWord)))
			case strings.HasPrefix(callParam.Data, "0x35ea6a75"):
				config := new(big.Int).Lsh(big.NewInt(8200), 16)
				_, _ = w.Write([]byte(rpcResult("0x" + fmt.Sprintf("%064x", config))))
			case strings.HasPrefix(callParam.Data, "0x3a66f3b7"):
				words := strings.Repeat("0", 64*7) + fmt.Sprintf("%064x", 1)
				_, _ = w.Write([]byte(rpcResult("0x" + words)))
			case strings.HasPrefix(callParam.Data, "0x99a88ec4"):
				_, _ = w.Write([]byte(rpcResult("0x" + fmt.Sprintf("%064x", 8500000000000))))
			case strings.HasPrefix(callParam.Data, "0xbf92857c"):
				account := getUserAccountDataResultFromWords(
					t,
					big.NewInt(10000000000000),
					big.NewInt(10000000000000),
					big.NewInt(1000000000000),
					big.NewInt(8000),
					big.NewInt(5000),
					big.NewInt(1320000000000000000),
				)
				_, _ = w.Write([]byte(rpcResult(account)))
			default:
				_, _ = w.Write([]byte(rpcResult(getUserAccountDataResult(t, 1320000000000000000))))
			}
		default:
			t.Fatalf("unexpected RPC method: %s", body.Method)
		}
	}))
	defer srv.Close()

	provider := aave.NewProvider(srv.URL, domain.NetworkBase)
	snapshot, err := provider.GetPositionSnapshot(context.Background(), userAddress)
	if err != nil {
		t.Fatalf("GetPositionSnapshot() error = %v", err)
	}
	if snapshot == nil {
		t.Fatal("GetPositionSnapshot() returned nil snapshot")
	}
	if snapshot.HealthFactor == nil || snapshot.HealthFactor.Value != 1.32 {
		t.Fatalf("HealthFactor = %+v, want 1.32", snapshot.HealthFactor)
	}
	if snapshot.ContributionBasis == nil {
		t.Fatal("expected BTC contribution basis, got unavailable snapshot")
	}
	if snapshot.ContributionBasis.Asset.Symbol != "BTC" {
		t.Errorf("Asset.Symbol = %q, want BTC", snapshot.ContributionBasis.Asset.Symbol)
	}
	if snapshot.ContributionBasis.Asset.Identifier == "" {
		t.Fatal("BTC contribution asset identifier must not be empty")
	}
	if !snapshot.ContributionBasis.PriceFreshnessValidated {
		t.Fatal("BTC oracle freshness must be validated")
	}
	if len(ethCalls) < 2 {
		t.Fatalf("eth_call count = %d, want account, reserve, oracle, and eMode reads", len(ethCalls))
	}

	firstBlockTag := ""
	for _, call := range ethCalls {
		if call.blockTag != "" {
			firstBlockTag = call.blockTag
			break
		}
	}
	if firstBlockTag == "" {
		t.Fatal("snapshot reads must pin a block")
	}
	for i, call := range ethCalls {
		if call.blockTag == "" {
			continue
		}
		if call.blockTag != firstBlockTag {
			t.Errorf("eth_call %d block tag = %q, want pinned block %q", i, call.blockTag, firstBlockTag)
		}
	}

	identifier := snapshot.ContributionBasis.Asset.Identifier
	if !strings.EqualFold(identifier, cbBTCAddress) && !strings.EqualFold(identifier, lBTCAddress) {
		t.Errorf("BTC asset identifier = %q, want live eligible BTC reserve", identifier)
	}
}

func TestGetPositionSnapshotSafeBaseSkipsEstimateReads(t *testing.T) {
	rpcCalls := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rpcCalls++
		var body struct {
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body.Method != "eth_call" {
			t.Fatalf("safe snapshot should not read estimate-only RPC data; method = %s", body.Method)
		}
		_, _ = w.Write([]byte(rpcResult(getUserAccountDataResult(t, 1500000000000000000))))
	}))
	defer srv.Close()

	provider := aave.NewProvider(srv.URL, domain.NetworkBase)
	snapshot, err := provider.GetPositionSnapshot(context.Background(), userAddress)
	if err != nil {
		t.Fatalf("GetPositionSnapshot() error = %v", err)
	}
	if snapshot == nil || snapshot.HealthFactor == nil {
		t.Fatal("expected successful safe snapshot")
	}
	if snapshot.HealthFactor.Value != 1.5 {
		t.Errorf("HealthFactor.Value = %v, want 1.5", snapshot.HealthFactor.Value)
	}
	if snapshot.ContributionBasis != nil || snapshot.ContributionUnavailableReason != "" {
		t.Fatalf("safe snapshot must not contain estimate data: %+v", snapshot)
	}
	if rpcCalls != 1 {
		t.Errorf("RPC call count = %d, want 1 account-data read", rpcCalls)
	}
}

func TestGetPositionSnapshotRejectsUneligibleBTCData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body.Method == "eth_call" {
			_, _ = w.Write([]byte(rpcResult(getUserAccountDataResult(t, 1320000000000000000))))
			return
		}
		_, _ = w.Write([]byte(rpcResult(hexUint256(t, 123456))))
	}))
	defer srv.Close()

	provider := aave.NewProvider(srv.URL, domain.NetworkBase)
	snapshot, err := provider.GetPositionSnapshot(context.Background(), userAddress)
	if err != nil {
		t.Fatalf("GetPositionSnapshot() error = %v", err)
	}
	if snapshot == nil || snapshot.HealthFactor == nil {
		t.Fatal("estimate-only failure must preserve the Health Factor")
	}
	if snapshot.HealthFactor.Value != 1.32 {
		t.Errorf("HealthFactor.Value = %v, want 1.32", snapshot.HealthFactor.Value)
	}
	if snapshot.ContributionBasis != nil {
		t.Fatal("unverified or ineligible BTC data must not produce a contribution basis")
	}
	if strings.TrimSpace(snapshot.ContributionUnavailableReason) == "" {
		t.Fatal("unavailable BTC estimate reason must be explicit")
	}
}

func TestGetPositionSnapshotAaveEstimateFailures(t *testing.T) {
	tests := []struct {
		name string
		mode string
	}{
		{name: "missing BTC reserve", mode: "missing-reserve"},
		{name: "ineligible BTC reserve", mode: "ineligible-reserve"},
		{name: "missing BTC price", mode: "missing-price"},
		{name: "zero BTC price", mode: "zero-price"},
		{name: "unsupported oracle source", mode: "unsupported-oracle"},
		{name: "unverifiable oracle freshness", mode: "stale-oracle"},
		{name: "invalid USD base unit", mode: "invalid-base-unit"},
		{name: "unavailable reserve configuration", mode: "missing-reserve-config"},
		{name: "unavailable eMode or collateral data", mode: "missing-emode"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Method string            `json:"method"`
					Params []json.RawMessage `json:"params"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatalf("decode request: %v", err)
				}

				switch body.Method {
				case "eth_blockNumber":
					_, _ = w.Write([]byte(rpcResult(hexUint256(t, 123456))))
				case "eth_call":
					if len(body.Params) == 0 {
						t.Fatal("missing eth_call params")
					}
					var callParam struct {
						To   string `json:"to"`
						Data string `json:"data"`
					}
					if err := json.Unmarshal(body.Params[0], &callParam); err != nil {
						t.Fatalf("decode eth_call param: %v", err)
					}

					switch {
					case strings.HasPrefix(callParam.Data, "0xbf92857c"):
						_, _ = w.Write([]byte(rpcResult(getUserAccountDataResult(t, 1320000000000000000))))
					case strings.HasPrefix(callParam.Data, "0xd1946f7a"):
						if tt.mode == "missing-reserve" {
							_, _ = w.Write([]byte(rpcResult("0x" + fmt.Sprintf("%064x", 32) + fmt.Sprintf("%064x", 0))))
						} else if tt.mode == "ineligible-reserve" {
							addressBytes, _ := hex.DecodeString(strings.Repeat("0", 38) + "1234")
							addressWord := fmt.Sprintf("%064x", new(big.Int).SetBytes(addressBytes))
							_, _ = w.Write([]byte(rpcResult("0x" + fmt.Sprintf("%064x", 32) + fmt.Sprintf("%064x", 1) + addressWord)))
						} else {
							addressBytes, _ := hex.DecodeString(strings.TrimPrefix("0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf", "0x"))
							addressWord := fmt.Sprintf("%064x", new(big.Int).SetBytes(addressBytes))
							_, _ = w.Write([]byte(rpcResult("0x" + fmt.Sprintf("%064x", 32) + fmt.Sprintf("%064x", 1) + addressWord)))
						}
					case strings.HasPrefix(callParam.Data, "0x35ea6a75"):
						if tt.mode == "missing-reserve-config" {
							_, _ = w.Write([]byte(rpcError("reserve config unavailable")))
						} else {
							config := new(big.Int).Lsh(big.NewInt(8200), 16)
							_, _ = w.Write([]byte(rpcResult("0x" + fmt.Sprintf("%064x", config))))
						}
					case strings.HasPrefix(callParam.Data, "0x3a66f3b7"):
						if tt.mode == "missing-emode" || tt.mode == "ineligible-reserve" {
							_, _ = w.Write([]byte(rpcError("eMode data unavailable")))
						} else {
							words := strings.Repeat("0", 64*7) + fmt.Sprintf("%064x", 1)
							_, _ = w.Write([]byte(rpcResult("0x" + words)))
						}
					case strings.HasPrefix(callParam.Data, "0x99a88ec4"):
						switch tt.mode {
						case "missing-price":
							_, _ = w.Write([]byte(rpcResult("0x")))
						case "zero-price":
							_, _ = w.Write([]byte(rpcResult("0x" + fmt.Sprintf("%064x", 0))))
						case "unsupported-oracle", "stale-oracle":
							_, _ = w.Write([]byte(rpcResult("0xzzzz")))
						case "invalid-base-unit":
							_, _ = w.Write([]byte(rpcResult("0x" + fmt.Sprintf("%064x", 1000000000000))))
						default:
							_, _ = w.Write([]byte(rpcResult("0x" + fmt.Sprintf("%064x", 8500000000000))))
						}
					default:
						_, _ = w.Write([]byte(rpcResult(getUserAccountDataResult(t, 1320000000000000000))))
					}
				default:
					t.Fatalf("unexpected RPC method: %s", body.Method)
				}
			}))
			defer srv.Close()

			provider := aave.NewProvider(srv.URL, domain.NetworkBase)
			snapshot, err := provider.GetPositionSnapshot(context.Background(), userAddress)
			if err != nil {
				t.Fatalf("GetPositionSnapshot() error = %v", err)
			}
			if snapshot == nil || snapshot.HealthFactor == nil {
				t.Fatal("estimate-only Aave failure must preserve the Health Factor")
			}
			if snapshot.HealthFactor.Value != 1.32 {
				t.Errorf("HealthFactor.Value = %v, want 1.32", snapshot.HealthFactor.Value)
			}
			if snapshot.ContributionBasis != nil {
				t.Fatalf("invalid Aave estimate data produced a basis: %+v", snapshot.ContributionBasis)
			}
			if strings.TrimSpace(snapshot.ContributionUnavailableReason) == "" {
				t.Fatal("Aave estimate failure must include an unavailable reason")
			}
		})
	}
}
