package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/marcus/health-factor-monitor/internal/domain"
	"github.com/marcus/health-factor-monitor/internal/infrastructure/config"
)

const (
	validEthAddress = "0x0000000000000000000000000000000000000001"
	validSolAddress = "11111111111111111111111111111111"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	return path
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name            string
		content         string
		path            string
		wantErr         bool
		wantErrContains string
		wantPositions   int
	}{
		{
			name: "valid config with two positions",
			content: `{
				"rpc_endpoints": {
					"ethereum": "https://eth.example.com",
					"solana": "https://sol.example.com"
				},
				"positions": [
					{
						"alias": "aave-main",
						"address": "` + validEthAddress + `",
						"network": "ethereum",
						"protocol": "aave"
					},
					{
						"alias": "kamino-main",
						"address": "` + validSolAddress + `",
						"network": "solana",
						"protocol": "kamino"
					}
				]
			}`,
			wantErr:       false,
			wantPositions: 2,
		},
		{
			name:    "missing file",
			content: "",
			path:    filepath.Join(t.TempDir(), "nonexistent.json"),
			wantErr: true,
		},
		{
			name: "malformed JSON",
			content: `{
				"rpc_endpoints": {
					"ethereum": "https://eth.example.com"
				},
				"positions": [
					{
						"alias": "aave-main",
						"address": "` + validEthAddress + `",
						"network": "ethereum",
						"protocol": "aave",
					}
				]
			}`,
			wantErr: true,
		},
		{
			name: "unsupported protocol",
			content: `{
				"rpc_endpoints": {
					"ethereum": "https://eth.example.com"
				},
				"positions": [
					{
						"alias": "morpho-main",
						"address": "` + validEthAddress + `",
						"network": "ethereum",
						"protocol": "morpho"
					}
				]
			}`,
			wantErr:         true,
			wantErrContains: "unsupported protocol",
		},
		{
			name: "missing required field",
			content: `{
				"rpc_endpoints": {
					"ethereum": "https://eth.example.com"
				},
				"positions": [
					{
						"alias": "aave-main",
						"address": "` + validEthAddress + `",
						"protocol": "aave"
					}
				]
			}`,
			wantErr:         true,
			wantErrContains: "unsupported network",
		},
		{
			name: "invalid address format",
			content: `{
				"rpc_endpoints": {
					"ethereum": "https://eth.example.com"
				},
				"positions": [
					{
						"alias": "aave-main",
						"address": "not-an-address",
						"network": "ethereum",
						"protocol": "aave"
					}
				]
			}`,
			wantErr:         true,
			wantErrContains: "address does not match network format",
		},
		{
			name: "empty positions",
			content: `{
				"rpc_endpoints": {
					"ethereum": "https://eth.example.com"
				},
				"positions": []
			}`,
			wantErr:         true,
			wantErrContains: "at least one lending position",
		},
		{
			name: "network without rpc endpoint",
			content: `{
				"rpc_endpoints": {
					"solana": "https://sol.example.com"
				},
				"positions": [
					{
						"alias": "aave-main",
						"address": "` + validEthAddress + `",
						"network": "ethereum",
						"protocol": "aave"
					}
				]
			}`,
			wantErr:         true,
			wantErrContains: "no rpc endpoint defined for network: ethereum",
		},
		{
			name: "missing alias is allowed",
			content: `{
				"rpc_endpoints": {
					"ethereum": "https://eth.example.com"
				},
				"positions": [
					{
						"address": "` + validEthAddress + `",
						"network": "ethereum",
						"protocol": "aave"
					}
				]
			}`,
			wantErr:       false,
			wantPositions: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.path
			if path == "" {
				path = writeConfig(t, tt.content)
			}

			r := config.NewReader(path)
			cfg, err := r.Load()

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("expected error to contain %q, got: %s", tt.wantErrContains, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg == nil {
				t.Fatal("expected config, got nil")
			}
			if len(cfg.Positions) != tt.wantPositions {
				t.Errorf("got %d positions, want %d", len(cfg.Positions), tt.wantPositions)
			}
			if len(cfg.RPCEndpoints) == 0 {
				t.Error("expected rpc endpoints to be populated")
			}
		})
	}
}

func TestLoadReturnsValidatedConfig(t *testing.T) {
	path := writeConfig(t, `{
		"rpc_endpoints": {
			"ethereum": "https://eth.example.com",
			"solana": "https://sol.example.com"
		},
		"positions": [
			{
				"alias": "aave-main",
				"address": "`+validEthAddress+`",
				"network": "ethereum",
				"protocol": "aave"
			},
			{
				"alias": "kamino-main",
				"address": "`+validSolAddress+`",
				"network": "solana",
				"protocol": "kamino"
			}
		]
	}`)

	loader := config.NewReader(path)
	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("loaded config should be valid, got: %v", err)
	}
}

func TestReaderImplementsConfigLoader(t *testing.T) {
	var _ domain.ConfigLoader = config.NewReader("")
}
