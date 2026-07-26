# Quickstart: Health Factor Monitor MVP

## Prerequisites

- Go 1.22+
- Network access to Ethereum and Solana RPC endpoints
- One or more wallet addresses with active positions on Aave (Ethereum) or Kamino (Solana)

## Setup

```bash
# Clone and build
git clone <repo-url>
cd health-factor-monitor
go build -o hfmon ./cmd/hfmon

# Create config file
cat > config.json << 'EOF'
{
  "rpc_endpoints": {
    "ethereum": "https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY",
    "solana": "https://api.mainnet-beta.solana.com"
  },
  "positions": [
    {
      "alias": "my-aave",
      "address": "0xYOUR_ETH_WALLET",
      "network": "ethereum",
      "protocol": "aave"
    },
    {
      "alias": "my-kamino",
      "address": "YOUR_SOL_WALLET",
      "network": "solana",
      "protocol": "kamino"
    }
  ]
}
EOF
```

## Validation Scenarios

### Scenario 1: All Providers Healthy

```bash
./hfmon -config config.json
```

**Expected Output**:
```
Position               Protocol    Network     Health Factor    Status
──────────────────────────────────────────────────────────────────────
my-aave                Aave        Ethereum    2.34             ✅ Safe
my-kamino              Kamino      Solana      1.82             ✅ Safe
```

**Exit Code**: `0`

---

### Scenario 2: Partial Provider Failure

Simulate by providing an invalid RPC URL for one network.

**Expected Output**:
```
Position               Protocol    Network     Health Factor    Status
──────────────────────────────────────────────────────────────────────
my-aave                Aave        Ethereum    2.34             ✅ Safe
my-kamino              Kamino      Solana      —                ❌ Error: failed to connect to RPC at https://invalid.example.com
```

**Exit Code**: `0` (at least one succeeded)

---

### Scenario 3: Total Failure

Simulate by providing invalid RPC URLs for both networks.

**Expected Output**:
```
Position               Protocol    Network     Health Factor    Status
──────────────────────────────────────────────────────────────────────
my-aave                Aave        Ethereum    —                ❌ Error: failed to connect to RPC
my-kamino              Kamino      Solana      —                ❌ Error: failed to connect to RPC
```

**Exit Code**: `1` (all failed)

---

### Scenario 4: Configuration Error

```bash
./hfmon -config nonexistent.json
```

**Expected Output**:
```
Error: cannot read config file "nonexistent.json": file does not exist
```

**Exit Code**: `1`

---

### Scenario 5: Invalid Wallet Address

```bash
./hfmon -config invalid-address-config.json
```

**Expected Output**:
```
Error: position "my-aave": invalid Ethereum address "0xbad"
```

**Exit Code**: `1`

---

## Running Tests

```bash
# Unit tests for domain and application layers
go test ./internal/domain/...
go test ./internal/application/...

# Integration tests (requires configured RPC endpoints)
go test ./tests/integration/...
```

## References

- [Data Model](data-model.md) — entity definitions and validation rules
- [Provider Interface Contract](contracts/provider-interface.md) — provider interface contract
- [Application Service Contract](contracts/application-service.md) — service interface contract
- [Configuration Contract](contracts/config-contract.md) — config file format and loader contract
- [Research Notes](research.md) — technology decisions and rationale
