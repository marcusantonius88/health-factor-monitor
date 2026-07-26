# Configuration Contract

## Purpose

Defines the configuration file format and the contract for loading/validating it.

## Config File Format (JSON)

```json
{
  "rpc_endpoints": {
    "ethereum": "<RPC_URL>",
    "solana": "<RPC_URL>"
  },
  "positions": [
    {
      "alias": "<optional display name>",
      "address": "<wallet address>",
      "network": "ethereum | solana",
      "protocol": "aave | kamino"
    }
  ]
}
```

## Config Loader Interface (Go)

```go
// ConfigLoader reads and validates wallet configuration.
type ConfigLoader interface {
    // Load reads configuration from the configured source path.
    // Returns an error if the file cannot be read, is malformed, or fails validation.
    Load() (*Config, error)
}
```

## Contract Rules

1. **`rpc_endpoints`**: Must contain at least one entry. Keys must match `network` values used in `positions`.
2. **`positions`**: Must be a non-empty array. Each entry must specify `address`, `network`, and `protocol`.
3. **Validation**: A malformed file, missing required fields, or unsupported protocol/network values MUST result in a clear error before any Health Factor retrieval is attempted.
4. **Default path**: The application SHOULD look for a default config file path (e.g., `./config.json` or a path specified via CLI flag).
