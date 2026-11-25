# Implementation Plan: Required Contribution Estimate

**Branch**: `002-contribution-estimate` | **Date**: 2026-11-17 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/002-contribution-estimate/spec.md`

## Summary

Estimate and display the USD value of BTC-denominated collateral for Aave on Base or SOL collateral for Kamino on Solana when a retrieved Health Factor is below the desired target 2.00. Classification remains Safe at HF >= 1.50. Keep protocol-specific data acquisition in each existing provider, normalize the risk inputs for one shared, deterministic estimator, and report an unavailable state rather than a numeric result when price, freshness, collateral eligibility, or risk data cannot be verified.

## Technical Context

**Language/Version**: Go 1.23 (existing module)

**Primary Dependencies**: Standard library; existing Aave JSON-RPC adapter and Kamino public API adapter. No new dependency planned.

**Storage**: N/A; estimates are calculated for each CLI run.

**Testing**: `go test` with `testing`, `httptest`, and deterministic fixtures for RPC/API responses and calculation cases.

**Target Platform**: Existing terminal application on Linux and macOS.

**Project Type**: Go CLI with provider-based architecture.

**Performance Goals**: Do not add external calls for safe positions. Keep estimate retrieval bounded by the existing external request timeout and process all configured positions independently.

**Constraints**:
- Preserve the existing `HealthFactorProvider` contract and provider separation.
- Estimate only Aave/Base and Kamino/Solana in this feature; retain existing output for other supported combinations.
- Treat HF exactly 1.50 as Seguro. The current domain classifier uses a strict `> 1.50` comparison, so implementation must align the boundary and its tests with the feature requirement.
- Use the protocol's own market price and risk parameters. No third-party price aggregator or local price cache is required.
- Never execute a transaction. If an authoritative eligible asset, positive price, fresh data, or applicable risk parameter cannot be established, no numeric estimate is produced.
- Display USD only; internal token-unit conversion is used only to account for the asset's precision and ensure the rounded amount meets the target.

**Scale/Scope**: Existing configured-position scale (approximately 1–50 wallets); the feature adds an estimate only for positions below the fixed safe threshold.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Gate I — Provider-Based Architecture
- Aave and Kamino continue to own their protocol-specific reads and normalization. ✅
- Shared estimator consumes protocol-neutral risk inputs and has no RPC/API knowledge. ✅
- The existing provider contract remains stable; estimate-specific capability is separate. ✅

### Gate II — Interface Independence
- Calculation and availability state are represented in domain/application types. ✅
- The CLI only formats the estimate returned by the application. ✅

### Gate III — Simplicity & Incremental Delivery
- Reuse existing HTTP clients, configured Aave RPC endpoints, and Kamino API integration. ✅
- No database, configurable target, trading integration, or generalized pricing framework. ✅

### Gate IV — Quality & Maintainability
- Protocol reads accept `context.Context`; all failure states are explicit. ✅
- Pure calculation is deterministic and covered by unit tests; adapters use mock HTTP/RPC fixtures. ✅

### Gate V — Extensibility
- Existing health-factor consumers need not implement contribution estimation. ✅
- Each target provider implements a narrow optional capability; adding a protocol later does not require changing the shared health-factor interface. ✅

**Result**: All gates pass. No constitutional deviations.

## Design Decisions

### Shared calculation, protocol-specific inputs

Add a small optional position-snapshot capability for Aave/Base and Kamino/Solana rather than changing `HealthFactorProvider`. The capability first retrieves the Health Factor; if it is already >= 1.50, it returns without estimate-only reads. If below target, it obtains the additional data needed to build a snapshot tied to the same selected position and market. A provider may return a valid Health Factor while marking only the contribution basis unavailable; auxiliary estimate failures must not turn a successful Health Factor into a provider failure. If a refreshed snapshot changes the Health Factor, use the snapshot value for both display and calculation.

The adapters map native protocol values into a common risk basis: current Health Factor, effective USD debt, the incremental liquidation-risk contribution of one USD of the chosen asset, USD price, token precision, and source freshness. A pure domain/application calculator computes the minimum amount, rounds upward to the smallest token unit and display cent, then validates the projected Health Factor is at least 2.00. The CLI never derives protocol formulas.

This keeps the estimator shared without forcing Aave and Kamino to share their transport, account models, or collateral rules. If a protocol snapshot cannot be represented safely by the normalized basis, that provider returns an unavailable estimate.

### Aave on Base

- Reuse the configured Base RPC and existing Aave provider.
- Start from the Aave Address Book's Base `PoolAddressesProvider` reference (`0xe20fCBdBfFC4Dd138cE8b2E6FBb6CB49777ad64D`), then resolve the current Pool, oracle, and pool data provider from that on-chain registry instead of treating copied Pool/oracle addresses as authoritative.
- Read account data, all reserve tokens, reserve configuration, user reserve data, the user's eMode state/category, and the oracle's USD-base price against one pinned chain block.
- Use the Aave Address Book only to identify recognized BTC reserve candidates; verify every candidate against the live market reserve list, current reserve configuration, supply state/caps, and applicable account eMode before use. Do not infer identity from an arbitrary ticker.
- If exactly one eligible BTC reserve exists, use it. If several are eligible, calculate each and select the most conservative (largest rounded USD estimate), so the output is safe regardless of which eligible BTC representation the user supplies. If no candidate has a valid quote and applicable risk data, report unavailable.
- Apply the candidate's effective liquidation threshold (including relevant user eMode treatment); do not use the LTV parameter for the Health Factor increment.
- Validate the USD base currency/unit, positive quote, supported source, and available freshness evidence. Reject unsupported or stale source data instead of substituting an exchange quote.

### Kamino on Solana

- Reuse the official Kamino Public API integration. Use the wallet portfolio response's matching active `lending` obligation, which includes its market, deposits, borrows, USD values, per-asset prices and liquidation parameters, plus section-level position/price freshness metadata.
- Before trusting portfolio position rows, read the current Solana slot from the configured RPC and pass it as `minSlot` with `sections=lending`; require the returned lending section's `appliedSlot` to meet that slot. Independently validate the SOL oracle quote's timestamp and `maxAgeInSeconds`, since the API documents that `minSlot` does not refresh prices.
- Select the SOL reserve/mint only from the same lending market and verify the collateral/borrow pair terms applicable to that obligation. Do not select by symbol alone.
- Use `/oracles/prices?markets=all`, match both the obligation's market and canonical SOL mint, and validate the quote timestamp against its published `maxAgeInSeconds`; cross-check the portfolio section's price freshness and errors.
- Account for the obligation's effective debt (including public borrow factors) and use the SOL deposit liquidation LTV, optionally reduced to the obligation aggregate liquidation LTV, as a conservative public fallback. The portfolio schema does not expose the active elevation-group ID, and the pair endpoint's auto-selected group is not proof of the obligation's current group. The fallback must not claim to confirm that group or substitute a pair-specific unverified term.
- The current provider's first-active-obligation behavior must be shared with the estimate path so the estimate cannot attach to another obligation.
- Missing, stale, null, inconsistent, or ambiguous portfolio/oracle values produce an unavailable estimate while retaining a valid Health Factor whenever possible.

### Numeric handling and output

Use exact/scaled decimal arithmetic for external decimal values and intermediate USD calculations where practical (`math/big` from the standard library); avoid adding a decimal package for this feature. Use the asset price and decimals to round the implied asset quantity upward internally, convert that rounded quantity back to USD, and round the displayed USD amount upward to cents. Do not expose token quantity or unit price.

Add the contribution line immediately after the existing line for a Base/Solana position below desired target 2.00. Preserve existing labels and output for positions at/above 2.00, provider failures, and non-target networks. A failed estimate is distinct from a failed Health Factor and must not affect process exit status when the Health Factor succeeded.

Align the existing Health Factor classification boundary so HF exactly 1.50 is Seguro, as specified. Keep this correction limited to the threshold behavior and cover it with a focused domain test.

## Project Structure

### Documentation (this feature)

```text
specs/002-contribution-estimate/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/
    └── contribution-estimate.md
```

### Source Code (repository root)

```text
internal/
├── domain/
│   ├── healthfactor.go                 # Existing Health Factor and result types
│   └── contribution.go                 # Planned estimate/basis types and pure calculation
├── application/
│   └── service.go                      # Estimate only after successful HF < 1.50
├── infrastructure/
│   ├── aave/
│   │   └── provider.go                 # Aave/Base snapshot, BTC reserve, oracle and risk normalization
│   └── kamino/
│       └── provider.go                 # Matching obligation, SOL reserve/oracle and risk normalization
└── interfaces/
    └── cli/
        └── app.go                      # Render available/unavailable estimate lines

internal/domain/*_test.go
internal/application/service_test.go
internal/infrastructure/{aave,kamino}/provider_test.go
internal/interfaces/cli/app_test.go
```

**Structure Decision**: Extend existing hexagonal layers in place. The optional snapshot/estimate capability is a domain port; its protocol implementations stay in the current infrastructure providers. Do not add a separate application, service, price-provider package, or transaction-capable component.

## Complexity Tracking

No Constitution violations found; no exception is required.
