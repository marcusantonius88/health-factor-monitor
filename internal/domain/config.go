package domain

import (
	"errors"
	"strings"
)

// Config represents the user's wallet and network configuration loaded from
// the config file.
type Config struct {
	RPCEndpoints map[string]string
	Positions    []LendingPosition
}

// Validate checks that the configuration meets the required constraints.
func (c Config) Validate() error {
	var errs []string
	if len(c.RPCEndpoints) == 0 {
		errs = append(errs, "at least one rpc endpoint must be defined")
	}
	if len(c.Positions) == 0 {
		errs = append(errs, "at least one lending position must be defined")
	}
	for _, pos := range c.Positions {
		if err := pos.Validate(); err != nil {
			errs = append(errs, err.Error())
		}
		if _, ok := c.RPCEndpoints[pos.Network]; !ok {
			errs = append(errs, "no rpc endpoint defined for network: "+pos.Network)
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

// ConfigLoader reads and validates wallet configuration.
type ConfigLoader interface {
	// Load reads configuration from the configured source path.
	// Returns an error if the file cannot be read, is malformed, or fails
	// validation.
	Load() (*Config, error)
}
