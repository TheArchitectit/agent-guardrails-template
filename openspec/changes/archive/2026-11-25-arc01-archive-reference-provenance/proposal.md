# Proposal: Reconcile archive manifest provenance

## Why

The archived AIGGP README reports six unique short-spec outputs, omits `isolation-levels`, and package headers refer to a retirement record absent from this checkout. These are historical reference and manifest defects distinct from `INV-01`'s current spec catalog.

## What Changes

- Reconcile the archive README's count and list against the four archived AIGGP packages and their actual delta paths.
- Repair or remove only the stale local retirement reference after checking the archived package headers and available historical records.
- Preserve archive package contents and map provenance to current specs without changing the canonical inventory.

## Capabilities

### New Capabilities

- `openspec-archive-provenance`: The archived-change manifest preserves verifiable links from historical packages to their produced specification files.

### Modified Capabilities

None. This change adds no behavior contract to an existing capability.

## Impact

`openspec/changes/archive/` package README and metadata references. The separate current catalog/count and short/verbose requirement identity remain scoped to `INV-01`.
