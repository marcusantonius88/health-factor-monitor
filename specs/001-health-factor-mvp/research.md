# Research: Health Factor Monitor MVP

## Blockchain RPC Interaction Pattern

### Decision
Use raw JSON-RPC calls over HTTP for both Ethereum and Solana. No blockchain SDK dependencies.

### Rationale
- Keeps business logic independent of third-party SDKs (Constitution requirement)
- No heavyweight dependencies like `go-ethereum` (~50MB dependency tree)
- Both Ethereum and Solana expose standard JSON-RPC interfaces
- Standard `net/http` and `encoding/json` are sufficient
- Adding an SDK provides minimal value for the read-only, single-call pattern needed here

### Alternatives Considered
- **go-ethereum**: Would add significant dependency weight; couples domain to Geth types
- **solana-go**: Similar weight and coupling concerns
- **GraphQL/subgraph**: Adds infrastructure dependency not justified for MVP scope
- **gRPC**: Over-engineered for simple read-only queries

---

## Config File Format

### Decision
JSON format for wallet configuration.

### Rationale
- `encoding/json` is in the Go standard library — zero external dependencies
- Simple to parse, well-understood, easy to generate manually
- Sufficient expressiveness for wallet lists and provider mappings
- Aligns with "prefer standard Go libraries" directive

### Alternatives Considered
- **YAML**: Requires `gopkg.in/yaml.v3` dependency; more complex parsing rules
- **TOML**: Requires third-party library; less familiar to most users
- **HCL**: Requires third-party library; over-engineered for flat wallet configs
- **Environment variables**: Impractical for multi-wallet config; harder to validate

### Format Design
```json
{
  "rpc_endpoints": {
    "ethereum": "https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY",
    "solana": "https://api.mainnet-beta.solana.com"
  },
  "positions": [
    {
      "alias": "my-aave-wallet",
      "address": "0x...",
      "network": "ethereum",
      "protocol": "aave"
    },
    {
      "alias": "my-kamino-wallet",
      "address": "...",
      "network": "solana",
      "protocol": "kamino"
    }
  ]
}
```

---

## Aave Health Factor Retrieval

### Decision
Call `getUserAccountData` via `eth_call` on the Aave V2/V3 Pool contract.

### Rationale
- `getUserAccountData` returns `(totalCollateralBase, totalDebtBase, availableBorrowsBase, currentLiquidationThreshold, ltv, healthFactor)` in a single call
- Health Factor is returned directly as a uint256 scaled by 1e18
- No event log scanning or state reconstruction needed
- Works for both Aave V2 and V3 (slight contract address difference)

### Implementation Notes
- Aave V3 Ethereum Pool: `0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2`
- `getUserAccountData` selector: `0x5c7783a3`
- Input: user address (32-byte padded)
- Output: decode 6 uint256 values from RLP (healthFactor is index 5)

---

## Kamino Health Factor Retrieval

### Decision
Call `getUserAccount` via the Kamino lending program on Solana.

### Rationale
- Kamino stores user account data in Program Derived Addresses (PDAs)
- Each user position has an on-chain account with a `healthFactor` field
- Solana getAccountInfo returns the raw account data buffer
- Parse the account structure using known field offsets

### Implementation Notes
- Kamino program ID: `KLend...` (specific Kamino lending program ID)
- Derive PDA using user wallet pubkey and market/obligation seeds
- Account data layout contains health factor as a `u128` (16 bytes) at a known offset
- Need to confirm exact account structure with Kamino IDL or source

---

## Health Classification Thresholds

### Decision
Three-tier classification with configurable thresholds:
| Classification | Range | Visual |
|---------------|-------|--------|
| Safe | > 1.5 | Green |
| Warning | 1.0 – 1.5 | Yellow |
| Critical | ≤ 1.0 | Red |

### Rationale
- Industry-standard thresholds used by Aave, Compound, and major DeFi position managers
- 1.0 = liquidation boundary (universal across lending protocols)
- 1.5 = common "safe" threshold; provides 50% buffer before liquidation
- Configurable per FR-008, but these defaults match user expectations

### Alternatives Considered
- **Single numeric output**: Fails User Story 4 (quick understanding)
- **Continuous gradient**: More complex, harder to read in CLI output
- **Protocol-specific thresholds**: Adds complexity; MVP should standardize

---

## Go Version — Feature Decisions

### Decision
Target Go 1.22. Use standard `flag` package for CLI parsing.

### Rationale
- Go 1.22 is the current stable release widely available in CI/CD and package managers
- Standard `flag` avoids external CLI framework dependency
- No need for advanced CLI features (subcommands, autocomplete) in MVP
- `slog` package (Go 1.21+) available for structured logging if needed

### Alternatives Considered
- **Go 1.21**: Acceptable, but Go 1.22 is current
- **spf13/cobra**: Adds dependency for CLI framework; not justified for single-command MVP
- **spf13/viper**: Adds dependency for config loading; over-engineered for simple JSON file
