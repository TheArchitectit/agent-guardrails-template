# Tasks

## 1. Reconcile each stale statement with evidence

- [x] 1.1 For Spec 07, compare the dated acceptance/headline claim with current implementation and test evidence; revise only the superseded text and retain residuals.
- [x] 1.2 For Spec 17, compare the dated dependency-state claim with current code/tests; preserve outstanding requirements and dates.
- [x] 1.3 For Spec 12, verify the current public-route registry and update the stale open-routes statement with scope limits.
- [x] 1.4 For Spec 16, verify current authorization/config secrecy source and named tests against the dated 2026-10-04 baseline; do not claim CI execution absent evidence.
- [x] 1.5 For Spec 23, reconcile the opening problem statement with its later test-closure section and name any remaining gap.

## 2. Check cross-document scope

- [x] 2.1 Confirm the resulting edits preserve dates, named references, and unresolved claims.
- [x] 2.2 Run the canonical Markdown/link checker for touched docs; see `evidence.md` for scoped results and the incomplete full-tree run.
- [x] 2.3 Confirm no `docs/platform-current-state.md` audit persistence text is changed; that item is handled by `reconcile-platform-current-state`.

## 3. Validation status

- [x] 3.1 Run pinned OpenSpec CLI 2.2.0 strict validation under Node 22 Linux. See `evidence.md`.
