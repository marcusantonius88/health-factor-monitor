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

	// getUserAccountDataSelector is the function selector for getUserAccountData(address).
	getUserAccountDataSelector = "0xbf92857c"

	// healthFactorIndex is the position of healthFactor in the returned tuple.
	healthFactorIndex = 5

	// ethCallTimeout bounds each rpc request made by the provider.
	ethCallTimeout = 30 * time.Second
)

// Provider implements the domain.HealthFactorProvider interface for the Aave
// protocol, retrieving health factors directly from the Aave Pool contract via
// an Ethereum JSON-RPC eth_call.
type Provider struct {
	rpcURL string
	network string
	client *http.Client
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
	totalDebtBase *big.Int
	healthFactor  *big.Int
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

	totalDebtBase, err := decodeWord(raw, 1)
	if err != nil {
		return nil, fmt.Errorf("decode total debt word: %w", err)
	}
	healthFactor, err := decodeWord(raw, healthFactorIndex)
	if err != nil {
		return nil, fmt.Errorf("decode health factor word: %w", err)
	}

	return &accountData{
		totalDebtBase: totalDebtBase,
		healthFactor:  healthFactor,
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
