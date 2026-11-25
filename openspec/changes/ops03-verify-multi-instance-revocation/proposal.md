# Proposal

## Why

`mcp-server/internal/auth/registry_lifecycle_test.go` demonstrates revocation propagation for one process-local registry source. That evidence does not establish behavior across a multi-replica deployment, the selected registry distribution mechanism, or failover.

## What Changes

- For a selected multi-instance topology, verify revocation visibility on every serving instance and during the supported failover path within the documented bound. Do not generalize a single-process result to a distributed deployment.

## Capabilities

### New Capabilities

- `multi-instance-revocation-evidence`: verify multi-instance revocation evidence for an enterprise rollout.

### Modified Capabilities

None.

## Impact

Revocation integration test or staging drill and a redacted result record. Use only synthetic test credentials and isolated infrastructure.
