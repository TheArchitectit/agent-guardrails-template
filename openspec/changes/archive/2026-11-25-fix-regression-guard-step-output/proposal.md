# Proposal

## Why

The Regression Guard workflow writes all PR commit subjects to `GITHUB_OUTPUT`
as a single-line `commits=` value. Multiple subjects create invalid output
records, so the step fails before its regression-test check can use the valid
numeric bug-fix count.

## What Changes

- Stop publishing the unused commit-subject output while preserving
  `bug_fix_count` and its existing consumers.
- Add a focused regression for mixed ordinary and conventional fix subjects;
  assert only valid output records are emitted.

## Capabilities

### New Capabilities

- `regression-guard-step-output`: Emit only valid, consumed GitHub Actions step
  outputs from PR commit analysis.

### Modified Capabilities

None.

## Impact

This changes the Regression Guard workflow's bug-fix analysis step and adds a
small helper plus focused tests. It does not change commit-range selection,
bug-fix classification, or the regression-test gate.
