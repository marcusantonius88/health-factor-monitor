package domain

import (
	"context"
	"errors"
	"strings"
	"testing"
)

const (
	validEthAddress = "0x0000000000000000000000000000000000000001"
	validSolAddress = "11111111111111111111111111111111"
)

func TestWalletValidate(t *testing.T) {
	tests := []struct {
		name    string
		wallet  Wallet
		wantErr bool
	}{
		{
			name:    "valid wallet with alias",
			wallet:  Wallet{Address: validEthAddress, Alias: "main"},
			wantErr: false,
		},
		{
			name:    "valid wallet without alias",
			wallet:  Wallet{Address: validEthAddress},
			wantErr: false,
		},
		{
			name:    "empty address",
			wallet:  Wallet{Address: "", Alias: "main"},
			wantErr: true,
		},
		{
			name:    "whitespace address",
			wallet:  Wallet{Address: "   ", Alias: "main"},
			wantErr: true,
		},
		{
			name:    "blank alias",
			wallet:  Wallet{Address: validEthAddress, Alias: "   "},
			wantErr: true,
		},
		{
			name:    "empty address and blank alias",
			wallet:  Wallet{Address: "", Alias: "   "},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.wallet.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Wallet.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLendingPositionValidate(t *testing.T) {
	tests := []struct {
		name     string
		position LendingPosition
		wantErr  bool
	}{
		{
			name:     "valid aave position",
			position: LendingPosition{Wallet: Wallet{Address: validEthAddress}, Protocol: ProtocolAave, Network: NetworkEthereum},
			wantErr:  false,
		},
		{
			name:     "valid kamino position",
			position: LendingPosition{Wallet: Wallet{Address: validSolAddress}, Protocol: ProtocolKamino, Network: NetworkSolana},
			wantErr:  false,
		},
		{
			name:     "unsupported protocol",
			position: LendingPosition{Wallet: Wallet{Address: validEthAddress}, Protocol: "morpho", Network: NetworkEthereum},
			wantErr:  true,
		},
		{
			name:     "unsupported network",
			position: LendingPosition{Wallet: Wallet{Address: validEthAddress}, Protocol: ProtocolAave, Network: "arbitrum"},
			wantErr:  true,
		},
		{
			name:     "ethereum address on solana network",
			position: LendingPosition{Wallet: Wallet{Address: validEthAddress}, Protocol: ProtocolKamino, Network: NetworkSolana},
			wantErr:  true,
		},
		{
			name:     "solana address on ethereum network",
			position: LendingPosition{Wallet: Wallet{Address: validSolAddress}, Protocol: ProtocolAave, Network: NetworkEthereum},
			wantErr:  true,
		},
		{
			name:     "invalid ethereum address format",
			position: LendingPosition{Wallet: Wallet{Address: "0xabc"}, Protocol: ProtocolAave, Network: NetworkEthereum},
			wantErr:  true,
		},
		{
			name:     "invalid solana address character",
			position: LendingPosition{Wallet: Wallet{Address: "00000000000000000000000000000000"}, Protocol: ProtocolKamino, Network: NetworkSolana},
			wantErr:  true,
		},
		{
			name:     "empty wallet address",
			position: LendingPosition{Wallet: Wallet{}, Protocol: ProtocolAave, Network: NetworkEthereum},
			wantErr:  true,
		},
		{
			name:     "multiple errors",
			position: LendingPosition{Wallet: Wallet{Address: ""}, Protocol: "unknown", Network: "unknown"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.position.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("LendingPosition.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHealthFactorValidate(t *testing.T) {
	tests := []struct {
		name    string
		health  HealthFactor
		wantErr bool
	}{
		{
			name:    "safe health factor",
			health:  HealthFactor{Value: 2.5, Classification: ClassificationSafe},
			wantErr: false,
		},
		{
			name:    "warning health factor",
			health:  HealthFactor{Value: 1.2, Classification: ClassificationWarning},
			wantErr: false,
		},
		{
			name:    "critical health factor",
			health:  HealthFactor{Value: 0.9, Classification: ClassificationCritical},
			wantErr: false,
		},
		{
			name:    "zero value",
			health:  HealthFactor{Value: 0, Classification: ClassificationSafe},
			wantErr: true,
		},
		{
			name:    "negative value",
			health:  HealthFactor{Value: -1, Classification: ClassificationSafe},
			wantErr: true,
		},
		{
			name:    "invalid classification",
			health:  HealthFactor{Value: 2.5, Classification: "unknown"},
			wantErr: true,
		},
		{
			name:    "zero value and invalid classification",
			health:  HealthFactor{Value: 0, Classification: ""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.health.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("HealthFactor.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProviderResultValidate(t *testing.T) {
	position := LendingPosition{Wallet: Wallet{Address: validEthAddress}, Protocol: ProtocolAave, Network: NetworkEthereum}
	validHF := &HealthFactor{Value: 2.5, Classification: ClassificationSafe}
	invalidHF := &HealthFactor{Value: 0, Classification: ClassificationSafe}

	tests := []struct {
		name    string
		result  ProviderResult
		wantErr bool
	}{
		{
			name:    "health factor present, no error",
			result:  ProviderResult{Position: position, HealthFactor: validHF},
			wantErr: false,
		},
		{
			name:    "error present, no health factor",
			result:  ProviderResult{Position: position, Error: "connection failed"},
			wantErr: false,
		},
		{
			name:    "both health factor and error",
			result:  ProviderResult{Position: position, HealthFactor: validHF, Error: "connection failed"},
			wantErr: true,
		},
		{
			name:    "neither health factor nor error",
			result:  ProviderResult{Position: position},
			wantErr: true,
		},
		{
			name:    "invalid health factor propagated",
			result:  ProviderResult{Position: position, HealthFactor: invalidHF},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.result.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ProviderResult.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	validPosition := LendingPosition{Wallet: Wallet{Address: validEthAddress}, Protocol: ProtocolAave, Network: NetworkEthereum}

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				RPCEndpoints: map[string]string{NetworkEthereum: "https://eth.example.com"},
				Positions:    []LendingPosition{validPosition},
			},
			wantErr: false,
		},
		{
			name: "empty rpc endpoints",
			config: Config{
				RPCEndpoints: map[string]string{},
				Positions:    []LendingPosition{validPosition},
			},
			wantErr: true,
		},
		{
			name: "empty positions",
			config: Config{
				RPCEndpoints: map[string]string{NetworkEthereum: "https://eth.example.com"},
			},
			wantErr: true,
		},
		{
			name: "position network without rpc endpoint",
			config: Config{
				RPCEndpoints: map[string]string{NetworkSolana: "https://sol.example.com"},
				Positions:    []LendingPosition{validPosition},
			},
			wantErr: true,
		},
		{
			name: "invalid position",
			config: Config{
				RPCEndpoints: map[string]string{NetworkEthereum: "https://eth.example.com"},
				Positions:    []LendingPosition{{Wallet: Wallet{Address: "bad"}, Protocol: ProtocolAave, Network: NetworkEthereum}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestErrorMessagesIncludeAllIssues(t *testing.T) {
	position := LendingPosition{Wallet: Wallet{Address: ""}, Protocol: "unknown", Network: "unknown"}
	err := position.Validate()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	for _, fragment := range []string{"unsupported protocol", "unsupported network", "address must not be empty"} {
		if !strings.Contains(err.Error(), fragment) {
			t.Errorf("expected error to contain %q, got: %s", fragment, err)
		}
	}
}

type mockProvider struct {
	protocol string
	network  string
	hf       *HealthFactor
	err      error
}

func (m *mockProvider) Protocol() string { return m.protocol }
func (m *mockProvider) Network() string  { return m.network }

func (m *mockProvider) GetHealthFactor(_ context.Context, _ string) (*HealthFactor, error) {
	return m.hf, m.err
}

var _ HealthFactorProvider = (*mockProvider)(nil)

func TestHealthFactorProviderContract(t *testing.T) {
	tests := []struct {
		name     string
		provider HealthFactorProvider
		protocol string
		network  string
		address  string
		wantHF   *HealthFactor
		wantErr  bool
	}{
		{
			name:     "aave provider returns health factor",
			provider: &mockProvider{protocol: ProtocolAave, network: NetworkEthereum, hf: &HealthFactor{Value: 2.5, Classification: ClassificationSafe}},
			protocol: ProtocolAave,
			network:  NetworkEthereum,
			address:  validEthAddress,
			wantHF:   &HealthFactor{Value: 2.5, Classification: ClassificationSafe},
		},
		{
			name:     "provider returns error",
			provider: &mockProvider{protocol: ProtocolKamino, network: NetworkSolana, err: errors.New("connection failed")},
			protocol: ProtocolKamino,
			network:  NetworkSolana,
			address:  validSolAddress,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.provider.Protocol(); got != tt.protocol {
				t.Errorf("Protocol() = %q, want %q", got, tt.protocol)
			}
			if got := tt.provider.Network(); got != tt.network {
				t.Errorf("Network() = %q, want %q", got, tt.network)
			}
			got, err := tt.provider.GetHealthFactor(context.Background(), tt.address)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetHealthFactor() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got == nil || got.Value != tt.wantHF.Value || got.Classification != tt.wantHF.Classification {
				t.Errorf("GetHealthFactor() = %+v, want %+v", got, tt.wantHF)
			}
		})
	}
}

func TestClassify(t *testing.T) {
	tests := []struct {
		name       string
		value      float64
		thresholds []float64
		want       string
	}{
		{name: "safe above default", value: 2.5, want: ClassificationSafe},
		{name: "safe above default boundary", value: 2.0, want: ClassificationSafe},
		{name: "warning at safe boundary", value: 1.5, want: ClassificationWarning},
		{name: "warning at upper bound", value: 1.4, want: ClassificationWarning},
		{name: "warning just above critical", value: 1.01, want: ClassificationWarning},
		{name: "critical at critical max", value: 1.0, want: ClassificationCritical},
		{name: "critical below default", value: 0.9, want: ClassificationCritical},
		{name: "critical at zero", value: 0, want: ClassificationCritical},
		{
			name:       "custom thresholds safe",
			value:      3.5,
			thresholds: []float64{3.0, 2.0},
			want:       ClassificationSafe,
		},
		{
			name:       "custom thresholds warning",
			value:      2.5,
			thresholds: []float64{3.0, 2.0},
			want:       ClassificationWarning,
		},
		{
			name:       "custom thresholds critical",
			value:      2.0,
			thresholds: []float64{3.0, 2.1},
			want:       ClassificationCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Classify(tt.value, tt.thresholds...); got != tt.want {
				t.Errorf("Classify(%v) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}
