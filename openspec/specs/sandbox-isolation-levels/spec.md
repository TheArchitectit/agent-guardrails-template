# Sandbox Isolation Levels

Status: CLOSED (EXERCISED 3)

## Purpose

Define and enforce explicit sandbox isolation levels with capability matrices covering filesystem mounts, network egress, process limits, and runtime boundaries, refusing execution when the requested or policy-declared level cannot be established.

## Requirements

### Requirement: named levels with capability matrices

The specification SHALL define four isolation levels (observer, restricted, contained, ephemeral), each with an explicit capability matrix covering filesystem, network, process, secrets, and state persistence.

#### Scenario: bundle references a level

- **GIVEN** a policy bundle assigning an isolation level to a workload class
- **WHEN** the bundle is validated
- **THEN** the level name resolves to its canonical capability matrix and unknown level names are rejected.

### Requirement: declared level enforcement

The runtime SHALL enforce the workload's declared isolation level or refuse execution. Silent downgrade to a weaker level SHALL NOT occur.

#### Scenario: host cannot provide required level

- **GIVEN** a workload requiring `contained` or `ephemeral` isolation and a host lacking the required container or namespace capability
- **WHEN** the workload is scheduled
- **THEN** execution is refused and an ERROR evidence envelope is emitted naming the missing capability.

### Requirement: fail-closed containment

Any containment fault or boundary violation during execution SHALL terminate the workload, emit an ERROR evidence envelope, and prevent a PASS verdict.

#### Scenario: containment fault mid-run

- **GIVEN** a running workload whose container or namespace isolation faults
- **WHEN** the runtime detects the fault
- **THEN** the workload is terminated and the resulting verdict is ERROR with fault evidence attached.

## Requirement Closure

| Requirement | Status | Implementation | Verification |
|---|---|---|---|
| `named levels with capability matrices` | **EXERCISED** | `mcp-server/internal/guardrails/sandbox_capability_matrix.go` (`SandboxCapabilityMatrix`, `CapabilityMatrixForTier`, `ResolveCapabilityMatrix`, `ScrubEnvironment`, `containerEnvArgs`), `mcp-server/internal/guardrails/sandbox_conformance.go` (`TierToSandboxLevel`), `mcp-server/internal/guardrails/sandbox_config.go` (`ResourceLimits.Secrets`, `ResourceLimits.StatePersistence`), `mcp-server/internal/guardrails/sandbox_config_file.go` (`LoadSandboxConfigFromYAML`), `mcp-server/internal/guardrails/sandbox_policy.go` (`mergeLimits`) | `mcp-server/internal/guardrails/sandbox_capability_matrix_test.go`, `mcp-server/internal/guardrails/sandbox_config_file_test.go`, `mcp-server/internal/guardrails/sandbox_policy_test.go` |
| `declared level enforcement` | **EXERCISED** | `mcp-server/internal/guardrails/sandbox.go` (`ExecuteFor`, `ExecuteForLimits`, `isSetupError`), `mcp-server/internal/mcp/tools_sandbox.go` (`handleSandboxExecute`, `handleSandboxConfig`) | `mcp-server/internal/mcp/tools_sandbox_test.go`, `mcp-server/internal/guardrails/sandbox_test.go`, `mcp-server/internal/guardrails/sandbox_capability_test.go` |
| `fail-closed containment` | **EXERCISED** | `mcp-server/internal/guardrails/sandbox.go` (`ErrSandboxViolation`, `ErrResourceExceeded`, `execL0`/`execL1` env scrub, `execL2` `--env` forwarding), `mcp-server/internal/guardrails/sandbox_mounts.go`, `mcp-server/internal/guardrails/sandbox_network.go`, `mcp-server/internal/guardrails/sandbox_capability_matrix.go` (`ScrubEnvironment`) | `mcp-server/internal/guardrails/sandbox_limits_test.go`, `mcp-server/internal/guardrails/sandbox_mounts_test.go`, `mcp-server/internal/guardrails/sandbox_network_test.go`, `mcp-server/internal/guardrails/sandbox_net_enabled_test.go`, `mcp-server/internal/guardrails/sandbox_capability_matrix_test.go` |
