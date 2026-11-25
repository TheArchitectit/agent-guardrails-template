# Proposal: Reconcile stale numbered design claims

## Why

The dated audit identifies numbered design documents whose opening status or problem statements conflict with current code and named tests: Specs 07 and 17 need bounded acceptance claims; Spec 12 has a stale open-routes statement; Spec 16's 2026-10-04 baseline conflicts with current authorization/config secrecy evidence; and Spec 23's opening problem statement conflicts with its later closure section. These are distinct documentation-drift cases, not one implementation finding.

## What Changes

- Reconcile each identified dated claim against source and named test evidence, preserving the original dates and any remaining limitations.
- Update only the specific stale statement and link the supporting implementation/test references.
- Keep the separate platform-current-state persistence claim under `INV-02`.

## Capabilities

### New Capabilities

- `numbered-design-status-evidence`: Numbered design documents retain dated context while making current evidence and residual requirements reviewable.

### Modified Capabilities

None. This change adds no behavior contract to an existing capability.

## Impact

`docs/specs/07-*`, `12-*`, `16-*`, `17-*`, and `23-*` as applicable. Do not infer that a cited test was run in a required release gate or erase requirements that remain open.
