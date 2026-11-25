# Proposal: Bind assembly to runtime evidence

## Why

The control-plane composition contract in Spec 08 requires evaluations to identify the manifest, policy, profile, and component versions actually used. A separate assembly pin and a runtime check do not yet prove that the selected deployed MCP build and policy digest are the same inputs exercised by those checks.

## What Changes

- Define assembly identity fields that link the selected MCP build and policy digest to runtime checks.
- Require exact match evidence between the selected deployment inputs and the runtime observations.
- Verify both a matching fixture and a deliberate mismatch fixture; keep transport to an external consumer out of scope.

## Capabilities

### New Capabilities

- `assembly-runtime-evidence`: binds assembly pins to the exact runtime inputs and results.

### Modified Capabilities

None.

## Impact

Assembly manifest and runtime evidence records, plus authorized fixture-based conformance checks. This preserves the Spec 08 runtime contract and does not change the separate assembly pins themselves.
