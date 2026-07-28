package domain

import (
	"errors"
	"regexp"
	"strings"
)

const (
	ProtocolAave   = "aave"
	ProtocolKamino = "kamino"

	NetworkEthereum = "ethereum"
	NetworkSolana   = "solana"
)

var ethAddressRe = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)

func isValidSolanaAddress(addr string) bool {
	if len(addr) < 32 || len(addr) > 44 {
		return false
	}
	for _, b := range []byte(addr) {
		if !isBase58Byte(b) {
			return false
		}
	}
	return true
}

func isBase58Byte(b byte) bool {
	return (b >= '1' && b <= '9') ||
		(b >= 'A' && b <= 'H') ||
		(b >= 'J' && b <= 'N') ||
		(b >= 'P' && b <= 'Z') ||
		(b >= 'a' && b <= 'k') ||
		(b >= 'm' && b <= 'z')
}

func supportedProtocols() []string {
	return []string{ProtocolAave, ProtocolKamino}
}

func supportedNetworks() []string {
	return []string{NetworkEthereum, NetworkSolana}
}

func isValidProtocol(p string) bool {
	for _, sp := range supportedProtocols() {
		if p == sp {
			return true
		}
	}
	return false
}

func isValidNetwork(n string) bool {
	for _, sn := range supportedNetworks() {
		if n == sn {
			return true
		}
	}
	return false
}

func isValidAddressForNetwork(addr, network string) bool {
	switch network {
	case NetworkEthereum:
		return ethAddressRe.MatchString(addr)
	case NetworkSolana:
		return isValidSolanaAddress(addr)
	default:
		return false
	}
}

// Wallet represents a cryptocurrency wallet monitored by the application.
type Wallet struct {
	Address string
	Alias   string
}

// Validate checks that the wallet fields meet the required constraints.
func (w Wallet) Validate() error {
	var errs []string
	if strings.TrimSpace(w.Address) == "" {
		errs = append(errs, "address must not be empty")
	}
	if w.Alias != "" && strings.TrimSpace(w.Alias) == "" {
		errs = append(errs, "alias must not be blank if provided")
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

// LendingPosition represents a wallet's participation in a lending protocol
// on a specific blockchain network.
type LendingPosition struct {
	Wallet   Wallet
	Protocol string
	Network  string
}

// Validate checks that the lending position fields meet the required
// constraints, including wallet validity for the target network.
func (p LendingPosition) Validate() error {
	var errs []string

	if !isValidProtocol(p.Protocol) {
		errs = append(errs, "unsupported protocol: "+p.Protocol)
	}
	if !isValidNetwork(p.Network) {
		errs = append(errs, "unsupported network: "+p.Network)
	}
	if err := p.Wallet.Validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if p.Protocol != "" && p.Network != "" && p.Wallet.Address != "" {
		if !isValidAddressForNetwork(p.Wallet.Address, p.Network) {
			errs = append(errs, "address does not match network format: "+p.Network)
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}