# Inspect complete commit bodies for attribution

## Why

The Guardrails Compliance workflow uses a multi-character string as the input set to `tr`. GNU `tr` treats its leading dashes as an invalid option. The surrounding process substitution lets the loop consume no commits and print a successful result. The defect was reproduced locally at draft #13 head `109c952`.

## What Changes

- Obtain the event-scoped commit IDs through the existing range helper.
- Inspect each commit's complete message and retain the existing advisory attribution policy.
- Fail the job when the range or a commit cannot be read, and state explicitly when the computed range is empty.

## Capabilities

### New Capabilities

- `commit-attribution-records` requires complete per-commit attribution checks.

### Modified Capabilities

None.

## Impact

The attribution job in `.github/workflows/guardrails-lint.yml`, the event range contract, and its focused regression coverage.
