# Proposal: Reconcile the canonical OpenSpec inventory

## Why

`openspec/INDEX.md` reports eight specs while the native tree contains fifteen; seven short-slug specs are omitted and overlapping short/verbose variants contain distinct requirements. A canonical inventory must identify files by full path and maintain status and archive provenance without collapsing requirements by name alone.

## What Changes

- Inventory all fifteen current spec paths and identify their distinct requirement sets.
- Reconcile the index count, status rows, and closure evidence against those files.
- Trace archived AIGGP packages to the outputs they produced, while separating archive-manifest corrections under `ARC-01`.
- Identify whether a checked-in generator or source exists for the current index before changing generated output.

## Capabilities

### New Capabilities

- `openspec-catalog-identity`: The OpenSpec catalog gives reviewers a complete, unambiguous view of specification files and their own status evidence.

### Modified Capabilities

None. This change adds no behavior contract to an existing capability.

## Impact

`openspec/INDEX.md`, the OpenSpec spec tree, and audit traceability. Preserve short and verbose spec files as distinct until requirement-level reconciliation establishes an accepted disposition. Do not change production implementation or imply that existing closure claims are fully verified.
