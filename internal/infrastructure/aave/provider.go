package aave

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/marcus/health-factor-monitor/internal/domain"
)

const (
	// poolContract is the Aave V3 Pool contract on Ethereum mainnet.
	poolContract = "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2"

	// getUserAccountDataSelector is the function selector for getUserAccountData(address).
	getUserAccountDataSelector = "0x5c7783a3"

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
	client *http.Client
}

// NewProvider creates an Aave provider that talks to the given JSON-RPC URL.
func NewProvider(rpcURL string) *Provider {
	return &Provider{
		rpcURL: rpcURL,
		client: &http.Client{Timeout: ethCallTimeout},
	}
}

// Protocol returns the provider protocol identifier.
func (p *Provider) Protocol() string { return domain.ProtocolAave }

// Network returns the provider network identifier.
func (p *Provider) Network() string { return domain.NetworkEthereum }

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

// GetHealthFactor retrieves the current health factor for the given wallet
// address on the Aave V3 Pool contract.
func (p *Provider) GetHealthFactor(ctx context.Context, address string) (*domain.HealthFactor, error) {
	if !strings.HasPrefix(address, "0x") {
		address = "0x" + address
	}

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

	value, err := extractHealthFactor(parsed.Result)
	if err != nil {
		return nil, err
	}

	return &domain.HealthFactor{
		Value:          value,
		Classification: domain.Classify(value),
	}, nil
}

// getUserAccountSelectorCallData builds the calldata for getUserAccountData(address).
func getUserAccountSelectorCallData(address string) string {
	return getUserAccountDataSelector + strings.TrimPrefix(strings.ToLower(address), "0x")
}

// extractHealthFactor decodes the ABI-encoded 6-tuple result and returns the
// health factor value (index 5) converted from 1e18 scaling to a float.
func extractHealthFactor(result string) (float64, error) {
	if !strings.HasPrefix(result, "0x") {
		return 0, errors.New("rpc result is not hex encoded")
	}
	raw := strings.TrimPrefix(result, "0x")
	if len(raw) < 64*(healthFactorIndex+1) {
		return 0, errors.New("rpc result too short for getUserAccountData")
	}

	// Each uint256 is 32 bytes (64 hex chars). healthFactor is the 6th word.
	start := healthFactorIndex * 64
	word := raw[start : start+64]

	decoded, err := hex.DecodeString(word)
	if err != nil {
		return 0, fmt.Errorf("decode health factor word: %w", err)
	}

	healthFactor := new(big.Int).SetBytes(decoded)
	if healthFactor.Sign() <= 0 {
		return 0, errors.New("health factor must be positive")
	}

	res := new(big.Float).SetInt(healthFactor)
	scale := new(big.Float).SetFloat64(1e18)
	value, _ := new(big.Float).Quo(res, scale).Float64()
	return value, nil
}
