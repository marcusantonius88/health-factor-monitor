# Data Model: Required Contribution Estimate

## Entity: Position Snapshot

One coherent read of a supported lending position, used for both its displayed Health Factor and its optional contribution estimate.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| Position | LendingPosition | yes | Configured Aave/Base or Kamino/Solana position |
| HealthFactor | HealthFactor | yes on successful position read | Current protocol Health Factor |
| ContributionBasis | ContributionBasis | no | Normalized data needed for an estimate; omitted if unavailable |
| ContributionUnavailableReason | string | no | Human-readable reason the estimate cannot be safely computed |
| Snapshot time/state | protocol-specific metadata | yes for estimate | Chain block or API timestamps/slot and relevant errors |

**Validation Rules**:
- A missing contribution basis does not invalidate a successfully retrieved Health Factor.
- A contribution basis must be tied to the same obligation/account and market as the returned Health Factor.
- A stale, internally inconsistent, or incomplete snapshot cannot produce a numeric estimate.

## Entity: Contribution Basis

Protocol-neutral inputs for calculating the Health Factor effect of adding the selected collateral asset.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| CurrentHealthFactor | positive decimal | yes | Position Health Factor from the same snapshot |
| EffectiveDebtUSD | positive decimal | yes | Debt in USD using the protocol's Health Factor denominator, including applicable adjustments |
| IncrementalLiquidationFactor | positive decimal | yes | Increase in liquidation-adjusted collateral capacity per USD of the selected asset under position-specific risk rules |
| Asset | ContributionAsset | yes | Verified eligible BTC-denominated asset or SOL reserve/mint |
| PriceUSD | positive decimal | yes | Current protocol-oracle price for the exact asset/mint |
| AssetPrecision | non-negative integer | yes | Smallest supported asset unit used for upward rounding |
| PriceFreshness | validated timestamp/state | yes | Evidence that the price and position data meet the source's freshness policy |
| Active risk group | protocol-specific metadata | when applicable | Verified Aave eMode or Kamino elevation-group state used for the asset's effective liquidation factor |

**Validation Rules**:
- `CurrentHealthFactor` must be below 1.50 before an estimate is attempted.
- `EffectiveDebtUSD`, `IncrementalLiquidationFactor`, and `PriceUSD` must be positive and finite.
- The asset must be active and eligible to contribute collateral for the selected position.
- Applicable user eMode/elevation-group risk parameters must be verified; a hypothetical pair's default/auto-selected group is not proof of the existing position's active group.
- Missing or unverifiable data results in an unavailable estimate; it is not replaced with a default.

## Entity: Contribution Asset

The particular collateral representation used to price and risk-weight the estimate.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| Symbol | string | yes | `BTC` or `SOL` for CLI display |
| Market asset identifier | address/mint | yes | Underlying Aave reserve or Kamino reserve/mint |
| Liquidation parameters | protocol-specific data | yes | Effective liquidation threshold/terms for the position |

**Validation Rules**:
- Aave/Base asset identity must be from the recognized Aave market reserve set and eligible under the account configuration.
- If multiple BTC reserves are eligible, evaluate each and use the largest valid rounded USD estimate.
- If no BTC reserve is eligible or has valid price/risk data, no asset is selected.
- Kamino/Solana asset must be the SOL collateral reserve in the same lending market, with applicable borrow/elevation terms.
- Display symbols are labels only and do not establish eligibility.

## Entity: Contribution Estimate

The informational contribution result displayed by the CLI.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| Status | enum | yes | `available` or `unavailable` |
| AssetSymbol | string | yes | `BTC` or `SOL` |
| AmountUSD | non-negative decimal | only when available | USD value, rounded upward to cents after asset-precision rounding |
| Reason | string | only when unavailable | Clear short explanation without a numeric amount |
| TargetHealthFactor | decimal | yes | Fixed target `1.50` |

**Validation Rules**:
- Safe positions do not receive a contribution estimate object or CLI line.
- Available amounts must be positive for a position below target and yield projected HF >= 1.50 after rounding.
- Unavailable estimates contain no numeric amount.
- Token quantity and unit price are not exposed in CLI output.

## Calculation Relationship

The provider translates its native collateral/debt/risk model into a contribution basis. The shared estimator finds the minimum USD contribution that closes the liquidation-capacity deficit to target 1.50, rounds up at asset precision and USD display precision, and validates the projected result. Protocol values that cannot be mapped faithfully to this basis must fail closed.
