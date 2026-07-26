# Provider Interface Contract

## Purpose

The `HealthFactorProvider` interface defines the contract that every lending protocol adapter must implement. Business logic depends only on this interface, never on concrete provider implementations.

## Interface Definition (Go)

```go
// HealthFactorProvider retrieves Health Factor data for a lending position.
type HealthFactorProvider interface {
    // Protocol returns the protocol identifier (e.g., "aave", "kamino").
    Protocol() string

    // Network returns the blockchain network identifier (e.g., "ethereum", "solana").
    Network() string

    // GetHealthFactor retrieves the current Health Factor for the given wallet address.
    // Returns an error if the position cannot be queried (network error, invalid address, etc.).
    GetHealthFactor(ctx context.Context, address string) (*HealthFactor, error)
}
```

## Contract Rules

1. **`Protocol()`**: Must return a stable, lowercase identifier matching the `protocol` field in the configuration file.
2. **`Network()`**: Must return a stable, lowercase identifier matching the `network` field in the configuration file.
3. **`GetHealthFactor()`**:
   - MUST accept a `context.Context` for cancellation and timeout
   - MUST return the Health Factor as a parsed `float64`
   - MUST return an error if the position cannot be queried (never panic)
   - MUST NOT mutate shared state
   - SHOULD respect context cancellation
4. **Provider Matching**: The application MUST match a configured position to a provider by comparing both `Protocol()` and `Network()` against the position's `protocol` and `network` fields.

## Implementations Required for MVP

| Protocol | Network | Provider Path |
|----------|---------|---------------|
| `aave` | `ethereum` | `internal/infrastructure/aave/provider.go` |
| `kamino` | `solana` | `internal/infrastructure/kamino/provider.go` |
