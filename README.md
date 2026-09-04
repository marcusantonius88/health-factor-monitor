<div align="center">

# Health Factor Monitor

**Monitor decentralized lending positions on Aave and Kamino directly from the terminal.**

A Go CLI to query lending positions' Health Factor, rank risk, and present results clearly — built with fault tolerance and an extensible provider architecture.

</div>

---

## 🎯 Problem

In the DeFi ecosystem, the Health Factor is a core metric to assess the risk of a credit position. In protocols such as Aave and Kamino, a position's health can change quickly due to market conditions, volatility, and leverage.

For users managing multiple positions across protocols, tracking this metric manually is slow, error-prone, and often suffers from integration issues.

Health Factor Monitor solves this by providing a simple, automated experience to:

- Load positions configured in JSON;
- Validate required network entries and wallet address formats before execution;
- Query multiple protocols in a single flow;
- Display results in a readable terminal summary grouped by network;
- Maintain robustness even when some providers fail.

---

## 🚀 Delivered MVP

The MVP focuses on productivity and reliability and has already been implemented.

### Current features

- Support for Aave on Base and Ethereum Mainnet;
- Support for Kamino on Solana;
- Load and validate configuration from JSON;
- Verify RPC endpoints and wallet addresses;
- Visual Health Factor indicators rendered by the CLI:
  - 🟩 Safe: HF >= 1.50
  - 🟨 Attention: 1.10 <= HF < 1.50
  - 🟥 Critical: HF < 1.10
- Run checks per protocol or for all configured positions;
- Per-position fault tolerance;
- Terminal output with emoji indicators and Health Factor values;
- Exit status `0` when at least one position succeeds, or `1` when no position succeeds.

### How the application works

The CLI reads a configuration file, validates the provided data, and then queries each position using the corresponding provider. Configuration validation checks required fields, supported protocols and networks, wallet address formats, and whether an endpoint entry exists for each position's network. It does not probe endpoint availability before querying.

If a position fails due to timeout, unavailable RPC/API, or malformed response, the rest of the positions are still processed and the program continues running. HTTP requests use a 30-second client timeout.

### Architecture

The application is designed with a clear separation of concerns between domain, application, and infrastructure.

```text
                +----------------------+
                |        CLI           |
                |      hfmon           |
                +----------+-----------+
                           |
                           v
                +----------------------+
                |   Check Service      |
                |  orchestration       |
                +----------+-----------+
                           |
        +---------------+----------------+
        |                                |
        v                                v
+-------------------+        +----------------------+
| Aave Provider     |        | Kamino Provider      |
| Ethereum/RPC      |        | Kamino API           |
+-------------------+        +----------------------+
          |
          v
+-------------------+
| Domain / Models   |
| Config / Provider |
| HealthFactor      |
+-------------------+
```

### Layers

- **Domain**: entities and business rules such as `Config`, `HealthFactor`, `ProviderResult`, and risk classification.
- **Application**: orchestrator responsible for mapping positions to providers and handling failures in a resilient way.
- **Infrastructure**: adapters for Aave and Kamino, plus the JSON configuration loader.
- **Interface**: CLI that renders a readable network-based summary with emoji indicators and defines exit status behavior.

### Main contracts

The architecture follows a provider interface pattern, enabling adding new protocols without coupling service logic to concrete protocol implementations.

---

## 🛠️ Technology Stack

### Language and Runtime

- Go
- Terminal-first CLI

### Supported Protocols and Networks

- Aave on Ethereum Mainnet
- Aave on Base (Layer 2)
- Kamino on Solana

### Integrations and Formats

- JSON-RPC for Ethereum and Base
- REST API for Kamino
- Ethereum and Solana address validation
- JSON configuration loading
- Emoji-based Health Factor classification

### Tests

- Unit tests for domain, providers, and service
- Validation of behavior for success, timeout, and error scenarios
- Coverage for classification rules, configuration, and failure recovery

---

## 🚀 How to Use

### 1. Configuration

Create a configuration file with positions and RPC endpoints.

```json
{
  "rpc_endpoints": {
    "ethereum": "https://ethereum-rpc.publicnode.com",
    "base": "https://mainnet.base.org",
    "solana": "https://api.mainnet-beta.solana.com"
  },
  "positions": [
    {
      "alias": "ethereum-borrower",
      "address": "0x168378977EDcB8B5c93025213e41cDD76e5EE058",
      "network": "base",
      "protocol": "aave"
    },
    {
      "alias": "solana-borrower",
      "address": "HX7qXRFZhgBFmJdE46BnsLEvtLdb14cBh1rMZiAA1x8C",
      "network": "solana",
      "protocol": "kamino"
    }
  ]
}
```

### 2. Run

```bash
go run ./cmd/hfmon -config ./config.json
```

### 3. Example Output

```text
Health Factor
-------------
Base:	🟩 1.97
Solana:	🟩 2.22
```

The output uses the network name rather than the configured alias or protocol. A position without an active debt is shown as `no active debt`; provider failures are shown as `HF: unavailable`.

### 4. Filter by Protocol

```bash
go run ./cmd/hfmon -config ./config.json -protocol aave
```

---

## 🧪 Validation and Robustness

The application is built with realistic operational failures in mind:

- Invalid or unavailable RPC;
- Malformed provider response;
- Query timeout;
- Invalid address;
- Unsupported protocol;
- Missing configuration or required data.

When an item fails, the system continues processing the rest without crashing. The CLI prints the unavailable result but does not print the underlying provider error. If at least one result succeeds, the process exits with status `0`; otherwise it exits with status `1`.

---

## 📁 Project Structure

```text
cmd/
  hfmon/
internal/
  application/
  domain/
  infrastructure/
    aave/
    kamino/
    config/
interfaces/
  cli/
tests/
  integration/
  testdata/
specs/
```

---

## 📄 License

This project is licensed under the MIT License.
