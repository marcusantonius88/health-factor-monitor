package domain

import "context"

// HealthFactorProvider is the interface that every lending protocol adapter
// must implement. Business logic depends only on this interface, never on
// concrete provider implementations.
type HealthFactorProvider interface {
	// Protocol returns the protocol identifier (e.g., "aave", "kamino").
	// The returned value MUST be a stable, lowercase identifier matching the
	// protocol field in the configuration file.
	Protocol() string

	// Network returns the blockchain network identifier (e.g., "ethereum",
	// "solana"). The returned value MUST be a stable, lowercase identifier
	// matching the network field in the configuration file.
	Network() string

	// GetHealthFactor retrieves the current Health Factor for the given
	// wallet address. Returns the Health Factor or an error if the position
	// cannot be queried (network error, invalid address, etc.).
	// Implementations MUST NOT panic and SHOULD respect context cancellation.
	GetHealthFactor(ctx context.Context, address string) (*HealthFactor, error)
}

// ContributionSnapshotProvider is an optional capability for providers that
// can retrieve protocol-specific data used to estimate additional collateral.
// Embedding HealthFactorProvider keeps the existing provider contract and
// provider registrations source-compatible.
//
// GetPositionSnapshot returns a snapshot with a successfully retrieved
// Health Factor independently of estimate availability. An error means that
// the Health Factor position read failed; estimate-only unavailability MUST be
// represented by ContributionUnavailableReason or a missing ContributionBasis
// on an otherwise valid snapshot.
type ContributionSnapshotProvider interface {
	HealthFactorProvider

	GetPositionSnapshot(ctx context.Context, address string) (*PositionSnapshot, error)
}
