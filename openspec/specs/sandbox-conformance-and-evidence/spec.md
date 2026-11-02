# Sandbox Conformance and Evidence

Status: CLOSED (EXERCISED 2)

## Purpose

Verify sandbox boundary confinement against escape fixtures and record the actual isolation level and measured host capability in execution results and compliance evidence.

## Requirements

### Requirement: escape-fixture conformance

Each isolation level SHALL be tested against escape fixtures covering filesystem traversal, unauthorized network egress, process/namespace escape, and undeclared secret access; any successful escape SHALL fail certification of that level.

#### Scenario: release conformance run

- **GIVEN** the escape fixture suite for a level
- **WHEN** conformance runs against a reference runtime
- **THEN** every escape attempt is blocked and logged as expected, or the level fails certification for that runtime.

### Requirement: isolation in evidence

Every evidence envelope for an executed workload SHALL record the declared isolation level, the enforcing runtime identity and version, and confirmation that containment held through completion.

#### Scenario: consumer verifies containment claim

- **GIVEN** an evidence envelope from a completed workload
- **WHEN** a downstream gate or auditor inspects the envelope
- **THEN** the isolation level and runtime identity are present and covered by the envelope signature.

## Requirement Closure

| Requirement | Status | Implementation | Verification |
|---|---|---|---|
| `escape-fixture conformance` | **EXERCISED** | `mcp-server/internal/guardrails/sandbox_conformance.go` (`RunSandboxEscapeConformance`, `DefaultSandboxEscapeFixtures`, `CheckPathContainment`, `CheckSecretContainment`), `mcp-server/internal/guardrails/sandbox_mounts.go`, `mcp-server/internal/guardrails/sandbox_network.go` | `mcp-server/internal/guardrails/sandbox_conformance_test.go`, `mcp-server/internal/guardrails/sandbox_mounts_test.go`, `scripts/test_check_runner_isolation.py` |
| `isolation in evidence` | **EXERCISED** | `mcp-server/internal/guardrails/sandbox.go` (`SandboxResult.ActualIsolationLvl`, `SandboxViolations`), `mcp-server/internal/guardrails/sandbox_capability.go` (`MeasureSandboxCapability`), `mcp-server/internal/guardrails/compliance_probe.go` | `mcp-server/internal/guardrails/sandbox_capability_test.go`, `mcp-server/internal/mcp/tools_sandbox_test.go` |
