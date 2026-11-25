# Quickstart: Required Contribution Estimate

This guide validates the estimate path without submitting transactions or depending on live market data. Provider and calculation tests should use deterministic fixtures.

## Prerequisites

- Go 1.23 or later.
- Repository dependencies restored (the feature plan introduces no new module dependency).

## Automated Validation

Run the focused tests:

```sh
go test ./internal/domain ./internal/application ./internal/infrastructure/aave ./internal/infrastructure/kamino ./internal/interfaces/cli
```

The test fixtures should include:

- Aave/Base account data with debt, current Health Factor, candidate BTC reserve configuration, user collateral/eMode state, USD-base oracle price, and valid source freshness.
- Kamino portfolio data with multiple lending rows, a selected active obligation, per-asset deposits/borrows, a lending section caught up to the requested `minSlot`, and an oracle entry matching both its market and SOL mint including `timestamp` and `maxAgeInSeconds`.

## Required Scenarios

1. **Aave/Base below desired target**: With a uniquely eligible BTC reserve and valid snapshot, verify the displayed USD amount corresponds to rounded-up eligible BTC collateral and projected HF >= 2.00.
2. **Kamino/Solana below desired target**: With a matching obligation/market, current SOL price, and verified active elevation-group terms for the SOL reserve, verify the USD amount accounts for applicable liquidation/elevation terms and projected HF >= 2.00.
3. **Boundary and safe position**: Verify HF exactly 1.50 is classified as safe. A position below 2.00 still receives an estimate line, while HF >= 2.00 produces no estimate and does not request estimate-only data.
4. **Unavailable inputs**: Missing/stale oracle, portfolio that has not reached `minSlot`, protocol/API error, ineligible/ambiguous asset, unknown active elevation group, or invalid risk data produces an explicit unavailable line with no numeric amount.
5. **Independent processing**: An estimate-only failure on one position does not suppress another position's valid Health Factor or estimate.
6. **Compatibility**: Existing Aave/Ethereum results and existing safe-position output retain their current format and behavior.
7. **No transaction side effect**: Test servers observe only the expected read requests; no transaction-building endpoint or on-chain write method is called.

## Manual Smoke Check

After implementation, run the CLI with a configured Aave/Base and Kamino/Solana wallet. Compare the estimate against the fixture-tested projected HF and verify output labels and USD formatting. Live values are volatile; do not assert a fixed USD amount or submit any transaction as part of this check.

Refer to [the model](data-model.md), [the estimate contract](contracts/contribution-estimate.md), and [the research decisions](research.md) for accepted data and failure behavior.
