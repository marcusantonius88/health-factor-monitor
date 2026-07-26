# Health Factor Monitor Constitution

## Core Principles

### I. Provider-Based Architecture

The application SHALL separate business logic from protocol-specific implementations.

The core domain MUST remain independent of blockchain networks, SDKs, RPC providers, and third-party APIs.

Every lending protocol (e.g., Aave, Kamino, Morpho, Compound) MUST implement a common Provider interface, allowing new integrations without modifying existing business logic.

The architecture SHALL favor composition over conditional logic, enabling new providers to be added through extension rather than modification.

---

### II. Interface Independence

Business logic MUST NOT depend on any specific user interface.

The CLI is the initial interface for the MVP, but the same application services MUST be reusable by future interfaces such as Telegram bots, REST APIs, scheduled jobs, or web dashboards.

No business rules SHALL be implemented directly inside interface adapters.

---

### III. Simplicity & Incremental Delivery

The project SHALL prioritize simple, maintainable solutions.

Each specification MUST deliver a small, complete, and independently valuable increment.

Infrastructure, dependencies, and architectural complexity MUST only be introduced when justified by explicit project requirements.

The project SHALL follow the YAGNI ("You Aren't Gonna Need It") principle by default.

---

### IV. Quality & Maintainability

All external dependencies MUST be abstracted behind interfaces.

Business logic SHOULD be deterministic, testable, and independent from infrastructure concerns.

All operations interacting with external resources MUST receive a `context.Context`.

Errors MUST be returned explicitly and never silently ignored.

Logging SHOULD be structured and provide enough information for troubleshooting without exposing sensitive information.

Automated tests MUST accompany all business logic.

---

### V. Extensibility

The architecture SHALL support continuous evolution without requiring significant refactoring.

Adding support for a new lending protocol, blockchain network, notification channel, or user interface SHOULD require only new implementations of existing abstractions.

Breaking existing functionality to introduce new capabilities is discouraged unless explicitly justified by a specification.

## Technology Standards

The project SHALL use:

- Go as the primary programming language.
- Git for version control.
- GitHub as the source code hosting platform.
- Go Modules for dependency management.
- Standard Go tooling whenever possible.

Technology choices not defined by this Constitution SHALL be decided during the planning phase of each specification.

## Development Workflow

All development MUST follow the Spec Kit workflow.

Each feature SHALL progress through the following phases:

1. Constitution
2. Specification
3. Clarification (when applicable)
4. Planning
5. Tasks
6. Implementation
7. Analysis (recommended)

Specifications MUST describe expected behavior and business outcomes rather than implementation details.

Implementation decisions belong to the Planning phase.

## Governance

This Constitution is the highest-level engineering authority for the project.

Every specification, implementation plan, and code contribution MUST comply with these principles.

Architectural deviations MUST be explicitly documented and justified within the corresponding specification.

Changes to this Constitution require:
- A documented rationale.
- A version increment.
- Review before adoption.

**Version**: 1.0.0  
**Ratified**: 2026-07-26  
**Last Amended**: 2026-07-26