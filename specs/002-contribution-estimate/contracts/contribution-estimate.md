# Contribution Estimate Contract

## Application Behavior

The existing Health Factor provider contract remains unchanged. The Aave/Base and Kamino/Solana adapters additionally expose a narrow optional snapshot capability for contribution estimation.

The optional capability:

- Uses the same configured position, network, account/obligation, and market as the Health Factor result.
- Retrieves the Health Factor first and performs no estimate-only reads if it is already >= 1.50.
- Returns a valid Health Factor independently of estimate availability.
- Supplies protocol-normalized risk inputs only when the asset is eligible and source data are sufficiently current and complete.
- Represents an estimate-input failure separately from a Health Factor retrieval failure.
- Respects `context.Context`; does not submit transactions.
- If it must refresh the Health Factor while gathering contribution data, returns that refreshed value so the CLI and estimate describe the same snapshot.

The application service:

1. Uses the optional position-snapshot capability when available; otherwise retrieves Health Factor through the existing provider contract.
2. Skips estimate calculation entirely when HF >= 1.50.
3. For supported Aave/Base and Kamino/Solana snapshots with HF < 1.50, calls the shared deterministic calculator.
4. Attaches either an available USD estimate or an unavailable reason to that position result.
5. Continues processing other positions if estimate-only data or calculation fails.

Providers that do not implement the optional estimate capability continue to use the existing Health Factor flow unchanged.

## Normalized Calculation Contract

Inputs include current HF `H`, effective debt in USD `D`, selected asset's incremental liquidation factor `L`, current asset USD price `P`, and asset precision. For target `T = 1.50`, the required unrounded USD collateral is:

```text
max(0, (T - H) * D / L)
```

The protocol adapter is responsible for normalizing `D` and `L` so the result matches the protocol's position-specific Health Factor semantics, including reserve eligibility, liquidation threshold, eMode/elevation-group configuration, and any effective borrow factor.

The calculator rounds the corresponding asset amount upward to the asset's smallest supported unit, converts that rounded quantity to USD using `P`, rounds the displayed USD value upward to cents, and verifies the projected HF is at least 1.50. Invalid/non-finite values, unverified freshness, or a failed post-rounding target check result in an unavailable estimate, never a success-shaped fallback.

## CLI Output Contract

Keep the existing `Health Factor` heading and network row. For a supported position with HF < 1.50, append exactly one indented line directly beneath its network row:

```text
Base:   🟨 1.32
        Aporte necessário (BTC): US$ 850,00
```

or, for an unavailable result:

```text
Base:   🟨 1.32
        Aporte necessário (BTC): indisponível (motivo)
```

Use `SOL` for Kamino/Solana. Do not show the line for HF >= 1.50. Do not print token quantity or unit price. Estimate unavailability does not change successful Health Factor handling or process exit status.
