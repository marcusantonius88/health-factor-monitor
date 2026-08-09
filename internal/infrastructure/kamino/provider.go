package kamino

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/marcus/health-factor-monitor/internal/domain"
)

const (
	// portfolioPath is the Kamino public API route that returns all lending
	// positions for a wallet across every market.
	portfolioPath = "/portfolio/"

	// apiTimeout bounds each request made by the provider.
	apiTimeout = 30 * time.Second
)

// Provider implements the domain.HealthFactorProvider interface for the Kamino
// protocol, deriving health factors from the user obligation data served by
// the Kamino public API.
type Provider struct {
	baseURL string
	client  *http.Client
}

// NewProvider creates a Kamino provider that talks to the configured API base
// URL (e.g., "https://api.kamino.finance").
func NewProvider(baseURL string) *Provider {
	return &Provider{
		baseURL: baseURL,
		client:  &http.Client{Timeout: apiTimeout},
	}
}

// Protocol returns the provider protocol identifier.
func (p *Provider) Protocol() string { return domain.ProtocolKamino }

// Network returns the provider network identifier.
func (p *Provider) Network() string { return domain.NetworkSolana }

type portfolioResponse struct {
	Wallet  string                     `json:"wallet"`
	Lending []portfolioLendingPosition `json:"lending"`
}

// portfolioLendingPosition represents a single obligation within a wallet's
// portfolio. Financial values (ltv, liquidationLtv) are returned as decimal
// strings and may be null when not applicable.
type portfolioLendingPosition struct {
	Market            string `json:"market"`
	Obligation        string `json:"obligation"`
	Tag               string `json:"tag"`
	NetValue          string `json:"netValue"`
	TotalDepositValue string `json:"totalDepositValue"`
	TotalBorrowValue  string `json:"totalBorrowValue"`
	Ltv               string `json:"ltv"`
	MaxLtv            string `json:"maxLtv"`
	LiquidationLtv    string `json:"liquidationLtv"`
	Leverage          string `json:"leverage"`
}

// GetHealthFactor retrieves the current health factor for the given wallet
// address. It queries the Kamino portfolio endpoint and derives the health
// factor as liquidationLtv / ltv from the obligation data.
func (p *Provider) GetHealthFactor(ctx context.Context, address string) (*domain.HealthFactor, error) {
	url := strings.TrimRight(p.baseURL, "/") + portfolioPath + address

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create portfolio request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get portfolio: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read portfolio response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("portfolio http status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed portfolioResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decode portfolio response: %w", err)
	}

	if len(parsed.Lending) == 0 {
		return nil, errors.New("no lending positions found for address")
	}

	value, err := deriveHealthFactor(parsed.Lending[0])
	if err != nil {
		return nil, err
	}

	return &domain.HealthFactor{
		Value:          value,
		Classification: domain.Classify(value),
	}, nil
}

// deriveHealthFactor computes the health factor for a lending position. The
// Kamino API does not expose the health factor directly; it is derived as
// liquidationLtv / ltv, following the standard lending convention.
func deriveHealthFactor(pos portfolioLendingPosition) (float64, error) {
	ltv, err := parseDecimal(pos.Ltv)
	if err != nil {
		return 0, fmt.Errorf("parse ltv: %w", err)
	}
	liquidationLtv, err := parseDecimal(pos.LiquidationLtv)
	if err != nil {
		return 0, fmt.Errorf("parse liquidation ltv: %w", err)
	}
	if ltv <= 0 {
		return 0, errors.New("ltv must be positive")
	}
	return liquidationLtv / ltv, nil
}

// parseDecimal converts a decimal string (e.g., "0.8") to a float64. An empty
// string (representing a null API field) is an error.
func parseDecimal(s string) (float64, error) {
	if strings.TrimSpace(s) == "" {
		return 0, errors.New("empty decimal value")
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, err
	}
	return v, nil
}
