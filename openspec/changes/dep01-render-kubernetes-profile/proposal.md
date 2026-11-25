# Proposal

## Why

`mcp-server/deploy/k8s-deployment.yaml` contains deployment placeholders and `${NAME:-default}` forms without a documented renderer, while the workload exposes Pod ports and probes that must agree with the selected profile. A bank rollout needs a reproducible, reviewed render for an explicitly supported private profile.

## What Changes

- Add a supported profile selection, deterministic manifest rendering and validation, and a documented apply/preflight path. Preserve the profile’s bind and exposure guardrails; do not weaken them to make probes pass.

## Capabilities

### New Capabilities

- `kubernetes-profile-evidence`: render and verify a supported kubernetes profile evidence for an enterprise rollout.

### Modified Capabilities

None.

## Impact

`mcp-server/deploy/k8s-deployment.yaml`, profile configuration/validation, and deployment runbook. Candidate evidence contains no secrets or private ingress endpoints.
