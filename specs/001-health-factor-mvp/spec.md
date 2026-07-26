# Feature Specification: Health Factor Monitor MVP

**Feature Branch**: `001-health-factor-mvp`

**Created**: 2026-07-26

**Status**: Draft

**Input**: User description: "Build the MVP of a Health Factor Monitor for DeFi lending positions."

---

# User Scenarios & Testing *(mandatory)*

## User Story 1 - Check Health Factors for Configured Lending Positions (Priority: P1)

A user configures one or more cryptocurrency wallets associated with supported DeFi lending protocols and executes the CLI application. The application retrieves the current Health Factor for every configured lending position and displays the results in a clear, human-readable format.

**Why this priority**

This is the core purpose of the application. Without this capability, the product provides no value.

**Independent Test**

Configure wallets with active lending positions on supported protocols and verify that the displayed Health Factor matches the value reported by the corresponding protocol.

### Acceptance Scenarios

1. **Given** a configured wallet with an active lending position, **When** the user runs the application, **Then** the current Health Factor is displayed.

2. **Given** multiple configured wallets across supported protocols, **When** the application executes, **Then** the Health Factor for every configured lending position is displayed.

---

## User Story 2 - Manage Wallet Configuration (Priority: P1)

A user configures all wallets to be monitored in a single configuration source before executing the application.

**Why this priority**

Users should configure their monitoring targets once and retrieve all Health Factors in a single execution.

**Independent Test**

Provide a valid configuration containing multiple wallets and verify that all configured lending positions are processed.

### Acceptance Scenarios

1. **Given** a valid configuration file, **When** the application starts, **Then** every configured lending position is processed.

2. **Given** an invalid or malformed configuration, **When** the application starts, **Then** a clear validation error is displayed and execution stops.

---

## User Story 3 - Continue When a Provider Fails (Priority: P2)

A user has lending positions across multiple providers. One provider becomes temporarily unavailable during execution. The application retrieves data from all remaining providers and reports the failure without terminating prematurely.

**Why this priority**

Partial results are more valuable than complete failure.

**Independent Test**

Simulate a provider outage and verify that the remaining providers continue to produce results.

### Acceptance Scenarios

1. **Given** one provider is unavailable, **When** the application executes, **Then** Health Factors from available providers are displayed and failures are reported individually.

2. **Given** every provider fails, **When** the application executes, **Then** the application exits with a non-zero status and reports every failure.

---

## User Story 4 - Quickly Understand Position Risk (Priority: P3)

A user wants to understand the health of each lending position without manually interpreting raw Health Factor values.

**Why this priority**

The application should communicate risk clearly rather than simply displaying numbers.

**Independent Test**

Verify that every displayed Health Factor also contains a corresponding health classification.

### Acceptance Scenarios

1. **Given** a retrieved Health Factor, **When** the result is displayed, **Then** an associated health classification is shown.

2. **Given** multiple lending positions, **When** results are displayed, **Then** every position includes both the numeric Health Factor and its health classification.

---

# Edge Cases

- Configuration file does not exist.
- Configuration file cannot be parsed.
- Wallet address format is invalid.
- Wallet has no active lending position.
- Wallet exists but the provider returns no Health Factor.
- Provider request times out.
- Provider returns malformed or unexpected data.
- Multiple providers fail simultaneously.
- Network connectivity is unavailable.

---

# Requirements *(mandatory)*

## Functional Requirements

### Configuration

- **FR-001**: The system MUST load wallet configuration from a predefined configuration source.

- **FR-002**: The system MUST validate the configuration before attempting to retrieve any Health Factor.

---

### Health Factor Retrieval

- **FR-003**: The system MUST retrieve the current Health Factor for every configured lending position using the configured provider.

- **FR-004**: The MVP MUST provide provider integrations for:
  - Aave (Ethereum)
  - Kamino (Solana)

- **FR-005**: The system MUST process every configured lending position independently.

---

### Output

- **FR-006**: The system MUST display the wallet identifier, lending protocol, blockchain network, and current Health Factor.

- **FR-007**: The system MUST associate every retrieved Health Factor with a health classification.

- **FR-008**: The criteria used to classify Health Factor values MUST be configurable.

---

### Error Handling

- **FR-009**: The failure of one provider MUST NOT prevent remaining providers from being processed.

- **FR-010**: The system MUST display meaningful error messages whenever a lending position cannot be processed.

- **FR-011**: The system MUST terminate with a non-zero exit code only when no Health Factors could be successfully retrieved.

---

# Key Entities

## Wallet

Represents a cryptocurrency wallet monitored by the application.

Attributes:

- Address
- Alias (optional)

---

## Lending Position

Represents a wallet's participation in a lending protocol.

Attributes:

- Wallet
- Provider
- Blockchain Network

---

## Provider

Represents a lending protocol capable of providing Health Factor information.

Examples:

- Aave
- Kamino

---

## Health Factor

Represents the liquidation safety metric returned by a lending protocol.

Attributes:

- Numeric Value
- Health Classification

---

# Success Criteria *(mandatory)*

## Measurable Outcomes

- **SC-001**: Every configured lending position supported by the MVP returns either a Health Factor or a meaningful error.

- **SC-002**: Retrieved Health Factors match those reported by the corresponding protocol.

- **SC-003**: Provider failures do not interrupt retrieval from remaining providers.

- **SC-004**: Users can immediately identify the relative health of every lending position from the application output.

---

# Assumptions

- Users provide valid wallet addresses.
- Users have network connectivity.
- Supported providers expose Health Factor information through publicly accessible mechanisms.
- Health Factor information is retrieved in real time.
- The application is intended for a single user.
- Persistent storage is outside the scope of the MVP.
- Notification systems are outside the scope of the MVP.
- Background monitoring is outside the scope of the MVP.