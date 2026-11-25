# Design

## Context

The [proposal](proposal.md) covers evidence for sandbox execution, semantic
classification, action and ingestion boundaries, and decision envelopes.
Each evidence record needs to identify the surface and configuration it covers.

## Decisions

- Keep focused tests for the existing policy and harness logic. For a
  runtime claim, attach results from benign permitted and denied operations
  observed through the supported runtime. Record platform, runtime version,
  and observed enforcement for each operation.
- Bind semantic results to the approved corpus, evaluation policy, and
  configured classifier versions. Record per-category outcomes against the
  declared budgets. Require matching immutable evidence before assigning a
  release verification status.
- Map each published action and ingestion boundary to the supported consumer
  and integration test that exercises it. Where the test exercises an
  explicit MCP tool, record its scope at the tool boundary.
- Carry the active policy digest and classifier identity, version, category,
  outcome, and signing state through the decision evidence path. Require
  signature verification for reports that claim signed evidence. Retain
  existing reader compatibility for records that predate these fields.

## Rollout

Publish diagnostics and versioned evidence before enabling a blocking gate.
Record unavailable runtime capabilities or configurations with a reason.
Use those results to scope pilot readiness claims to the consumer versions,
policy bundles, and runtime profiles exercised. Repeat the checks for each
additional supported profile and for changes that invalidate prior evidence.
