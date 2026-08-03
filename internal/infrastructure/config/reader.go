package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/marcus/health-factor-monitor/internal/domain"
)

// fileConfig mirrors the JSON configuration file format.
type fileConfig struct {
	RPCEndpoints map[string]string `json:"rpc_endpoints"`
	Positions    []filePosition    `json:"positions"`
}

// filePosition mirrors the JSON position entry format.
type filePosition struct {
	Alias    string `json:"alias"`
	Address  string `json:"address"`
	Network  string `json:"network"`
	Protocol string `json:"protocol"`
}

// Reader loads wallet configuration from a JSON file.
// It implements domain.ConfigLoader.
type Reader struct {
	path string
}

// NewReader creates a Reader that reads configuration from the given path.
func NewReader(path string) *Reader {
	return &Reader{path: path}
}

// Load reads, parses, and populates the configuration from the configured
// path. It returns an error if the file cannot be read or parsed as JSON.
func (r *Reader) Load() (*domain.Config, error) {
	data, err := os.ReadFile(r.path)
	if err != nil {
		return nil, fmt.Errorf("cannot read config file %q: %w", r.path, err)
	}

	var fc fileConfig
	if err := json.Unmarshal(data, &fc); err != nil {
		return nil, fmt.Errorf("cannot parse config file %q: %w", r.path, err)
	}

	return toDomainConfig(fc), nil
}

// toDomainConfig converts the flat file format into the domain Config,
// nesting position wallet fields into the domain Wallet entity.
func toDomainConfig(fc fileConfig) *domain.Config {
	positions := make([]domain.LendingPosition, 0, len(fc.Positions))
	for _, p := range fc.Positions {
		positions = append(positions, domain.LendingPosition{
			Wallet: domain.Wallet{
				Address: p.Address,
				Alias:   p.Alias,
			},
			Protocol: p.Protocol,
			Network:  p.Network,
		})
	}

	return &domain.Config{
		RPCEndpoints: fc.RPCEndpoints,
		Positions:    positions,
	}
}