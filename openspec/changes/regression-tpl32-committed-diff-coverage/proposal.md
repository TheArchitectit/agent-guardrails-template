# Proposal

## Why

`.github/workflows/regression-guard.yml` runs `regression_check.py --all` in a clean checkout, but that mode reads staged and unstaged working-tree diffs. It does not inspect the committed PR or push patch. Additionally, `--all` discovers staged and unstaged paths but currently reads only staged patch content for pattern matching.

## What Changes

- Add explicit base/head range evaluation for PR and main-push CI events, with missing or empty CI ranges failing closed.
- Preserve and correctly test staged, unstaged, and combined local modes.
- Add safe temporary-Git-history tests for committed and working-tree matches plus clean controls.

## Capabilities

### New Capabilities

- `regression-guard-committed-diff`: Evaluate committed CI patches and local working-tree changes without vacuous success.

### Modified Capabilities

None.

## Impact

Affects `scripts/regression_check.py`, `.github/workflows/regression-guard.yml`, and its tests. Existing local workflows remain supported. No implementation is included in this planning package.
