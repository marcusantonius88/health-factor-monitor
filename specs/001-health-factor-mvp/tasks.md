# Tasks: Health Factor Monitor MVP

**Input**: Design documents from `specs/001-health-factor-mvp/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Organization**: Tasks grouped by user story for independent implementation and testing.

**Format**: `[ID] [P?] [Story?] Description with file path`

## Path Conventions

All paths below are relative to repository root: `/home/marcus/projects/health-factor-monitor`

---

## Phase 1: Setup

**Purpose**: Project initialization, directory structure, and test fixtures

- [x] T001 Initialize Go module at project root with `go mod init github.com/marcus/health-factor-monitor` and create directory structure per plan.md (`cmd/hfmon/`, `internal/domain/`, `internal/application/`, `internal/infrastructure/aave/`, `internal/infrastructure/kamino/`, `internal/infrastructure/config/`, `internal/interfaces/cli/`, `tests/integration/`, `tests/testdata/`)
- [x] T002 [P] Create sample testdata config file at `tests/testdata/valid-config.json` with valid dummy RPC endpoints and two positions (one Aave/Ethereum, one Kamino/Solana) matching the JSON format from `contracts/config-contract.md`

---

## Phase 2: Foundational — Domain Entities & Ports

**Purpose**: Core domain types and interface contracts that all user stories depend on. MUST complete before any user story phase.

- [x] T003 [P] Define Wallet and LendingPosition domain entities in `internal/domain/wallet.go` with fields and validation per `data-model.md`
- [x] T004 [P] Define HealthFactor domain entity (Value float64, classification string) and ProviderResult struct (Position, HealthFactor pointer, Error string) in `internal/domain/healthfactor.go` per `data-model.md`
- [x] T005 [P] Define HealthFactorProvider interface (Protocol(), Network(), GetHealthFactor(ctx, address)) in `internal/domain/provider.go` per `contracts/provider-interface.md`
- [x] T006 [P] Define Config struct (RPCEndpoints map, Positions slice) and ConfigLoader interface (Load()) in `internal/domain/config.go` per `contracts/config-contract.md`
- [x] T007 Write table-driven unit tests for all domain entities, ProviderResult validation (exactly one of HealthFactor or Error), and HealthFactorProvider interface in `internal/domain/domain_test.go`

**Checkpoint**: Foundation ready — entities, ports, and shared types defined and tested.

---

## Phase 3: User Story 2 — Wallet Configuration (Priority: P1)

**Goal**: Users can define their wallets and RPC endpoints in a JSON config file and get clear validation errors for misconfiguration.

**Independent Test**: Provide a valid config with two positions and verify all are loaded; provide a malformed config (missing fields, bad address, unsupported protocol) and verify a descriptive error is returned without any provider calls.

### Implementation for User Story 2

- [x] T008 [P] [US2] Write config reader unit tests in `internal/infrastructure/config/reader_test.go` covering valid config, missing file, malformed JSON, unsupported protocol, missing required field, invalid address format
- [x] T009 [US2] implement JSON config file loader in `internal/infrastructure/config/reader.go` — parse `config.json`, populate Config struct, return error for I/O failures
- [ ] T010 [US2] Implement config validation rules in `internal/infrastructure/config/reader.go` — validate at least one RPC endpoint, position address format (Ethereum 0x-prefixed hex, Solana base58), supported protocol+network combos, non-empty alias
- [ ] T011 [US2] Wire ConfigLoader into CLI app entry point in `cmd/hfmon/main.go` — parse `-config` flag, load and validate config on startup, print error and exit(1) on failure

**Checkpoint**: Configuration management works independently. `go test ./internal/infrastructure/config/...` passes. Running with `-config nonexistent.json` prints a clear error.

---

## Phase 4: User Story 1 — Core Health Factor Check (Priority: P1) 🎯 MVP

**Goal**: Users can check Health Factors for their Aave lending positions via the CLI. Results include health classification (US4) and individual provider failures don't crash the application (US3).

**Independent Test**: Run with a valid config pointing to a real Ethereum RPC and an Aave wallet address with an active position. Verify output shows alias, protocol, network, HF value, and classification. Run with an invalid RPC URL and verify other providers still get results.

### Tests for User Story 1

- [ ] T012 [P] [US1] Write Aave provider unit tests in `internal/infrastructure/aave/provider_test.go` with a mock HTTP server returning realistic `getUserAccountData` responses, covering success, timeout, and malformed response
- [ ] T013 [P] [US3] Write CheckService unit tests in `internal/application/service_test.go` covering: all providers succeed, one provider fails (verify partial results), all providers fail (verify all errors returned), unknown protocol (verify error returned)

### Implementation for User Story 1 (includes US3 resilience + US4 classification)

- [ ] T014 [P] [US1] Implement Aave provider adapter in `internal/infrastructure/aave/provider.go` — `Protocol()` returns "aave", `Network()` returns "ethereum", `GetHealthFactor()` calls `eth_call` to `getUserAccountData` on the Aave V3 Pool contract, decodes the 6-uint256 response, extracts healthFactor at index 5, converts from 1e18 scaling to float64
- [ ] T015 [P] [US4] Implement HealthFactor classification logic in `internal/domain/healthfactor.go` — `Classify(value float64, thresholds ...)` returns "safe" (>1.5), "warning" (1.0–1.5), "critical" (≤1.0) with configurable thresholds; write tests alongside
- [ ] T016 [US1] Implement CheckService in `internal/application/service.go` — `NewCheckService(config, providers map)` matching providers to positions by protocol+network, `CheckAll(ctx)` iterates all positions, calls matched provider's `GetHealthFactor`, wraps result in `ProviderResult`, never panics on unmatched protocol, never stops on individual errors
- [ ] T017 [US3] Implement error handling in `internal/application/service.go` — ensure CheckService processes all positions even when individual providers return errors or timeouts; each position gets exactly one ProviderResult (success or error)
- [ ] T018 [US1] Implement CLI output formatting in `internal/interfaces/cli/app.go` — `FormatResults(results)` renders aligned table with columns: Position, Protocol, Network, Health Factor, Status; includes classification labels; handles both success and error results
- [ ] T019 [US3] Implement CLI exit code logic in `internal/interfaces/cli/app.go` — exit(0) if at least one HF retrieved, exit(1) if all positions failed; wire into main flow
- [ ] T020 [US1] Wire complete application in `cmd/hfmon/main.go` — init config loader, init providers (Aave only for MVP), init CheckService, call CheckAll, format and print results, set exit code

**Checkpoint**: MVP functional. `go build ./cmd/hfmon` succeeds. Running with valid Aave config prints table with HF and classification. Running with bad RPC shows error for that provider and still exits 0 if others succeed.

---

## Phase 5: User Story 1 Extension — Kamino Provider (Priority: P1)

**Goal**: Extend the MVP to also support Kamino on Solana, completing the two-protocol requirement.

**Independent Test**: Add a Kamino wallet to config and verify the output includes two rows (Aave + Kamino) with correct HFs. Provider is isolated — no changes to domain, application, or CLI layers needed.

### Implementation for Kamino Provider

- [ ] T021 [P] [US1] Write Kamino provider unit tests in `internal/infrastructure/kamino/provider_test.go` with mock responses covering success, API error, and timeout
- [ ] T022 [P] [US1] Implement Kamino provider adapter in `internal/infrastructure/kamino/provider.go` — `Protocol()` returns "kamino", `Network()` returns "solana", `GetHealthFactor()` calls Kamino's official API/REST interface to retrieve user obligation data, derives Health Factor from returned data
- [ ] T023 [US1] Register Kamino provider in `cmd/hfmon/main.go` — add Kamino provider to the provider map alongside Aave; no other code changes needed
- [ ] T024 [US1] Validate both providers end-to-end: run with config containing both Aave and Kamino positions, verify output displays both with correct HFs

**Checkpoint**: Both providers work. `go test ./internal/infrastructure/kamino/...` passes. Output shows both protocols side by side.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Integration tests, quality validation, and quickstart verification.

- [ ] T025 [P] Add Aave provider integration test in `tests/integration/aave_test.go` — test helper that connects to a real Ethereum RPC (configured via env var), calls `getUserAccountData` for a known test address, validates response structure; mark as `//go:build integration`
- [ ] T026 [P] Add Kamino provider integration test in `tests/integration/kamino_test.go` — test helper that connects to Kamino API, validates response structure; mark as `//go:build integration`
- [ ] T027 [P] Add unit tests for CLI output formatting edge cases in `internal/interfaces/cli/app_test.go` — zero results, all errors, mixed results, classification color mapping
- [ ] T028 Run `go mod tidy`, `go vet ./...`, and verify all unit tests pass
- [ ] T029 Verify all quickstart validation scenarios from `quickstart.md` produce expected output and exit codes

---

## Dependencies & Execution Order

### Phase Dependencies

```
Phase 1: Setup        — No dependencies
       ↓
Phase 2: Foundational — Depends on Setup — BLOCKS all user stories
       ↓
Phase 3: US2 (Config) — Depends on Foundational
       ↓
Phase 4: US1 (MVP)    — Depends on Foundational + US2's Config type for input
       ↓
Phase 5: Kamino Prov  — Depends on Foundational + US1 (matches existing service interface)
       ↓
Phase 6: Polish       — Depends on US1 completion
```

### User Story Dependencies

- **US2 (Config, P1)**: No dependencies on other stories. Can be built and tested in isolation. First independently valuable slice.
- **US1 (Core Check, P1)**: Depends on US2 for configuration input. US3 and US4 are implemented within this phase (not separate phases) because they share the same service, CLI, and domain code.
- **Kamino Extension (P1)**: Depends on US1's service interface and provider pattern. Completely isolated — no changes to any existing files beyond `main.go` registration.

### Within Each User Story

- Tests written alongside implementation (see phases above)
- Infrastructure tasks are isolated from domain tasks per user preference
- Each phase is independently testable

### Parallel Opportunities

| Phase | Parallel Tasks | Rationale |
|-------|---------------|-----------|
| Phase 1 | T002 | Independent files |
| Phase 2 | T003, T004, T005, T006 | Independent entities/interfaces |
| Phase 3 | T008 | Test-only, no dependencies |
| Phase 4 | T012, T013, T014, T015 | Aave provider, service tests, classification are independent files |
| Phase 5 | T021, T022 | Kamino test + implementation are independent |
| Phase 6 | T025, T026, T027 | Integration and CLI tests are independent files |

---

## Parallel Example: User Story 1 (Phase 4)

```bash
# Launch all test tasks together:
Task: "Write Aave provider unit tests in internal/infrastructure/aave/provider_test.go"
Task: "Write CheckService unit tests in internal/application/service_test.go"

# Launch all independent implementation tasks together:
Task: "Implement Aave provider adapter in internal/infrastructure/aave/provider.go"
Task: "Implement HealthFactor classification logic in internal/domain/healthfactor.go"

# Sequential (service depends on provider + classification):
Task: "Implement CheckService in internal/application/service.go"
```

## Implementation Strategy

### MVP First (Phases 1-4 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: US2 — Config loading
4. Complete Phase 4: US1 — Aave provider + CheckService + CLI + classification
5. **STOP and VALIDATE**: Run `go build ./cmd/hfmon` and test with real Aave config
6. MVP delivered: single-protocol Health Factor checking

### Incremental Delivery

1. Phases 1-2 → Foundation ready
2. Phase 3 → Config loading (independently testable)
3. Phase 4 → Aave + CLI MVP (deployable value!)
4. Phase 5 → Add Kamino (extends without breaking)
5. Phase 6 → Quality polish

### Parallel Team Strategy

With multiple developers:
1. Team completes Phase 1 + Phase 2 together
2. Developer A: Phase 3 (Config)
3. Developer B: Phase 4 (Aave + Service + CLI) — can start after Phase 2
4. Both merge → Phase 5 (any remaining dev)

---

## Total Task Count: 29

| Phase | Count | Description |
|-------|-------|-------------|
| Phase 1: Setup | 2 | Project init |
| Phase 2: Foundational | 5 | Domain + interfaces |
| Phase 3: US2 Config | 4 | Configuration |
| Phase 4: US1 Core MVP | 9 | Aave + Service + CLI + classification + error handling |
| Phase 5: Kamino | 4 | Kamino provider |
| Phase 6: Polish | 5 | Integration tests, vet, quickstart |
| **Total** | **29** | |
