# Research: Health Factor Monitor MVP

## Integration Philosophy

### Decision

Prefer the Go standard library whenever practical.

Third-party dependencies may be introduced when they provide significant value, are officially maintained by the protocol, or substantially reduce implementation complexity without compromising the architecture.

The application architecture must never depend on a specific SDK. All protocol-specific implementations remain isolated behind provider interfaces.

### Rationale

- Keeps the codebase simple and maintainable.
- Avoids unnecessary dependencies.
- Allows official SDKs or client libraries when they provide clear advantages.
- Prevents infrastructure decisions from leaking into the domain layer.
- Aligns with the Constitution principles of Provider-Based Architecture and Interface Independence.

### Guidelines

Prefer:

- Go standard library
- Official SDKs
- Official client libraries
- Well-established libraries with strong community adoption

Avoid:

- Dependencies that only save a few lines of code
- Unmaintained libraries
- Scraping unofficial websites
- Libraries that couple business logic to infrastructure

---

# Blockchain Integration Strategy

## Decision

Use the most stable official integration mechanism available for each supported protocol.

- **Aave** → Direct smart contract interaction through Ethereum JSON-RPC.
- **Kamino** → Official REST API or official SDK whenever sufficient.
- Keep protocol-specific communication completely encapsulated inside provider implementations.

### Rationale

Different protocols expose different integration models.

Rather than forcing every provider to use raw RPC, each provider should use the most stable and maintainable official interface.

The domain layer should remain completely unaware of whether data came from:

- JSON-RPC
- REST
- SDK
- GraphQL
- Future protocol integrations

---

# Configuration File Format

## Decision

Use JSON as the configuration format.

## Rationale

- Supported by the Go standard library (`encoding/json`)
- No additional dependency required
- Easy validation
- Easy generation
- Sufficient for MVP

### Alternatives Considered

### YAML

Pros:

- More human friendly

Cons:

- Requires external dependency

### TOML

Pros:

- Readable

Cons:

- Requires external dependency

### Environment Variables

Rejected because multi-wallet configuration becomes difficult to manage.

## Example

```json
{
  "rpc_endpoints": {
    "ethereum": "https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY",
    "solana": "https://api.mainnet-beta.solana.com"
  },
  "positions": [
    {
      "alias": "Main Aave",
      "provider": "aave",
      "network": "ethereum",
      "address": "0x..."
    },
    {
      "alias": "Main Kamino",
      "provider": "kamino",
      "network": "solana",
      "address": "..."
    }
  ]
}
```

---

# Aave Health Factor Retrieval

## Decision

Retrieve Health Factor directly from the Aave Pool contract using Ethereum JSON-RPC (`eth_call`).

### Method

```
getUserAccountData(address)
```

### Rationale

- Official Aave interface
- Single blockchain call
- Returns Health Factor directly
- No state reconstruction required
- Stable across Aave versions

### Implementation Notes

Pool Contract (Ethereum V3):

```
0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2
```

Returned values:

- totalCollateralBase
- totalDebtBase
- availableBorrowsBase
- currentLiquidationThreshold
- ltv
- healthFactor

Health Factor is returned already calculated (scaled by 1e18).

---

# Kamino Health Factor Retrieval

## Decision

Prefer Kamino's official interfaces instead of manually decoding Solana account layouts.

Preferred integration order:

1. Official REST API
2. Official SDK
3. Direct Solana RPC

### Rationale

- Recommended by Kamino documentation
- Lower maintenance cost
- Less protocol-specific parsing
- More resilient to protocol upgrades
- Easier testing
- Cleaner implementation

### Implementation Notes

The provider should retrieve user lending obligations through Kamino's official interfaces.

Health Factor should be derived from the returned obligation data.

Only use direct RPC if official interfaces cannot satisfy the application's requirements.

Manual parsing of account offsets should be treated as a last resort.

---

# Provider Architecture

## Decision

Separate business logic from transport mechanisms.

Architecture:

```
Application

↓

Provider

↓

Data Source

↓

REST / RPC / SDK
```

Example:

```
AaveProvider

↓

EthereumRPCDataSource

↓

Ethereum JSON-RPC
```

```
KaminoProvider

↓

KaminoAPIDataSource

↓

Official REST API
```

Responsibilities:

### Provider

- Business rules
- Health Factor retrieval
- Error handling
- Result normalization

### Data Source

- HTTP
- RPC
- SDK interaction
- Authentication
- Serialization

---

# Health Classification

## Decision

Use configurable Health Factor classifications.

Default thresholds:

| Classification | Default Threshold |
|----------------|-------------------|
| Safe | > 1.5 |
| Warning | > 1.0 and ≤ 1.5 |
| Critical | ≤ 1.0 |

Thresholds must be configurable.

### Rationale

- Easy interpretation
- Consistent CLI output
- Satisfies specification requirements
- Allows future customization

---

# Go Version

## Decision

Target the latest stable Go release supported by the project.

### Rationale

- Latest language improvements
- Better performance
- Better tooling
- Longer support lifecycle

---

# CLI Framework

## Decision

Use the Go standard `flag` package.

### Rationale

The MVP contains a single command with few options.

No advanced CLI framework is currently justified.

---

# Dependency Policy

## Decision

Do not enforce a "Zero External Dependencies" rule.

Instead:

- Prefer the Go standard library.
- Introduce dependencies only when they provide significant value.
- Prefer official SDKs and official client libraries when available.
- Evaluate each dependency based on maintainability, stability, and architectural impact.

### Examples of Acceptable Dependencies

- Official blockchain SDKs
- Official protocol SDKs
- Well-established Go libraries
- Widely adopted community libraries

### Examples of Unacceptable Dependencies

- Libraries that only reduce a few lines of code
- Unmaintained projects
- Scraping libraries for unofficial websites
- Libraries that couple business logic to infrastructure

---

# Future Evolution

The Provider abstraction allows adding new protocols without modifying the application layer.

Potential future providers:

- Morpho
- Compound
- Spark
- Venus
- Radiant
- MarginFi

Each provider may internally choose the most appropriate integration mechanism:

- REST
- JSON-RPC
- SDK
- GraphQL

The application layer remains completely independent of these implementation details.