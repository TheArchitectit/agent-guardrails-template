# Proposal: Verify sandbox runtime conformance

## Why

The current sandbox conformance evaluator checks policy and generated plans but does not execute a conformance workload in a supported runtime. That evidence cannot establish runtime containment.

## What Changes

- Require supported-runtime execution and runtime-specific evidence for claims that benign capability checks were exercised.
- Require permitted operations to succeed and denied operations to be blocked through the configured runtime.
- Bind the observed isolation level, enforcing runtime identity/version, and containment-through-completion result into signed, offline-verifiable workload evidence.
- Keep policy-plan checks separate and preserve full escape-fixture conformance as unverified until its separate suite runs.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `sandbox-conformance-and-evidence`: distinguish safe runtime capability evidence from escape-fixture conformance, and bind sandbox identity and completion state into signed evidence.

## Impact

The sandbox conformance harness, supported runtime integrations, and closure evidence. Safe capability checks do not replace escape-fixture testing or establish resistance to escape attempts. No production code or closure status changes are included in this proposal.
