# Proposal

## Why

Release Coverage run `45556758733` on main `943bcf6` correctly reports that HEAD is not release `v3.8.11` at `c7c4044`, where VERSION and its annotated tag point. The current gate enforces the declared tag-at-HEAD contract; approved release metadata and the authorized release operation are pending.

## What Changes

- Prepare VERSION and changelog metadata for the approved commit under project policy.
- Test the existing gate with isolated matching and stale-tag fixtures without modifying real refs or publishing.
- Keep actual tag creation and publication separate as an authorized release-owner follow-up; do not weaken Release Coverage.

## Capabilities

### New Capabilities

- `release-metadata-delivery`: Validate that release metadata and annotated tag identify the same reviewed commit.

### Modified Capabilities

None.

## Impact

Affects VERSION, CHANGELOG, and release-coverage test procedures. Tests must not create a tag or release in the real repository or contact a remote. No implementation or release action is included in this planning package.
