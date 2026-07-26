# Data Model: Health Factor Monitor MVP

## Entity: Wallet

Represents a single cryptocurrency wallet that the user wants to monitor.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| Address | string | yes | Blockchain address (hex for Ethereum, base58 for Solana) |
| Alias | string | no | Human-readable label for display purposes |

**Validation Rules**:
- Address must be non-empty
- Address format must match the associated blockchain network (Ethereum: 0x-prefixed 40 hex chars; Solana: base58 32-44 chars)
- Alias, if provided, must be non-empty

---

## Entity: LendingPosition

Represents a wallet's participation in a specific lending protocol on a specific blockchain network.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| Wallet | Wallet | yes | The wallet being monitored |
| Protocol | string | yes | Lending protocol identifier (e.g., "aave", "kamino") |
| Network | string | yes | Blockchain network identifier (e.g., "ethereum", "solana") |

**Validation Rules**:
- Protocol must be one of the supported values
- Network must be one of the supported values
- Wallet must be valid for the specified network

**State Transitions**: None — this entity is immutable once defined

---

## Entity: HealthFactor

Represents the health metric returned by a lending protocol for a specific position.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| Value | float64 | yes | Numeric Health Factor (1e18-scaled uint256 converted to decimal) |
| Classification | string | yes | Derived health state: "safe", "warning", or "critical" |

**Validation Rules**:
- Value must be a positive number
- Classification must be one of: "safe" (>1.5), "warning" (1.0–1.5), "critical" (≤1.0)

**State Transitions**: None — value is retrieved, not mutated

---

## Entity: ProviderResult

Represents the outcome of attempting to retrieve a Health Factor from a provider for a specific position.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| Position | LendingPosition | yes | The position that was queried |
| HealthFactor | HealthFactor | no | Present only on success |
| Error | string | no | Present only on failure; human-readable error description |

**Validation Rules**:
- Exactly one of HealthFactor or Error must be populated (Result or Error, never both, never neither)

---

## Entity: Configuration

Represents the user's wallet and network configuration loaded from the config file.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| RPCEndpoints | map[string]string | yes | Network → RPC URL mapping (keys: "ethereum", "solana") |
| Positions | []LendingPosition | yes | At least one position to monitor |

**Validation Rules**:
- At least one RPC endpoint must be defined
- Each referenced network must have a corresponding RPC endpoint
- At least one lending position must be defined
- All positions must pass their individual validation rules

---

## Relationships

```
Configuration (1) ──has many──> LendingPosition (N)
LendingPosition (1) ──references──> Wallet (1)
LendingPosition (1) ──produces──> ProviderResult (1) [per execution]
ProviderResult (1) ──contains 0..1──> HealthFactor (0..1)
```
