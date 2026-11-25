# Verify the Gitleaks CLI before scanning

## Why

Secret Validation previously collided with another job's Gitleaks install in shared scratch. The pinned Action now uses isolated scratch, but it can download and execute the scanner without checking the archive digest or the installed version. The maintainer selected direct CLI execution and local verification so scanner acceptance does not depend on a licensed hosted Action.

## What Changes

- Add a repository-owned command that verifies the pinned Gitleaks release archive and version before scanning.
- Use the command on the existing self-hosted workflow and on the laptop. Record local verification separately from GitHub job results.
- Preserve full-history checkout, commit-range coverage, redaction, read-only permissions, and fail-closed results. Report the result in the job summary.

## Capabilities

### New Capabilities

- `gitleaks-runner-installation` provides an isolated, integrity-verified scanner for both CI and local runs.

### Modified Capabilities

None.

## Impact

The Secret Validation job replaces its licensed GitHub Action with the direct Gitleaks CLI. Action license and scan-token inputs and PR comment integrations are removed. Other Secret Validation jobs retain their current behavior.
