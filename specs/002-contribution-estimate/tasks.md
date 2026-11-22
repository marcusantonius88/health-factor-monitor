# Tasks: Required Contribution Estimate

**Input**: Design documents from `specs/002-contribution-estimate/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/contribution-estimate.md`, `quickstart.md`

**Tests**: Tests are included as requested. Write the focused tests before or alongside implementation and confirm the new cases fail before the corresponding implementation.

**Organization**: Tasks are grouped by user story after shared foundational types and contracts.

## Phase 1: Setup

**Purpose**: Prepare the existing Go project for this feature.

No setup changes are required: the Go module, standard test tooling, provider packages, and CLI already exist. No external dependency or configuration format change is planned.

---

## Phase 2: Foundational — Domain Types and Optional Provider Capability

**Purpose**: Establish estimate/result types and the optional capability without changing the existing `HealthFactorProvider` contract. These tasks block both user stories.

- [x] T001 [P] Add tests for contribution basis/estimate validation in `internal/domain/contribution_test.go`, covering missing/invalid numeric inputs, available estimate with positive USD amount, and unavailable estimate with a reason but no amount.
- [x] T002 Define `ContributionBasis`, `ContributionEstimate`, and position snapshot result types in `internal/domain/contribution.go`; extend `domain.ProviderResult` in `internal/domain/healthfactor.go` to carry an optional estimate while preserving the existing Health Factor/error invariants.
- [x] T003 Add the optional contribution snapshot provider interface in `internal/domain/provider.go`; require `context.Context`, keep `HealthFactorProvider` source-compatible, and distinguish Health Factor retrieval errors from estimate-only unavailability;

**Checkpoint**: Shared domain types compile and validate; existing providers and callers can continue using `HealthFactorProvider` unchanged.

---

## Phase 3: User Story 1 — Estimate Contributions for Below-Target Positions (Priority: P1)

**Goal**: Show a conservative USD estimate for BTC collateral on Aave/Base and SOL collateral on Kamino/Solana when valid, current, position-specific inputs are available. Classify HF exactly 1.50 as Seguro and do not make estimate-only reads for safe positions.

**Independent Test**: With deterministic provider fixtures for both target networks and a Health Factor below 1.50, verify that the displayed USD amount, after asset-unit rounding, yields a projected HF >= 1.50. Verify HF = 1.50 is Seguro and has no estimate line.

### Tests for User Story 1

- [ ] T004 [P] [US1] Add classification tests for HF just below, exactly at, and just above 1.50 in `internal/domain/domain_test.go`; the exact boundary must expect `ClassificationSafe`.
- [ ] T005 [P] [US1] Add shared estimator tests in `internal/domain/contribution_test.go` for the normalized calculation, values already at target, tiny deficits, upward asset-unit/USD-cent rounding, projected HF validation, and invalid risk factors.
- [ ] T006 [P] [US1] Add Aave/Base snapshot tests in `internal/infrastructure/aave/provider_test.go` for pinned-block reads, account data, eligible BTC reserves, effective liquidation threshold/eMode, oracle price, and conservative selection when multiple BTC reserves are eligible.
- [ ] T007 [P] [US1] Add Kamino/Solana snapshot tests in `internal/infrastructure/kamino/provider_test.go` for the selected active obligation, current-slot `minSlot`, matching `markets=all` market+SOL mint, and estimate basis only when the active elevation group is verified.
- [ ] T008 [P] [US1] Add application tests in `internal/application/service_test.go` for attaching an estimate to a successful below-target Health Factor and skipping estimate-only data/calculation at HF >= 1.50.
- [ ] T009 [P] [US1] Add CLI tests in `internal/interfaces/cli/app_test.go` for the BTC/SOL USD line directly below the network row, and no contribution line for a safe position or an out-of-scope protocol/network.

### Implementation for User Story 1

- [ ] T010 [US1] Change `domain.Classify` in `internal/domain/healthfactor.go` so HF exactly 1.50 is `ClassificationSafe`; retain custom-threshold behavior and pass T004.
- [ ] T011 [US1] Implement the shared contribution calculation in `internal/domain/contribution.go` using the normalized effective debt and incremental liquidation factor; round upward at asset precision and USD cents, and reject results whose projected HF remains below 1.50.
- [ ] T012 [US1] Extend the Aave adapter in `internal/infrastructure/aave/provider.go` to resolve the Base Pool/oracle/data provider and read account, reserve, user collateral, and eMode data at one pinned block; preserve a valid Health Factor if estimate-only reads fail.
- [ ] T013 [US1] Implement Aave BTC reserve selection and normalized contribution inputs in `internal/infrastructure/aave/provider.go`; validate live reserve eligibility, supply state/caps, effective liquidation threshold, USD base unit, price/source freshness, and use the largest valid rounded estimate if multiple eligible BTC representations exist.
- [ ] T014 [US1] Pass the configured Solana RPC endpoint into the Kamino provider from `cmd/hfmon/main.go` and update `kamino.NewProvider` in `internal/infrastructure/kamino/provider.go` to retain both RPC and API configuration.
- [ ] T015 [US1] Implement Kamino portfolio refresh in `internal/infrastructure/kamino/provider.go`: read the current Solana slot, request `sections=lending` with `minSlot`, verify `appliedSlot`, and retrieve `/oracles/prices?markets=all` with timestamp/max-age validation for the exact market and SOL mint.
- [ ] T016 [US1] Normalize Kamino obligation debt and SOL liquidation inputs in `internal/infrastructure/kamino/provider.go`, including borrow factors and verified active elevation-group terms; never substitute the hypothetical pair endpoint's auto-selected group for the obligation's active group.
- [ ] T017 [US1] Orchestrate the optional contribution snapshot in `internal/application/service.go`; use the same position/obligation as the Health Factor, calculate only below 1.50, use any refreshed snapshot Health Factor for display, and preserve independent processing of positions.
- [ ] T018 [US1] Render available contribution estimates in `internal/interfaces/cli/app.go` as `Aporte necessário (BTC|SOL): US$ ...` directly below the matching network row, without token quantity or unit price.

**Checkpoint**: US1 is independently demonstrable using fixtures for valid Aave/Base and Kamino/Solana inputs, with an inclusive safe threshold and no extra estimate reads for safe positions.

---

## Phase 4: User Story 2 — Communicate Safely When an Estimate Is Unavailable (Priority: P2)

**Goal**: Clearly report estimate unavailability without a numeric amount, while retaining successful Health Factors and continuing to process other positions.

**Independent Test**: For below-target positions, inject missing/stale prices, incomplete data, ineligible/ambiguous assets, unconfirmed risk groups, and estimate-only provider errors. Verify an unavailable message with no USD amount, valid Health Factors remain visible, and other positions still complete.

### Tests for User Story 2

- [ ] T019 [P] [US2] Add application tests in `internal/application/service_test.go` proving estimate-only failure does not populate the provider's Health Factor error, suppress other positions, or change success/exit-status behavior.
- [ ] T020 [P] [US2] Add Aave failure tests in `internal/infrastructure/aave/provider_test.go` for missing/zero price, unsupported or unverifiably stale oracle source, invalid USD base unit, absent/ineligible BTC reserve, and unavailable reserve/eMode data.
- [ ] T021 [P] [US2] Add Kamino failure tests in `internal/infrastructure/kamino/provider_test.go` for API/RPC failure, `appliedSlot < minSlot`, stale/missing SOL quote, wrong market/mint, incomplete obligation data, and unknown active elevation group.
- [ ] T022 [P] [US2] Add CLI tests in `internal/interfaces/cli/app_test.go` proving unavailable estimates show a reason but no numeric USD amount, token amount, or unit price; safe positions still show no contribution line.

### Implementation for User Story 2

- [ ] T023 [US2] Map Aave estimate-only price, reserve, eligibility, or freshness failures to an unavailable contribution in `internal/infrastructure/aave/provider.go` without discarding a successfully retrieved Health Factor or inventing a BTC asset/price.
- [ ] T024 [US2] Map Kamino estimate-only API, freshness, reserve, or active-group failures to an unavailable contribution in `internal/infrastructure/kamino/provider.go`; preserve a valid Health Factor and never infer missing risk parameters.
- [ ] T025 [US2] Update `internal/application/service.go` to carry estimate unavailability separately from provider failure and continue processing all remaining positions.
- [ ] T026 [US2] Render the unavailable message in `internal/interfaces/cli/app.go` using the relevant BTC/SOL label and a concise reason; ensure no unsupported numeric amount is formatted.

**Checkpoint**: For every unsafe-to-calculate case, the CLI communicates unavailability, never a success-shaped fallback, while retaining other successful results.

---

## Phase 5: Polish and Cross-Cutting Validation

**Purpose**: Document the user-visible behavior and validate the completed feature.

- [ ] T027 [P] Update `README.md` with the below-1.50 USD estimate behavior, supported Aave/Base BTC and Kamino/Solana SOL assets, unavailable state, and the fact that no transactions are executed.
- [ ] T028 Run `go test ./...` and `go vet ./...`; execute the scenarios in `specs/002-contribution-estimate/quickstart.md` with deterministic fixtures and confirm no live transaction or fixed-price assumption is used.

---

## Dependencies & Execution Order

### Phase Dependencies

```text
Phase 1: Setup — no changes required
    ↓
Phase 2: Foundational types/port — blocks both user stories
    ↓
Phase 3: US1 (P1) — shared estimate + protocol success paths + CLI
    ↓
Phase 4: US2 (P2) — fail-closed behavior and unavailable output
    ↓
Phase 5: Polish — README and full validation
```

### Task Dependencies

- T001 → T002 → T003.
- T004 → T010; T005 → T011.
- T006 → T012 → T013; T007 → T014 → T015 → T016.
- T008, T013, and T016 → T017; T009 and T017 → T018.
- T019–T022 are written after the US1 result shape exists and before the matching US2 failure-path changes.
- T020 → T023; T021 → T024; T019, T023, and T024 → T025; T022 and T025 → T026.
- T027 follows the final output behavior; T028 depends on all implementation and documentation tasks.

### User Story Dependencies

- **US1 (P1)**: Depends on the foundational domain types and optional provider capability. Aave and Kamino adapter work can proceed in parallel after the shared calculator contract is established.
- **US2 (P2)**: Depends on US1's result and output model because it adds failure semantics for the same estimate. Its provider tests can be developed in parallel; the two protocol failure implementations remain isolated.

### Parallel Opportunities

| Tasks | Reason |
|-------|--------|
| T004–T009 | Tests use separate domain, Aave, Kamino, application, and CLI test files. |
| T010–T011 | Classification and shared calculation are separate domain files. |
| T012–T013 after T011; T014–T016 in sequence | Aave work is independent of Kamino; each provider's freshness/discovery work precedes its normalization. |
| T019–T022 | US2 failure tests target separate package test files. |
| T023–T024 | Aave and Kamino fail-closed behavior uses separate provider packages. |
| T027 | README update is independent of code validation once output behavior is settled. |

## Parallel Example: User Story 1

```text
# Write independent failing tests first:
T004 Health Factor boundary tests
T005 shared calculation tests
T006 Aave/Base provider snapshot tests
T007 Kamino/Solana provider snapshot tests
T008 application orchestration tests
T009 CLI formatting tests

# After foundational types and tests:
T010 classification boundary implementation  || T011 shared estimator implementation
T012–T013 Aave implementation                || T014–T016 Kamino implementation
```

## Implementation Strategy

1. Complete Phase 2 so both provider adapters share stable result and snapshot types.
2. Deliver US1 first: classification boundary, shared calculation, and Aave/Kamino valid-input paths, with CLI output.
3. Complete US2 before treating the feature as production-ready; failure to validate a source or risk parameter must remain unavailable, never a numeric fallback.
4. Finish with documentation, `go test ./...`, `go vet ./...`, and quickstart fixture validation.

## Format Validation

- All task entries use a checkbox, sequential `T###` ID, optional `[P]`, required story labels for story-phase tasks, and concrete file paths.
- Tasks are ordered by prerequisites and include test-first work for both user stories.
- No code is implemented by this task-generation artifact.
