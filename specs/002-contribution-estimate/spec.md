# Feature Specification: Required Contribution Estimate

**Feature Branch**: `002-contribution-estimate`

**Created**: 2026-11-17

**Status**: Draft

**Input**: User description: "Adicionar ao Health Factor Monitor a capacidade de calcular e apresentar o valor estimado, em dólares (USD), que precisa ser aportado em criptomoedas para que uma posição volte à classificação Seguro (HF >= 1,50). Para Aave na Base, considerar BTC na representação compatível com os ativos aceitos como garantia na posição. Para Kamino na Solana, considerar SOL. Apresentar somente quando HF < 1,50 e não executar transações. Se não for possível calcular com segurança, comunicar a indisponibilidade sem apresentar um valor incorreto."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Estimate Contribution to Restore Safe Status (Priority: P1)

As a user monitoring lending positions, I want to see the estimated USD value of BTC or SOL that I would need to add as collateral when a position is below the safe Health Factor threshold, so that I can make an informed decision about restoring its safe classification.

**Why this priority**: This is the central user benefit of the feature: turning a warning or critical Health Factor into an actionable, protocol-aware estimate.

**Independent Test**: Provide positions with Health Factors below 1.50 and sufficient current price and protocol risk data. Verify that Aave on Base displays an estimated BTC contribution and Kamino on Solana displays an estimated SOL contribution, each sufficient to reach at least 1.50 under the applicable position parameters.

**Acceptance Scenarios**:

1. **Given** an Aave position on Base with HF below 1.50 and a BTC-denominated collateral asset eligible for that position, **When** the CLI displays the position, **Then** it displays the estimated USD contribution in BTC needed to reach HF >= 1.50.
2. **Given** a Kamino position on Solana with HF below 1.50 and SOL eligible as collateral, **When** the CLI displays the position, **Then** it displays the estimated USD contribution in SOL needed to reach HF >= 1.50.
3. **Given** a position with HF equal to or above 1.50, **When** the CLI displays the position, **Then** it is classified as Seguro and no contribution estimate is displayed.
4. **Given** a position below 1.50, **When** the CLI displays its network summary, **Then** the contribution information appears directly below that network's Health Factor and is expressed only in USD with the corresponding asset label.

### User Story 2 - Understand When an Estimate Is Unavailable (Priority: P2)

As a user monitoring a position below the safe threshold, I want the CLI to say when a reliable contribution estimate cannot be produced, so that I am not misled by a fabricated or unsafe amount.

**Why this priority**: Accurate estimates depend on current market and protocol data. An explicit unavailable state is safer than an unsupported number.

**Independent Test**: Use positions below 1.50 with missing, stale, invalid, or insufficient price or protocol parameters, or without an eligible BTC/SOL collateral asset. Verify that the CLI reports that the estimate is unavailable and displays no numeric contribution.

**Acceptance Scenarios**:

1. **Given** a position below 1.50 and unavailable or insufficient data needed to estimate the contribution, **When** the CLI displays the position, **Then** it communicates that the estimate could not be calculated and does not show a numeric amount.
2. **Given** a position below 1.50 and no supported BTC collateral asset available for the Aave position, **When** the CLI displays the position, **Then** it communicates that the estimate is unavailable and does not substitute a different asset.
3. **Given** a position below 1.50 with a calculable estimate, **When** the CLI completes the check, **Then** it only reports the estimate and does not buy, deposit, or otherwise transact.

### Edge Cases

- The Health Factor is exactly 1.50; no contribution information is shown.
- The position's Health Factor is unavailable or invalid; no estimate is shown.
- A required asset price is unavailable, invalid, or not current enough to be considered reliable.
- Protocol risk parameters or position data required for the estimate are missing or inconsistent.
- The position has no eligible BTC-denominated collateral asset on Aave Base or does not permit SOL collateral on Kamino Solana.
- The position has a very small deficit to the target, requiring a non-zero estimate that still reaches at least 1.50.
- The required collateral estimate cannot be represented as a valid non-negative USD amount.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: For each successfully retrieved Aave/Base or Kamino/Solana position with HF below 1.50, the system MUST attempt to estimate the additional collateral needed to reach a projected HF of at least 1.50.
- **FR-002**: For Aave positions on Base, the estimate MUST use a BTC-denominated asset accepted and eligible as collateral for the position; it MUST NOT assume an unsupported BTC representation.
- **FR-003**: For Kamino positions on Solana, the estimate MUST use SOL as the collateral asset.
- **FR-004**: The estimate MUST account for the position's current collateral and debt, the applicable collateral risk parameters, and the USD value of the contribution asset.
- **FR-005**: The system MUST use sufficiently current and reliable price and protocol data for the estimate. If the required data cannot be validated or is insufficient, it MUST report the estimate as unavailable without displaying a numeric contribution.
- **FR-006**: The system MUST display contribution information only when the Health Factor is below 1.50. A position with HF >= 1.50 MUST NOT display a contribution amount or unavailable-estimate message.
- **FR-007**: When an estimate is available, the CLI MUST display it as a USD amount with the relevant asset label (BTC or SOL), directly below the corresponding network's Health Factor.
- **FR-008**: The CLI MUST NOT display the estimated token quantity or unit price as part of this feature.
- **FR-009**: The estimate MUST be informational only. The system MUST NOT execute a purchase, deposit, or other transaction.
- **FR-010**: When a position below 1.50 cannot be estimated safely, the CLI MUST clearly communicate that the estimate is unavailable and MUST NOT present an unsupported numeric amount.
- **FR-011**: Contribution estimation failures for one position MUST NOT prevent the system from reporting Health Factors or contribution estimates for other positions.
- **FR-012**: The system MUST classify a Health Factor of exactly 1.50 as Seguro, consistent with the target boundary.

### Key Entities

- **Contribution Estimate**: An informational USD estimate for additional eligible collateral required by one lending position to reach the target Health Factor. It identifies the corresponding contribution asset and whether a reliable amount is available.
- **Lending Position**: A monitored Aave/Base or Kamino/Solana position whose current collateral, debt, and applicable protocol risk parameters determine its estimated contribution.
- **Market and Protocol Data**: Current asset valuation and position-specific risk information needed to decide whether an estimate is valid and calculate its USD amount.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For every supported below-threshold position with valid, sufficient inputs, the displayed estimate is in USD and the resulting projected Health Factor is at least 1.50.
- **SC-002**: Positions with HF >= 1.50 are classified as Seguro and zero contribution lines are displayed.
- **SC-003**: For positions below 1.50 that lack reliable required data or an eligible contribution asset, zero numeric contribution amounts are displayed and the estimate is explicitly reported as unavailable.
- **SC-004**: In all supported scenarios, Aave/Base estimates identify BTC and Kamino/Solana estimates identify SOL; no token quantity or unit price is shown.
- **SC-005**: Running the feature performs no purchase, deposit, or other position-changing transaction.

## Assumptions

- The safe classification boundary is HF >= 1.50, as required for the contribution target; existing classification logic must be aligned if its boundary is exclusive.
- Contribution estimates are in scope only for Aave on Base and Kamino on Solana; other supported protocol/network combinations retain their existing Health Factor behavior.
- The estimate represents adding the purchased asset as collateral, with no automatic transaction or guarantee that the user will execute it.
- BTC means the BTC-denominated collateral representation actually supported and eligible in the Aave market and position; if none is available, no estimate can be provided.
- Estimates are position-specific and use applicable protocol collateral parameters rather than assuming that a dollar of contribution has the same effect across protocols or assets.
- The terminal remains the presentation surface for this feature, and amounts are formatted as USD using the CLI's existing locale conventions.
