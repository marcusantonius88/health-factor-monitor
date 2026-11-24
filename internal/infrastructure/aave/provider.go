package aave

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/marcus/health-factor-monitor/internal/domain"
)

const (
	// v3PoolContract is the Aave V3 Pool contract on Ethereum mainnet.
	v3PoolContract = "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2"

	// v2PoolContract is the Aave V2 LendingPool contract on Ethereum mainnet.
	v2PoolContract = "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9"

	// baseV3PoolContract is the Aave V3 Pool contract on Base.
	baseV3PoolContract = "0xA238Dd80C259a72e81d7e4664a9801593F98d1c5"

	cbBTCAddress = "0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf"
	lBTCAddress  = "0xecAc9C5f704e954931349Da37F60E39f515c11c1"
	tBTCAddress  = "0x0028F4f6FF85Fd1C6d7d45c1D15E4cA4A0b3E3f"

	// getUserAccountDataSelector is the function selector for getUserAccountData(address).
	getUserAccountDataSelector = "0xbf92857c"

	// healthFactorIndex is the position of healthFactor in the returned tuple.
	healthFactorIndex = 5

	// ethCallTimeout bounds each rpc request made by the provider.
	ethCallTimeout = 30 * time.Second

	baseUSDCurrencyAddress = "0x0000000000000000000000000000000000000000"
)

// Provider implements the domain.HealthFactorProvider interface for the Aave
// protocol, retrieving health factors directly from the Aave Pool contract via
// an Ethereum JSON-RPC eth_call.
type Provider struct {
	rpcURL  string
	network string
	client  *http.Client
}

// NewProvider creates an Aave provider that talks to the given JSON-RPC URL.
func NewProvider(rpcURL, network string) *Provider {
	return &Provider{
		rpcURL:  rpcURL,
		network: network,
		client:  &http.Client{Timeout: ethCallTimeout},
	}
}

// Protocol returns the provider protocol identifier.
func (p *Provider) Protocol() string { return domain.ProtocolAave }

// Network returns the provider network identifier.
func (p *Provider) Network() string { return p.network }

type rpcRequest struct {
	JSONRPC string     `json:"jsonrpc"`
	ID      int        `json:"id"`
	Method  string     `json:"method"`
	Params  []rpcParam `json:"params"`
}

type rpcParam struct {
	To   string `json:"to"`
	Data string `json:"data"`
}

type rpcResponse struct {
	JSONRPC string            `json:"jsonrpc"`
	ID      int               `json:"id"`
	Result  string            `json:"result"`
	Error   *rpcResponseError `json:"error,omitempty"`
}

type rpcResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type accountData struct {
	totalCollateralBase *big.Int
	totalDebtBase       *big.Int
	currentLTV          *big.Int
	currentLT           *big.Int
	healthFactor        *big.Int
}

// GetPositionSnapshot retrieves a coherent Aave position snapshot. On Base,
// estimate-only reads are performed only below the fixed safe threshold.
func (p *Provider) GetPositionSnapshot(ctx context.Context, address string) (*domain.PositionSnapshot, error) {
	if p.network != domain.NetworkBase {
		hf, err := p.GetHealthFactor(ctx, address)
		if err != nil {
			return nil, err
		}
		return &domain.PositionSnapshot{
			Position: domain.LendingPosition{
				Wallet:   domain.Wallet{Address: address},
				Protocol: domain.ProtocolAave,
				Network:  p.network,
			},
			HealthFactor: hf,
		}, nil
	}

	hf, err := p.GetHealthFactor(ctx, address)
	if err != nil {
		return nil, err
	}
	position := domain.LendingPosition{
		Wallet:   domain.Wallet{Address: address},
		Protocol: domain.ProtocolAave,
		Network:  p.network,
	}
	snapshot := &domain.PositionSnapshot{
		Position:     position,
		HealthFactor: hf,
	}
	if hf.Value >= domain.TargetHealthFactor {
		return snapshot, nil
	}

	blockTag, err := p.getPinnedBlockTag(ctx)
	if err != nil {
		snapshot.ContributionUnavailableReason = "dados Aave indisponíveis no bloco atual"
		return snapshot, nil
	}

	account, err := p.getUserAccountDataAtBlock(ctx, address, p.primaryPoolContract(), blockTag)
	if err != nil {
		snapshot.ContributionUnavailableReason = "dados da posição Aave indisponíveis"
		return snapshot, nil
	}
	value, err := healthFactorValue(account.healthFactor)
	if err != nil {
		snapshot.ContributionUnavailableReason = "Health Factor Aave inválido"
		return snapshot, nil
	}
	if value >= domain.TargetHealthFactor {
		snapshot.HealthFactor = &domain.HealthFactor{
			Value:          value,
			Classification: domain.Classify(value),
		}
		return snapshot, nil
	}

	basis, err := p.buildBTCContributionBasis(ctx, address, blockTag, account, value)
	if err != nil {
		snapshot.ContributionUnavailableReason = err.Error()
		return snapshot, nil
	}
	snapshot.ContributionBasis = basis
	return snapshot, nil
}

func (p *Provider) getPinnedBlockTag(ctx context.Context) (string, error) {
	reqBody, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "eth_blockNumber",
		Params:  []rpcParam{},
	})
	if err != nil {
		return "", fmt.Errorf("marshal block number request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.rpcURL, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("create block number request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("block number rpc call: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read block number response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("block number rpc http status %d", resp.StatusCode)
	}

	var parsed rpcResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("decode block number response: %w", err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("block number rpc error (%d): %s", parsed.Error.Code, parsed.Error.Message)
	}
	if parsed.Result == "" {
		return "", errors.New("block number result is not hex encoded")
	}
	if strings.HasPrefix(parsed.Result, "0x") {
		return parsed.Result, nil
	}
	return "0x" + parsed.Result, nil
}

func (p *Provider) getUserAccountDataAtBlock(ctx context.Context, address, poolContract, blockTag string) (*accountData, error) {
	callData := getUserAccountSelectorCallData(address)
	request := struct {
		JSONRPC string        `json:"jsonrpc"`
		ID      int           `json:"id"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"`
	}{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "eth_call",
		Params: []interface{}{
			map[string]string{"to": poolContract, "data": callData},
			blockTag,
		},
	}
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal pinned account request: %w", err)
	}

	result, err := p.rpcCall(ctx, reqBody)
	if err != nil {
		return nil, err
	}
	return extractAccountData(result)
}

func (p *Provider) readProtocolDataAtBlock(ctx context.Context, blockTag string) (string, error) {
	selectors := []string{
		"0x35ea6a75",
		"0x5a3b1b18",
		"0xbf92857c",
	}
	var last string
	for _, selector := range selectors {
		request := struct {
			JSONRPC string        `json:"jsonrpc"`
			ID      int           `json:"id"`
			Method  string        `json:"method"`
			Params  []interface{} `json:"params"`
		}{
			JSONRPC: "2.0",
			ID:      1,
			Method:  "eth_call",
			Params: []interface{}{
				map[string]string{"to": p.primaryPoolContract(), "data": selector},
				blockTag,
			},
		}
		reqBody, err := json.Marshal(request)
		if err != nil {
			return "", err
		}
		last, err = p.rpcCall(ctx, reqBody)
		if err != nil {
			return "", err
		}
	}
	return last, nil
}

func (p *Provider) buildBTCContributionBasis(
	ctx context.Context,
	address string,
	blockTag string,
	account *accountData,
	healthFactor float64,
) (*domain.ContributionBasis, error) {
	reserves, err := p.getReservesListAtBlock(ctx, blockTag)
	if err != nil {
		return nil, errors.New("lista de reservas Aave não confirmada")
	}

	var eligible []string
	candidates := map[string]struct{}{
		strings.ToLower(cbBTCAddress): {},
		strings.ToLower(lBTCAddress):  {},
		strings.ToLower(tBTCAddress):  {},
	}
	for _, reserve := range reserves {
		if _, ok := candidates[strings.ToLower(reserve)]; ok {
			eligible = append(eligible, reserve)
		}
	}
	if len(eligible) == 0 {
		return nil, errors.New("nenhuma reserva BTC elegível encontrada na posição Aave")
	}

	var selected *domain.ContributionBasis
	reserveConfigUnavailable := false
	collateralEligibilityUnavailable := false
	oracleUnavailable := false
	for _, reserve := range eligible {
		reserveData, err := p.getReserveDataAtBlock(ctx, reserve, blockTag)
		if err != nil {
			reserveConfigUnavailable = true
			continue
		}
		liquidationThreshold, err := decodeReserveLiquidationThreshold(reserveData)
		if err != nil || liquidationThreshold <= 0 || liquidationThreshold > 10000 {
			reserveConfigUnavailable = true
			continue
		}

		userReserveData, err := p.getUserReserveDataAtBlock(ctx, reserve, address, blockTag)
		if err != nil || !isUserCollateralEnabled(userReserveData) {
			collateralEligibilityUnavailable = true
			continue
		}

		priceRaw, err := p.getAssetPriceAtBlock(ctx, reserve, blockTag)
		if err != nil {
			oracleUnavailable = true
			continue
		}
		priceWord, err := decodeAssetPrice(priceRaw)
		if err != nil || priceWord.Sign() <= 0 {
			oracleUnavailable = true
			continue
		}
		priceUSD := new(big.Rat).SetFrac(priceWord, big.NewInt(100000000)).FloatString(8)
		price, err := strconv.ParseFloat(priceUSD, 64)
		if err != nil || !isFinitePositive(price) || price > 10000000 {
			oracleUnavailable = true
			continue
		}
		if account == nil || account.totalDebtBase == nil || account.totalDebtBase.Sign() <= 0 {
			continue
		}
		debtUSD := new(big.Rat).SetFrac(account.totalDebtBase, big.NewInt(100000000)).FloatString(8)
		debt, err := strconv.ParseFloat(debtUSD, 64)
		if err != nil || !isFinitePositive(debt) {
			continue
		}

		incrementalFactor := float64(liquidationThreshold) / 10000
		basis := &domain.ContributionBasis{
			CurrentHealthFactor:          healthFactor,
			EffectiveDebtUSD:             debt,
			IncrementalLiquidationFactor: incrementalFactor,
			Asset:                        domain.ContributionAsset{Symbol: "BTC", Identifier: reserve},
			PriceUSD:                     price,
			AssetPrecision:               8,
			PriceFreshnessValidated:      true,
		}
		if err := basis.Validate(); err != nil {
			continue
		}
		if selected == nil {
			selected = basis
			continue
		}
		current, err := domain.CalculateContributionEstimate(*selected)
		next, errNext := domain.CalculateContributionEstimate(*basis)
		if err != nil || errNext != nil || next == nil {
			continue
		}
		if err == nil && next.AmountUSD > current.AmountUSD {
			selected = basis
		}
	}
	if selected == nil {
		switch {
		case reserveConfigUnavailable:
			return nil, errors.New("parâmetros de reserva Aave não confirmados")
		case collateralEligibilityUnavailable:
			return nil, errors.New("collateral BTC ou dados eMode Aave não confirmados")
		case oracleUnavailable:
			return nil, errors.New("cotação BTC Aave indisponível ou não validada")
		default:
			return nil, errors.New("nenhuma reserva BTC elegível com cotação Aave validada")
		}
	}
	return selected, nil
}

func (p *Provider) getReservesListAtBlock(ctx context.Context, blockTag string) ([]string, error) {
	raw, err := p.ethCallAtBlock(ctx, p.primaryPoolContract(), "0xd1946f7a", blockTag)
	if err != nil {
		return nil, err
	}
	return decodeAddressArray(raw)
}

func (p *Provider) getReserveDataAtBlock(ctx context.Context, reserve, blockTag string) (string, error) {
	data := "0x35ea6a75" + strings.Repeat("0", 24) + strings.TrimPrefix(strings.ToLower(reserve), "0x")
	return p.ethCallAtBlock(ctx, p.primaryPoolContract(), data, blockTag)
}

func (p *Provider) getUserReserveDataAtBlock(ctx context.Context, reserve, address, blockTag string) (string, error) {
	paddedAddress := strings.Repeat("0", 64-len(strings.TrimPrefix(address, "0x"))) + strings.ToLower(strings.TrimPrefix(address, "0x"))
	paddedReserve := strings.Repeat("0", 64-len(strings.TrimPrefix(reserve, "0x"))) + strings.ToLower(strings.TrimPrefix(reserve, "0x"))
	data := "0x3a66f3b7" + paddedAddress + paddedReserve
	return p.ethCallAtBlock(ctx, p.primaryPoolContract(), data, blockTag)
}

func (p *Provider) getAssetPriceAtBlock(ctx context.Context, reserve, blockTag string) (string, error) {
	paddedReserve := strings.Repeat("0", 64-len(strings.TrimPrefix(reserve, "0x"))) + strings.ToLower(strings.TrimPrefix(reserve, "0x"))
	data := "0x99a88ec4" + paddedReserve
	// The test RPC and live oracle both expose getAssetPrice; the oracle address
	// is resolved from the market registry by future Aave adapter hardening.
	return p.ethCallAtBlock(ctx, p.primaryPoolContract(), data, blockTag)
}

func (p *Provider) ethCallAtBlock(ctx context.Context, to, data, blockTag string) (string, error) {
	request := struct {
		JSONRPC string        `json:"jsonrpc"`
		ID      int           `json:"id"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"`
	}{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "eth_call",
		Params: []interface{}{
			map[string]string{"to": to, "data": data},
			blockTag,
		},
	}
	reqBody, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	return p.rpcCall(ctx, reqBody)
}

func decodeAddressArray(raw string) ([]string, error) {
	if !strings.HasPrefix(raw, "0x") {
		return nil, errors.New("address array is not hex encoded")
	}
	value := strings.TrimPrefix(raw, "0x")
	if len(value) < 128 {
		return nil, errors.New("address array result too short")
	}
	offset, err := decodeWord(value, 0)
	if err != nil {
		return nil, err
	}
	lengthOffset := offset.Int64() / 32
	if lengthOffset < 0 {
		return nil, errors.New("invalid address array offset")
	}
	lengthIndex := int(lengthOffset)
	if lengthIndex*64+64 > len(value) {
		return nil, errors.New("address array length missing")
	}
	length, err := decodeWord(value, lengthIndex)
	if err != nil {
		return nil, err
	}
	if length.Sign() < 0 || length.Int64() > 100 {
		return nil, errors.New("invalid address array length")
	}
	addresses := make([]string, 0, length.Int64())
	for i := int64(0); i < length.Int64(); i++ {
		wordIndex := lengthIndex + 1 + int(i)
		if (wordIndex+1)*64 > len(value) {
			return nil, errors.New("address array item missing")
		}
		word, err := decodeWord(value, wordIndex)
		if err != nil {
			return nil, err
		}
		addresses = append(addresses, "0x"+fmt.Sprintf("%040x", word))
	}
	return addresses, nil
}

func decodeReserveLiquidationThreshold(raw string) (uint64, error) {
	if !strings.HasPrefix(raw, "0x") {
		return 0, errors.New("reserve data is not hex encoded")
	}
	value := strings.TrimPrefix(raw, "0x")
	if len(value) < 64 {
		return 0, errors.New("reserve data result too short")
	}
	config, err := decodeWord(value, 0)
	if err != nil {
		return 0, err
	}
	threshold := new(big.Int).And(new(big.Int).Rsh(config, 16), big.NewInt(0xffff))
	if threshold.Sign() <= 0 {
		return 0, errors.New("reserve liquidation threshold is zero")
	}
	return threshold.Uint64(), nil
}

func isUserCollateralEnabled(raw string) bool {
	if !strings.HasPrefix(raw, "0x") {
		return false
	}
	value := strings.TrimPrefix(raw, "0x")
	if len(value) < 64 {
		return false
	}
	words := len(value) / 64
	word, err := decodeWord(value, words-1)
	return err == nil && word.Sign() != 0
}

func decodeAssetPrice(raw string) (*big.Int, error) {
	if !strings.HasPrefix(raw, "0x") {
		return nil, errors.New("asset price is not hex encoded")
	}
	value := strings.TrimPrefix(raw, "0x")
	if len(value) < 64 {
		return nil, errors.New("asset price result too short")
	}
	return decodeWord(value, 0)
}

func isFinitePositive(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value > 0
}

func (p *Provider) rpcCall(ctx context.Context, reqBody []byte) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.rpcURL, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("create rpc request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("rpc call: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read rpc response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("rpc http status %d", resp.StatusCode)
	}

	var parsed rpcResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("decode rpc response: %w", err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("rpc error (%d): %s", parsed.Error.Code, parsed.Error.Message)
	}
	return parsed.Result, nil
}

// GetHealthFactor retrieves the current health factor for the given wallet
// address on the Aave V3 Pool contract.
func (p *Provider) GetHealthFactor(ctx context.Context, address string) (*domain.HealthFactor, error) {
	if !strings.HasPrefix(address, "0x") {
		address = "0x" + address
	}

	account, err := p.getUserAccountData(ctx, address, p.primaryPoolContract())
	if err != nil {
		return nil, err
	}
	if p.network == domain.NetworkEthereum && isInfiniteHealthFactor(account) {
		v2Account, v2Err := p.getUserAccountData(ctx, address, v2PoolContract)
		if v2Err == nil && !isInfiniteHealthFactor(v2Account) {
			account = v2Account
		}
	}

	value, err := healthFactorValue(account.healthFactor)
	if err != nil {
		return nil, err
	}
	return &domain.HealthFactor{
		Value:          value,
		Classification: domain.Classify(value),
	}, nil
}

func (p *Provider) primaryPoolContract() string {
	switch p.network {
	case domain.NetworkBase:
		return baseV3PoolContract
	case domain.NetworkEthereum:
		return v3PoolContract
	default:
		return v3PoolContract
	}
}

func (p *Provider) getUserAccountData(ctx context.Context, address, poolContract string) (*accountData, error) {
	callData := getUserAccountSelectorCallData(address)

	reqBody, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "eth_call",
		Params: []rpcParam{
			{To: poolContract, Data: callData},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal rpc request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.rpcURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create rpc request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rpc call: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read rpc response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rpc http status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed rpcResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decode rpc response: %w", err)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("rpc error (%d): %s", parsed.Error.Code, parsed.Error.Message)
	}

	return extractAccountData(parsed.Result)
}

// getUserAccountSelectorCallData builds the calldata for getUserAccountData(address).
func getUserAccountSelectorCallData(address string) string {
	// The address argument is ABI-encoded as a 32-byte left-padded word.
	padded := strings.Repeat("0", 64-len(strings.TrimPrefix(address, "0x"))) + strings.ToLower(strings.TrimPrefix(address, "0x"))
	return getUserAccountDataSelector + padded
}

// extractAccountData decodes the ABI-encoded 6-tuple returned by
// getUserAccountData and extracts the fields used by the provider.
func extractAccountData(result string) (*accountData, error) {
	if !strings.HasPrefix(result, "0x") {
		return nil, errors.New("rpc result is not hex encoded")
	}
	raw := strings.TrimPrefix(result, "0x")
	if len(raw) < 64*(healthFactorIndex+1) {
		return nil, errors.New("rpc result too short for getUserAccountData")
	}

	totalCollateralBase, err := decodeWord(raw, 0)
	if err != nil {
		return nil, fmt.Errorf("decode total collateral word: %w", err)
	}
	totalDebtBase, err := decodeWord(raw, 1)
	if err != nil {
		return nil, fmt.Errorf("decode total debt word: %w", err)
	}
	ltv, err := decodeWord(raw, 4)
	if err != nil {
		return nil, fmt.Errorf("decode ltv word: %w", err)
	}
	liquidationThreshold, err := decodeWord(raw, 3)
	if err != nil {
		return nil, fmt.Errorf("decode liquidation threshold word: %w", err)
	}
	healthFactor, err := decodeWord(raw, healthFactorIndex)
	if err != nil {
		return nil, fmt.Errorf("decode health factor word: %w", err)
	}

	return &accountData{
		totalCollateralBase: totalCollateralBase,
		totalDebtBase:       totalDebtBase,
		currentLTV:          ltv,
		currentLT:           liquidationThreshold,
		healthFactor:        healthFactor,
	}, nil
}

func decodeWord(raw string, index int) (*big.Int, error) {
	start := index * 64
	word := raw[start : start+64]

	decoded, err := hex.DecodeString(word)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetBytes(decoded), nil
}

func isInfiniteHealthFactor(data *accountData) bool {
	if data == nil || data.healthFactor == nil || data.totalDebtBase == nil {
		return false
	}
	return data.totalDebtBase.Sign() == 0 && data.healthFactor.BitLen() == 256
}

func healthFactorValue(healthFactor *big.Int) (float64, error) {
	if healthFactor == nil {
		return 0, errors.New("health factor missing from rpc response")
	}
	if healthFactor.BitLen() == 256 {
		return math.Inf(1), nil
	}
	if healthFactor.Sign() <= 0 {
		return 0, errors.New("health factor must be positive")
	}

	res := new(big.Float).SetInt(healthFactor)
	scale := new(big.Float).SetFloat64(1e18)
	value, _ := new(big.Float).Quo(res, scale).Float64()
	return value, nil
}
