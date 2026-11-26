# Spec Delta

## Purpose

The optional Vision dispatch runs ahead of the common MCP authorization and decision audit. Vision REST has its own registration group. An adopter needs a verified decision boundary for every enabled path.

## ADDED Requirements

### Requirement: Enabled optional tools use the registered authorization and audit boundary
Every enabled optional operation SHALL pass the same trusted caller, scope, role and resource authorization and redacted decision-audit contract as other operations before reading scoped data, calling a provider, or causing a mutation. The supported profile SHALL keep optional operations disabled by default.

#### Scenario: Authorized optional operation
- **WHEN** an enabled optional operation is called with the required server-resolved authority
- **THEN** the common decision permits the correct operation and records a redacted outcome

#### Scenario: Denied optional operation
- **WHEN** authority, resource scope or policy is missing or mismatched
- **THEN** the operation is denied before data access, provider invocation or mutation and a bounded denial is recorded

#### Scenario: Disabled optional operation
- **WHEN** the selected profile disables the optional surface
- **THEN** the surface is absent from discovery and its routes cannot execute
