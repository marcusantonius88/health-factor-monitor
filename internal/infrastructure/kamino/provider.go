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
	rpcURL  string
	client  *http.Client
}

// NewProvider creates a Kamino provider that talks to the configured API base
// URL (e.g., "https://api.kamino.finance") and optionally uses a Solana RPC
// endpoint for snapshot freshness reads.
func NewProvider(baseURL string, rpcURL ...string) *Provider {
	rpcEndpoint := ""
	if len(rpcURL) > 0 {
		rpcEndpoint = rpcURL[0]
	}
	return &Provider{
		baseURL: baseURL,
		rpcURL:  rpcEndpoint,
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
	Market               string             `json:"market"`
	Obligation           string             `json:"obligation"`
	Tag                  string             `json:"tag"`
	NetValue             string             `json:"netValue"`
	TotalDepositValue    string             `json:"totalDepositValue"`
	TotalBorrowValue     string             `json:"totalBorrowValue"`
	Ltv                  string             `json:"ltv"`
	MaxLtv               string             `json:"maxLtv"`
	LiquidationLtv       string             `json:"liquidationLtv"`
	Leverage             string             `json:"leverage"`
	AppliedSlot          string             `json:"appliedSlot"`
	ActiveElevationGroup string             `json:"activeElevationGroup"`
	Deposits             []portfolioDeposit `json:"deposits"`
	Borrows              []portfolioBorrow  `json:"borrows"`
}

type portfolioDeposit struct {
	Mint                 string `json:"mint"`
	Reserve              string `json:"reserve"`
	Symbol               string `json:"symbol"`
	LiquidationLtv       string `json:"liquidationLtv"`
	ActiveElevationGroup string `json:"activeElevationGroup"`
}

type portfolioBorrow struct {
	Mint         string `json:"mint"`
	Value        string `json:"value"`
	BorrowFactor string `json:"borrowFactor"`
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

	// A wallet may hold several obligations; pick the first with an active
	// borrow (ltv > 0). Supply-only positions have ltv 0 and must be skipped.
	for _, position := range parsed.Lending {
		if !hasActiveBorrow(position) {
			continue
		}
		value, err := deriveHealthFactor(position)
		if err != nil {
			return nil, err
		}
		return &domain.HealthFactor{
			Value:          value,
			Classification: domain.Classify(value),
		}, nil
	}

	return nil, errors.New("no active borrow position found for address")
}

// GetPositionSnapshot retrieves the same active-borrow obligation used by
// GetHealthFactor and adds only estimate-specific Kamino data when the position
// is below the fixed safe threshold.
func (p *Provider) GetPositionSnapshot(ctx context.Context, address string) (*domain.PositionSnapshot, error) {
	hf, err := p.GetHealthFactor(ctx, address)
	if err != nil {
		return nil, err
	}
	position := domain.LendingPosition{
		Wallet:   domain.Wallet{Address: address},
		Protocol: domain.ProtocolKamino,
		Network:  domain.NetworkSolana,
	}
	snapshot := &domain.PositionSnapshot{
		Position:     position,
		HealthFactor: hf,
	}
	if hf.Value >= domain.TargetHealthFactor {
		return snapshot, nil
	}

	slot, err := p.getCurrentSlot(ctx)
	if err != nil {
		snapshot.ContributionUnavailableReason = "slot Solana indisponível"
		return snapshot, nil
	}
	_, positionData, err := p.getRefreshedLendingPosition(ctx, address, slot)
	if err != nil {
		snapshot.ContributionUnavailableReason = "dados da API Kamino indisponíveis ou incompletos"
		return snapshot, nil
	}
	if positionData.AppliedSlot == "" {
		snapshot.ContributionUnavailableReason = "appliedSlot Kamino indisponível"
		return snapshot, nil
	}
	appliedSlot, err := strconv.ParseUint(positionData.AppliedSlot, 10, 64)
	if err != nil || appliedSlot < slot {
		snapshot.ContributionUnavailableReason = "portfólio Kamino não alcançou o slot solicitado"
		return snapshot, nil
	}

	solPrice, err := p.getSOLOracleAtMarket(ctx, positionData.Market)
	if err != nil {
		snapshot.ContributionUnavailableReason = "cotação SOL Kamino indisponível ou desatualizada"
		return snapshot, nil
	}
	basis, err := buildSOLContributionBasis(positionData, hf, solPrice)
	if err != nil {
		snapshot.ContributionUnavailableReason = err.Error()
		return snapshot, nil
	}
	snapshot.ContributionBasis = basis
	return snapshot, nil
}

func (p *Provider) getCurrentSlot(ctx context.Context) (uint64, error) {
	if strings.TrimSpace(p.rpcURL) == "" {
		return 0, errors.New("Solana RPC endpoint is not configured")
	}
	body := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"getSlot","params":[]}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.rpcURL, body)
	if err != nil {
		return 0, fmt.Errorf("create slot request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("get Solana slot: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("read Solana slot response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("Solana slot http status %d", resp.StatusCode)
	}
	var parsed struct {
		Result uint64 `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return 0, fmt.Errorf("decode Solana slot response: %w", err)
	}
	if parsed.Error != nil {
		return 0, fmt.Errorf("Solana RPC error: %s", parsed.Error.Message)
	}
	if parsed.Result == 0 {
		return 0, errors.New("Solana slot is zero")
	}
	return parsed.Result, nil
}

func (p *Provider) getRefreshedLendingPosition(
	ctx context.Context,
	address string,
	slot uint64,
) (*portfolioResponse, *portfolioLendingPosition, error) {
	url := fmt.Sprintf(
		"%s%s%s?sections=lending&minSlot=%d",
		strings.TrimRight(p.baseURL, "/"),
		portfolioPath,
		address,
		slot,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("create refreshed portfolio request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("get refreshed portfolio: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read refreshed portfolio: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("refreshed portfolio http status %d", resp.StatusCode)
	}
	var parsed portfolioResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, nil, fmt.Errorf("decode refreshed portfolio: %w", err)
	}
	for i := range parsed.Lending {
		position := &parsed.Lending[i]
		if hasActiveBorrow(*position) {
			return &parsed, position, nil
		}
	}
	return nil, nil, errors.New("no active borrow position found")
}

func (p *Provider) getSOLOracleAtMarket(ctx context.Context, market string) (*float64, error) {
	if strings.TrimSpace(market) == "" {
		return nil, errors.New("Kamino market is missing")
	}
	url := strings.TrimRight(p.baseURL, "/") + "/oracles/prices?markets=all"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create SOL oracle request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get SOL oracle: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read SOL oracle: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SOL oracle http status %d", resp.StatusCode)
	}

	const solMint = "So11111111111111111111111111111111111111112"
	var parsed struct {
		Markets []struct {
			LendingMarket string `json:"lendingMarket"`
			Prices        []struct {
				Mint          string `json:"mint"`
				Price         string `json:"price"`
				Timestamp     string `json:"timestamp"`
				MaxAgeSeconds int64  `json:"maxAgeInSeconds"`
			} `json:"prices"`
		} `json:"markets"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("decode SOL oracle: %w", err)
	}
	for _, marketData := range parsed.Markets {
		if marketData.LendingMarket != market {
			continue
		}
		for _, quote := range marketData.Prices {
			if quote.Mint != solMint {
				continue
			}
			price, err := parseDecimal(quote.Price)
			if err != nil || price <= 0 {
				return nil, errors.New("invalid SOL oracle price")
			}
			quoteTime, err := time.Parse(time.RFC3339, quote.Timestamp)
			if err != nil || quote.MaxAgeSeconds <= 0 {
				return nil, errors.New("invalid SOL oracle freshness")
			}
			age := time.Since(quoteTime)
			if age < -5*time.Minute || age > time.Duration(quote.MaxAgeSeconds)*time.Second {
				return nil, errors.New("stale SOL oracle price")
			}
			return &price, nil
		}
	}
	return nil, errors.New("SOL oracle quote not found for market")
}

func buildSOLContributionBasis(
	position *portfolioLendingPosition,
	hf *domain.HealthFactor,
	solPrice *float64,
) (*domain.ContributionBasis, error) {
	if position == nil || hf == nil || solPrice == nil {
		return nil, errors.New("dados de contribuição SOL incompletos")
	}
	if strings.TrimSpace(position.ActiveElevationGroup) == "" {
		return nil, errors.New("grupo de elevação ativo Kamino não confirmado")
	}

	const solMint = "So11111111111111111111111111111111111111112"
	var solDeposit *portfolioDeposit
	for i := range position.Deposits {
		deposit := &position.Deposits[i]
		if deposit.Mint == solMint &&
			strings.EqualFold(deposit.Symbol, "SOL") &&
			deposit.ActiveElevationGroup == position.ActiveElevationGroup {
			solDeposit = deposit
			break
		}
	}
	if solDeposit == nil {
		return nil, errors.New("reserva SOL elegível não confirmada")
	}
	liquidationLtv, err := parseDecimal(solDeposit.LiquidationLtv)
	if err != nil || liquidationLtv <= 0 {
		return nil, errors.New("liquidação SOL Kamino não confirmada")
	}

	effectiveDebt := 0.0
	if len(position.Borrows) == 0 {
		effectiveDebt, err = parseDecimal(position.TotalBorrowValue)
	} else {
		for _, borrow := range position.Borrows {
			value, err := parseDecimal(borrow.Value)
			if err != nil || value < 0 {
				return nil, errors.New("valor de dívida Kamino incompleto")
			}
			factor := 1.0
			if strings.TrimSpace(borrow.BorrowFactor) != "" {
				factor, err = parseDecimal(borrow.BorrowFactor)
				if err != nil || factor <= 0 {
					return nil, errors.New("borrow factor Kamino inválido")
				}
			}
			effectiveDebt += value * factor
		}
	}
	if err != nil || effectiveDebt <= 0 || *solPrice <= 0 {
		return nil, errors.New("dados de dívida ou preço SOL inválidos")
	}

	basis := &domain.ContributionBasis{
		CurrentHealthFactor:          hf.Value,
		EffectiveDebtUSD:             effectiveDebt,
		IncrementalLiquidationFactor: liquidationLtv,
		Asset:                        domain.ContributionAsset{Symbol: "SOL", Identifier: solMint},
		PriceUSD:                     *solPrice,
		AssetPrecision:               9,
		PriceFreshnessValidated:      true,
	}
	if err := basis.Validate(); err != nil {
		return nil, err
	}
	return basis, nil
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

// hasActiveBorrow reports whether the lending position carries debt, i.e. it
// has a positive ltv. Supply-only positions expose ltv 0 and are not borrows.
func hasActiveBorrow(pos portfolioLendingPosition) bool {
	ltv, err := parseDecimal(pos.Ltv)
	if err != nil {
		return false
	}
	return ltv > 0
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
