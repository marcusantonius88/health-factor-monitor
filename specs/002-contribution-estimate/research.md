# Research: Required Contribution Estimate

**Date**: 2026-11-17

## Summary

Use each protocol's own market data path. Aave's market oracle and reserve configuration are the source of truth for Aave/Base. Kamino's official portfolio and oracle-price endpoints provide obligation-level inputs plus explicit timestamps and an allowed price age. Keep protocol reads and mappings inside their existing providers, normalize only the risk inputs needed by a common pure estimator, and fail closed when any required datum is missing or stale.

## Aave V3 on Base

### Decision

Use the Aave market oracle and reserve configuration associated with the existing Base Pool. Obtain the asset price in the market's base currency and confirm that the market base unit represents USD. Read the candidate BTC reserve's effective liquidation threshold and the user's collateral/eMode eligibility. Resolve the BTC underlying from the official market reserve set; do not infer identity from an arbitrary token ticker.

### Rationale

- The official [Aave V3 Pool documentation](https://aave.com/docs/aave-v3/smart-contracts/pool) describes the Pool as market-specific and exposes `getUserAccountData`, which already supplies the current Health Factor and aggregate debt/collateral information.
- The [Pool Addresses Provider documentation](https://aave.com/docs/aave-v3/smart-contracts/pool-addresses-provider) states that it is the market registry and specifically recommends resolving the Pool through it. Its interface also exposes the market's price-oracle and pool-data-provider references.
- The [Aave Oracle documentation](https://aave.com/docs/aave-v3/smart-contracts/oracles) documents `getAssetPrice(asset)`, its base-currency denomination, and the USD-market base unit of 8 decimals. This is more authoritative for Aave risk calculations than an unrelated market-price service.
- The [Aave `IPriceOracleGetter` interface](https://github.com/aave-dao/aave-v3-origin/blob/8305565ae342f1773c42cd2e4593f175fe5968a0/src/contracts/interfaces/IPriceOracleGetter.sol) exposes `BASE_CURRENCY`, `BASE_CURRENCY_UNIT`, and `getAssetPrice`; USD is represented by the zero-address base currency and `1e8` unit. Read these values rather than assuming them.
- The [Aave `IPoolDataProvider` interface](https://github.com/aave-dao/aave-v3-origin/blob/8305565ae342f1773c42cd2e4593f175fe5968a0/src/contracts/interfaces/IPoolDataProvider.sol) exposes live reserve enumeration/configuration and user reserve data. The implementation separates reserve-level collateral capability (non-zero liquidation threshold) from the user's collateral-enabled flag.
- Aave Health Factor uses liquidation-threshold-weighted collateral divided by debt. The amount contributed therefore depends on the selected reserve's effective liquidation threshold and the user's active eMode/collateral configuration, not just the current Health Factor.
- Fetching account data, asset price, and reserve configuration at a consistent recent chain state avoids mixing incompatible snapshots.

### BTC asset selection

The [Aave V3 Base Address Book](https://github.com/aave-dao/aave-address-book/blob/266429eefae828d52c91ffcb3a75656ab18f7160/src/AaveV3Base.sol) identifies the Base `PoolAddressesProvider` at `0xe20fCBdBfFC4Dd138cE8b2E6FBb6CB49777ad64D`; use it as the registry entry point, then resolve current Pool, oracle, and data-provider references on-chain. The address book and [TypeScript market exports](https://github.com/aave-dao/aave-address-book/blob/266429eefae828d52c91ffcb3a75656ab18f7160/src/ts/AaveV3Base.ts) list cbBTC, LBTC, and tBTC candidate reserves. These addresses are only a candidate inventory: the market reserve list and live configuration must confirm the assets remain available, supplyable, and usable as collateral for the position.

If exactly one candidate is eligible, use it. If several candidates are eligible, compute the estimate for each and choose the largest rounded USD amount. This conservative selection supports the generic BTC label without implying a specific token quantity or representation. If none has a valid quote and applicable risk data, report unavailable.

### Freshness and failure

The Aave market oracle is the protocol-relevant price, but its `getAssetPrice` return does not itself include a timestamp. The official AaveOracle implementation uses its configured source and may fall back to a market fallback oracle, so the oracle quote alone does not prove that the underlying source is fresh. Resolve and validate the source where its interface permits a timestamp check; if the source is stale, unsupported, or freshness cannot be established safely, do not publish an estimate. A failed auxiliary oracle/configuration read must not erase a successfully retrieved Health Factor.

All required account, price, reserve, and eMode reads should use the same pinned block. Check reserve paused/frozen/active state and supply caps in addition to collateral threshold. Use both the reserve-level collateral eligibility and the user/account-specific configuration; they are not interchangeable.

The relevant Aave calculation, using `C` and `D` as account collateral/debt in the market base unit, `LTcurrent` as the account's weighted liquidation threshold and `LTadd` as the selected asset's effective threshold (basis points), is:

```text
additionalBase = max(0, (1.50 * D * 10,000 - C * LTcurrent) / LTadd)
```

Convert from market base units to USD only after confirming the oracle base currency/unit. For the added token's rounding, convert the resulting USD-equivalent collateral through the verified asset price and token decimals, round up, then validate the projected HF from the same risk inputs. A zero debt or non-positive `LTadd` cannot produce a below-target estimate.

### Alternatives considered

- **Third-party exchange/aggregator price**: Rejected because it may differ from the Aave market's pricing source and would introduce an independent price dependency.
- **Hard-coded BTC/USD quote or liquidation threshold**: Rejected because both asset availability and risk parameters are market-governed and can change.
- **Use only `getUserAccountData`'s weighted liquidation threshold**: Insufficient for the incremental contribution of a particular BTC reserve, especially where eMode changes the effective threshold.
- **Trust the Address Book without live market reads**: Rejected because listed assets and configuration may no longer be active or eligible.

## Kamino on Solana

### Decision

Use Kamino's official `GET /portfolio/{pubkey}` response to identify the active lending obligation, and `GET /oracles/prices?markets=all` for the exact market and SOL mint's protocol oracle quote. The `all` response groups prices by `lendingMarket`, so match both market and mint. Advance the portfolio's lending `minSlot` to the current slot read from the configured Solana RPC and require `appliedSlot` to meet it. Validate the oracle item's timestamp against its `maxAgeInSeconds` independently.

Use SOL collateral/borrow terms only when the exact reserve eligibility and active elevation-group parameters applicable to that obligation are known. The documented portfolio schema does not expose the active elevation-group ID, and the pair endpoint's auto-selected group does not prove which group the existing obligation currently uses. If that state cannot be established from a documented official data source, mark the estimate unavailable instead of assuming a group or risk parameter. Do not add a TypeScript SDK runtime dependency or port account decoding into Go merely to force a numeric result.

### Rationale

- The official [Kamino Public API OpenAPI specification](https://api.kamino.finance/openapi/json?openapi=3.0.0) documents `/portfolio/{pubkey}` and per-position lending details. A lending row contains the market and obligation identifiers; total deposit/borrow USD values; LTV and liquidation LTV; per-deposit reserve, mint, USD price/value, and liquidation LTV; and per-borrow reserve, value, price, and borrow factor.
- The portfolio lending section provides `positionsRefreshedOn`, `pricesRefreshedOn`, `appliedSlot`, and section errors. Its documented `minSlot` query can wait up to 10 seconds for requested sections to reflect at least the supplied slot; `appliedSlot` covers position rows, not prices. Read the current Solana slot and request `minSlot` for the `lending` section; reject a response that has not caught up or reports section errors.
- The same official specification documents `/oracles/prices`; with `markets=all`, each market group identifies its `lendingMarket` and contains price items with mint, price, timestamp, and `maxAgeInSeconds`. The SOL quote must match both the obligation's market and canonical SOL mint. The endpoint supplies an explicit freshness bound for the contribution asset price.
- `/markets/collateral-reserves` documents the lending market, collateral reserve and mint, supported borrow reserves, and pair-specific `liquidationLtv`/`borrowFactor` terms with elevation-group metadata. This is necessary to verify SOL eligibility and use risk parameters that apply to the existing debt.
- The API is unauthenticated but rate-limited; reuse/couple requests per check rather than issuing one market-wide request per field or per calculation.

### Matching and calculation inputs

The existing provider selects the first active-borrow obligation from a wallet's portfolio. Preserve that selection for this feature, and use that exact obligation and market for all estimate inputs. Do not let an independently queried estimator pick a different obligation. Use the canonical SOL mint and the market's actual reserve, not a symbol match. Include borrow factors and pair/elevation-group-specific collateral terms when normalizing effective debt and the incremental liquidation capacity of SOL.

Kamino's own SDK/on-chain implementation confirms that liquidation uses borrow-factor-adjusted debt and liquidation-threshold-weighted collateral; the unhealthy borrow value is compared against the adjusted debt. For debt `D`, current unhealthy borrow value `U`, target `T`, and the selected SOL collateral liquidation factor `t`, the required USD-equivalent SOL collateral is:

```text
additionalUSD = max(0, (T * D - U) / t)
```

This is equivalent to the shared formula when current `HF = U / D`. The official [Kamino KLend SDK obligation calculation](https://github.com/Kamino-Finance/klend-sdk/blob/6a26d6f1b5afb29953ffd139ee8c0c4ca5f26ce6/src/classes/obligation.ts#L1484-L1617) and [on-chain unhealthy-value comparison](https://github.com/Kamino-Finance/klend/blob/686913ee421e0c5f3dc44c4c8344461ee0e4fb89/programs/klend/src/lending_market/lending_operations.rs#L1680-L1699) are reference sources for parity tests, not runtime dependencies.

The portfolio API does not expose the existing obligation's active elevation group, while pair terms can report the group that a hypothetical pair would auto-select. Those values must not be conflated. The official [`GET /kamino-market/{marketPubkey}/users/{userPubkey}/obligations`](https://api.kamino.finance/openapi/json?openapi=3.0.0#/paths/~1kamino-market~1%7BmarketPubkey%7D~1users~1%7BuserPubkey%7D~1obligations/get) has no typed response schema. Use it only if the response's exact group and risk fields can be verified; otherwise an estimate for that obligation is unavailable.

### Freshness and failure

Reject an absent/zero/non-finite price, a price older than its API-provided `maxAgeInSeconds`, a lending section that has not applied the requested current slot, section errors, or incomplete/ambiguous obligation or reserve fields. Return the unavailable-estimate state while retaining the Health Factor if the latter remains valid. Do not substitute portfolio token prices from a different obligation/market or an unrelated third-party quote.

### Alternatives considered

- **Use only the current Kamino aggregate `ltv` and `liquidationLtv`**: Insufficient to verify the SOL reserve's eligibility or its pair-specific liquidation threshold.
- **Directly decode all Kamino account layouts through Solana RPC**: Rejected as the default because it adds protocol-specific binary parsing and maintenance. If the documented API cannot expose the exact active elevation group, prefer an unavailable estimate over an unverified assumption; reconsider a narrowly scoped on-chain read only if product requirements demand estimates for those cases.
- **Assume the pair endpoint's auto-selected elevation group is the current obligation's group**: Rejected because the API describes this as the group a hypothetical pair would choose, not the existing obligation's state.
- **Use the default `markets=main` quote for every obligation**: Rejected because the user's obligation may belong to another Kamino market.
- **Use an external SOL/USD price feed**: Rejected in favor of the protocol's own oracle quote and freshness metadata.

## Shared Calculation

### Decision

Normalize protocol snapshots into:

- current Health Factor;
- effective debt USD in the protocol's Health Factor denominator;
- the USD increase in liquidation-adjusted collateral per USD of the selected contribution asset;
- positive USD asset price and token precision;
- source freshness/eligibility validation status.

For target `T = 1.50`, estimate the minimum USD amount from the health-factor deficit and effective debt, divided by the protocol-normalized incremental liquidation factor. Convert the result to the smallest representable asset amount by rounding upward using the current asset price and precision; convert it back to USD and round the displayed cents upward. Re-evaluate the projected Health Factor after rounding and refuse the estimate if it remains below 1.50.

### Rationale

This centralizes the common arithmetic and boundary/rounding behavior while letting Aave and Kamino provide different effective debt, collateral eligibility, threshold, freshness, and price semantics. It avoids duplicated formula implementations without incorrectly pretending the protocols have identical risk models.

Use standard-library exact/scaled arithmetic where practical. Do not add a general-purpose oracle framework or decimal dependency until a concrete need is demonstrated.

### Alternatives considered

- **Calculate separately in each provider**: Rejected because the target boundary, rounding, result validation, and error mapping would drift.
- **Assume a dollar of collateral always increases Health Factor by the same amount**: Rejected because the selected reserve's liquidation threshold and Kamino borrow/elevation terms affect the result.

## Source References

- Aave Pool: https://aave.com/docs/aave-v3/smart-contracts/pool
- Aave Pool Addresses Provider: https://aave.com/docs/aave-v3/smart-contracts/pool-addresses-provider
- Aave V3 Oracle: https://aave.com/docs/aave-v3/smart-contracts/oracles
- Aave V3 Base Address Book: https://github.com/aave-dao/aave-address-book/blob/266429eefae828d52c91ffcb3a75656ab18f7160/src/AaveV3Base.sol
- Aave V3 Base token exports: https://github.com/aave-dao/aave-address-book/blob/266429eefae828d52c91ffcb3a75656ab18f7160/src/ts/AaveV3Base.ts
- Aave Pool Data Provider interface: https://github.com/aave-dao/aave-v3-origin/blob/8305565ae342f1773c42cd2e4593f175fe5968a0/src/contracts/interfaces/IPoolDataProvider.sol
- Kamino Public API documentation landing page: https://api.kamino.finance/
- Kamino Public API OpenAPI: https://api.kamino.finance/openapi/json?openapi=3.0.0
- Kamino API machine-readable overview: https://api.kamino.finance/llms.txt
