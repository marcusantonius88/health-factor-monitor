# Application Service Contract

## Purpose

The application service orchestrates the Health Factor retrieval workflow. It composes providers and configuration, executes the checks, and returns results independent of any interface (CLI, REST, etc.).

## Service Definition (Go)

```go
// CheckService is the primary use case for the MVP.
type CheckService interface {
    // CheckAll retrieves Health Factors for all configured positions.
    // Returns one ProviderResult per position, regardless of individual failures.
    CheckAll(ctx context.Context) []ProviderResult

    // CheckByProtocol retrieves Health Factors for positions matching the given protocol.
    CheckByProtocol(ctx context.Context, protocol string) []ProviderResult
}
```

## Contract Rules

1. **`CheckAll()`**:
   - MUST process every configured position
   - MUST NOT stop on individual provider failures
   - MUST return one result per position (success or error)
   - MUST accept `context.Context`
2. **`CheckByProtocol()`**:
   - MUST filter positions by protocol identifier
   - MUST follow the same error-tolerance rules as `CheckAll()`
3. Results MUST contain either a Health Factor or an error, never both, never neither.
