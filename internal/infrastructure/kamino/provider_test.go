package kamino_test

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/marcus/health-factor-monitor/internal/domain"
	"github.com/marcus/health-factor-monitor/internal/infrastructure/kamino"
)

const (
	// portfolioPath is the Kamino public API route that returns all lending
	// positions for a wallet across every market.
	portfolioPath = "/portfolio/"

	// userAddress is a valid base58 Solana wallet used in test requests.
	userAddress = "11111111111111111111111111111111"
)

// portfolioResponse builds a realistic body for GET /portfolio/{pubkey}. The
// Kamino API returns lending values as decimal strings and does not expose a
// health factor directly; the provider derives it from the position's ltv and
// liquidationLtv.
func portfolioResponse(liquidationLtv, ltv string) string {
	return fmt.Sprintf(`{
  "wallet": %q,
  "lending": [
    {
      "market": "7u3HeHxYDLhnCoErrtycNokbQYbWGzLs6JSDqGAv5PfF",
      "obligation": "6RtT9hBxDQJ5cFCxZcbq7f4fQnCzFvCRK5pNFt9D2mBn",
      "tag": "Vanilla",
      "netValue": "10000.0",
      "totalDepositValue": "20000.0",
      "totalBorrowValue": "10000.0",
      "ltv": %q,
      "maxLtv": "0.75",
      "liquidationLtv": %q,
      "leverage": "1.0",
      "deposits": [],
      "borrows": []
    }
  ]
}`, userAddress, ltv, liquidationLtv)
}

func apiErrorBody(msg string) string {
	return fmt.Sprintf(`{"error": %q}`, msg)
}

func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *kamino.Provider) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv, kamino.NewProvider(srv.URL)
}

func TestGetHealthFactor(t *testing.T) {
	tests := []struct {
		name           string
		liquidationLtv string
		ltv            string
		wantValue      float64
		wantClass      string
	}{
		{
			name:           "safe position",
			liquidationLtv: "0.8",
			ltv:            "0.4",
			wantValue:      2.0,
			wantClass:      domain.ClassificationSafe,
		},
		{
			name:           "warning position",
			liquidationLtv: "0.8",
			ltv:            "0.64",
			wantValue:      1.25,
			wantClass:      domain.ClassificationWarning,
		},
		{
			name:           "critical position",
			liquidationLtv: "0.8",
			ltv:            "0.8",
			wantValue:      1.0,
			wantClass:      domain.ClassificationCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, provider := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(portfolioResponse(tt.liquidationLtv, tt.ltv)))
			})

			got, err := provider.GetHealthFactor(context.Background(), userAddress)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got == nil {
				t.Fatal("expected health factor, got nil")
			}
			if diff := math.Abs(got.Value - tt.wantValue); diff > 1e-9 {
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
		_, _ = w.Write([]byte(portfolioResponse("0.8", "0.4")))
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if _, err := provider.GetHealthFactor(ctx, userAddress); err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestGetHealthFactorErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{
			name:   "api internal error",
			status: http.StatusInternalServerError,
			body:   apiErrorBody("An internal error occurred"),
		},
		{
			name:   "wallet not found",
			status: http.StatusNotFound,
			body:   `{"metadata": "Account could not be found"}`,
		},
		{
			name:   "malformed json",
			status: http.StatusOK,
			body:   "this is not json",
		},
		{
			name:   "no lending positions",
			status: http.StatusOK,
			body:   fmt.Sprintf(`{"wallet": %q, "lending": []}`, userAddress),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, provider := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			})

			if _, err := provider.GetHealthFactor(context.Background(), userAddress); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestGetHealthFactorSkipsSupplyOnlyPosition(t *testing.T) {
	body := fmt.Sprintf(`{
  "wallet": %q,
  "lending": [
    {
      "market": "supply-only-market",
      "obligation": "11111111111111111111111111111111",
      "tag": "Vanilla",
      "netValue": "1000.0",
      "totalDepositValue": "1000.0",
      "totalBorrowValue": "0.0",
      "ltv": "0",
      "maxLtv": "0.75",
      "liquidationLtv": "0.75",
      "leverage": "1.0"
    },
    {
      "market": "borrow-market",
      "obligation": "22222222222222222222222222222222",
      "tag": "Vanilla",
      "netValue": "10000.0",
      "totalDepositValue": "20000.0",
      "totalBorrowValue": "12800.0",
      "ltv": "0.64",
      "maxLtv": "0.75",
      "liquidationLtv": "0.8",
      "leverage": "1.0"
    }
  ]
}`, userAddress)

	_, provider := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	})

	got, err := provider.GetHealthFactor(context.Background(), userAddress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(got.Value-1.25) > 1e-9 {
		t.Errorf("Value = %v, want 1.25", got.Value)
	}
	if got.Classification != domain.ClassificationWarning {
		t.Errorf("Classification = %q, want %q", got.Classification, domain.ClassificationWarning)
	}
}

func TestGetHealthFactorOnlySupplyOnlyPositions(t *testing.T) {
	body := fmt.Sprintf(`{
  "wallet": %q,
  "lending": [
    {
      "market": "supply-only-market",
      "obligation": "11111111111111111111111111111111",
      "tag": "Vanilla",
      "netValue": "1000.0",
      "totalDepositValue": "1000.0",
      "totalBorrowValue": "0.0",
      "ltv": "0",
      "maxLtv": "0.75",
      "liquidationLtv": "0.75",
      "leverage": "1.0"
    }
  ]
}`, userAddress)

	_, provider := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	})

	if _, err := provider.GetHealthFactor(context.Background(), userAddress); err == nil {
		t.Fatal("expected error for supply-only position, got nil")
	}
}

func TestProviderRequestsPortfolioEndpoint(t *testing.T) {
	var capturedPath string

	_, provider := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		_, _ = w.Write([]byte(portfolioResponse("0.8", "0.4")))
	})

	if _, err := provider.GetHealthFactor(context.Background(), userAddress); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(capturedPath, portfolioPath) {
		t.Errorf("path = %q, want prefix %q", capturedPath, portfolioPath)
	}
	if !strings.HasSuffix(capturedPath, userAddress) {
		t.Errorf("path = %q, want suffix %q", capturedPath, userAddress)
	}
}

func TestProviderIdentity(t *testing.T) {
	provider := kamino.NewProvider("https://api.kamino.finance")

	if got := provider.Protocol(); got != domain.ProtocolKamino {
		t.Errorf("Protocol() = %q, want %q", got, domain.ProtocolKamino)
	}
	if got := provider.Network(); got != domain.NetworkSolana {
		t.Errorf("Network() = %q, want %q", got, domain.NetworkSolana)
	}
}

var _ domain.HealthFactorProvider = (*kamino.Provider)(nil)
