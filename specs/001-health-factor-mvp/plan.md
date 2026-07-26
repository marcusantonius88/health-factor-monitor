# Implementation Plan: Health Factor Monitor MVP

**Branch**: `001-health-factor-mvp` | **Date**: 2026-07-26 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/001-health-factor-mvp/spec.md`

## Summary

Build a CLI application that reads wallet configuration from a JSON file, retrieves the current Health Factor for each configured lending position on Aave (Ethereum) and Kamino (Solana) via raw JSON-RPC calls over HTTP, and displays results with health classifications. The application follows Hexagonal Architecture (Ports & Adapters), keeps business logic free of blockchain SDK dependencies, and tolerates individual provider failures.

## Technical Context

**Language/Version**: Go 1.22+

**Primary Dependencies**:
- Standard library only (`flag`, `net/http`, `encoding/json`, `context`, `os`, `fmt`, `slog`)
- No third-party blockchain SDKs — raw JSON-RPC over HTTP for both Ethereum and Solana
- No third-party CLI frameworks — standard `flag` package
- No third-party config parsers — `encoding/json` for JSON config files

**Storage**: N/A — no persistent storage. Runtime state only.

**Testing**: `go test` with standard `testing` package. Table-driven tests for domain logic. Integration test helpers for provider contract verification.

**Target Platform**: Linux, macOS (cross-compiled via `GOOS`/`GOARCH`)

**Project Type**: CLI (`cmd/hfmon` entry point)

**Performance Goals**: Complete all configured checks within 60 seconds (dominated by RPC round-trips, not computation)

**Constraints**: Network access to Ethereum and Solana RPC endpoints required; no external database, no background jobs, no notifications

**Scale/Scope**: Single user, ~1–50 wallets, up to 2 providers (Aave, Kamino)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Gate I — Provider-Based Architecture
- **Check**: Business logic separated from protocol-specific code → ✅ (Provider interface in domain, impls in infrastructure)
- **Check**: Core domain independent of blockchain SDKs → ✅ (raw RPC via net/http, no SDKs)
- **Check**: Common Provider interface for all protocols → ✅ (HealthFactorProvider interface defined in contracts/)
- **Check**: Composition over conditional logic → ✅ (service composes providers; dispatches by protocol+network match)

### Gate II — Interface Independence
- **Check**: Business logic independent of CLI → ✅ (domain + application layers have zero CLI imports)
- **Check**: Same services reusable by other interfaces → ✅ (CheckService is interface-agnostic; driving adapters swappable)
- **Check**: No business rules in interface adapters → ✅ (CLI adapter only formats output; rules in domain)

### Gate III — Simplicity & Incremental Delivery
- **Check**: Minimal, independently valuable increment → ✅ (MVP delivers core HF checking only, no extras)
- **Check**: YAGNI applied → ✅ (no DB, no notifications, no background jobs, no auth, no web interface)
- **Check**: Complexity justified by requirements → ✅ (provider abstraction justified by two-protocol requirement)

### Gate IV — Quality & Maintainability
- **Check**: External dependencies abstracted behind interfaces → ✅ (HealthFactorProvider, ConfigLoader)
- **Check**: Business logic deterministic and testable → ✅ (domain entities are pure data; service is mockable)
- **Check**: `context.Context` on all external operations → ✅ (GetHealthFactor, Load signatures include ctx)
- **Check**: Errors returned explicitly → ✅ (error return values, never panics; ProviderResult discriminates success/error)
- **Check**: Automated tests for all business logic → ✅ (planned for tasks phase)

### Gate V — Extensibility
- **Check**: Adding new provider requires only new impl of existing abstraction → ✅ (implement HealthFactorProvider interface + register)
- **Check**: Breaking changes avoided unless justified → ✅ (MVP is v1, no prior API to break)

### Technology Standards
- **Check**: Go as primary language → ✅
- **Check**: Go Modules for dependency management → ✅
- **Check**: Standard Go tooling → ✅

**Result**: ALL GATES PASS — no violations to justify.

## Project Structure

### Documentation (this feature)

```text
specs/001-health-factor-mvp/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   ├── provider-interface.md
│   ├── application-service.md
│   └── config-contract.md
└── tasks.md             # (created by /speckit.tasks)
```

### Source Code (repository root)

```text
cmd/
└── hfmon/
    └── main.go                  # CLI entry point (flag parsing, wiring)

internal/
├── domain/
│   ├── wallet.go                # Wallet address, LendingPosition, HealthFactor entities
│   └── provider.go              # HealthFactorProvider interface (port)
│
├── application/
│   └── service.go               # CheckService — primary use case orchestrator
│
├── infrastructure/
│   ├── aave/
│   │   └── provider.go          # Aave protocol adapter (eth_call via JSON-RPC)
│   ├── kamino/
│   │   └── provider.go          # Kamino protocol adapter (getAccount via JSON-RPC)
│   └── config/
│       └── reader.go            # JSON config loader and validator
│
└── interfaces/
    └── cli/
        └── app.go               # CLI adapter (flag parsing, output formatting, exit code)

tests/
├── integration/
│   └── rpc_test.go              # Provider contract verification (optional, requires live RPC)
└── testdata/
    └── valid-config.json        # Sample config for unit tests
```

**Structure Decision**: Hexagonal Architecture (Ports & Adapters). Domain owns ports (HealthFactorProvider, ConfigLoader interfaces). Infrastructure implements driven adapters (Aave, Kamino, config). Interfaces implements driving adapters (CLI). Application orchestrates the flow.

## Complexity Tracking

> No Constitution violations found — section not applicable.
