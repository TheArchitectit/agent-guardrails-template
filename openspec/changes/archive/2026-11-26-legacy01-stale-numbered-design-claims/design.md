# Design

## Context

The audit's `LEGACY-01` groups several dated design statements that have separate evidence and residuals. The OpenSpec requirements and platform-current-state inventory are different documents and remain outside this change.

## Goals / Non-Goals

**Goals:** Make each named document's dated status statement consistent with the cited current source/test evidence, while retaining unresolved acceptance and test-gating limits.

**Non-Goals:** Reopen or close requirements based only on documentation edits, rewrite the 261-row dated historical ledger, or combine the separate `INV-02` platform persistence reconciliation.

## Decisions

- Handle each numbered spec independently, using a small source/test evidence note and explicit residuals.
- Preserve historical dates and distinguish source inspection from tests actually run.
- Do not treat cross-reference to current code as evidence of external consumer/runtime coverage.

## Risks / Trade-offs

- Current implementation may be narrower than a stale problem statement suggests; retain requirements not established by the cited evidence and record owner acceptance as open where necessary.
