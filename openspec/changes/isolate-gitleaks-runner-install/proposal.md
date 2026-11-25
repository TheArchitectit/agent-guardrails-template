# Proposal

## Why

The current Secret Validation run at `943bcf6` failed before scanning when the Gitleaks action attempted to install v8.24.3 at `/tmp/gitleaks.tmp`, which already existed. No leak was reported; the scan did not run. Shared runner scratch can collide across concurrent or stale jobs, leaving secret scanning without an executed result.

## What Changes

- Install and verify the pinned scanner in a unique, job-owned scratch directory.
- Preserve fail-closed behavior so installation or integrity failures fail the scan job.
- Add concurrency, stale-path, clean-scan, and safe synthetic-detection acceptance coverage without logging fixture secret values.

## Capabilities

### New Capabilities
- `gitleaks-runner-installation`: isolated and integrity-verified Gitleaks installation for CI scanning.

### Modified Capabilities
- None.

## Impact

This addresses TPL-38 in `.github/workflows/secret-validation.yml` and the pinned Gitleaks runner setup. It does not change scan scope or report a secret leak. No implementation or scan result is claimed by this proposal.
