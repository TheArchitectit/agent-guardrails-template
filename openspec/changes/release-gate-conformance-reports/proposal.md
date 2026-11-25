# Proposal: Gate releases on current conformance reports

## Why

The repository has conformance harnesses and version-invalidation helpers, but no required version-bound report/promotion gate is evidenced. Harness results alone do not support a release verification claim.

## What Changes

- Require immutable conformance reports bound to corpus, classifier, policy, and supported runtime versions.
- Require release workflows to produce and evaluate current reports.
- Block verified promotion for missing, stale, or failing results.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `prompt-injection-evidence-and-conformance`: require a fresh passing report for verified release claims.
- `semantic-filtering-evidence-and-conformance`: invalidate stale claims and gate promotion on a current report.

## Impact

Release workflows, conformance artifacts, and verified promotion status. It does not redefine fixture content or evidence envelope fields.
