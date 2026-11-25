# Tasks

## 1. Reconcile each stale statement with evidence

- [ ] 1.1 For Spec 07, compare the dated acceptance/headline claim with current implementation and test evidence; revise only the superseded text and retain residuals.
- [ ] 1.2 For Spec 17, compare the dated dependency-state claim with current code/tests; preserve outstanding requirements and dates.
- [ ] 1.3 For Spec 12, verify the current public-route registry and update the stale open-routes statement with scope limits.
- [ ] 1.4 For Spec 16, verify current authorization/config secrecy source and named tests against the dated 2026-10-04 baseline; do not claim CI execution absent evidence.
- [ ] 1.5 For Spec 23, reconcile the opening problem statement with its later test-closure section and name any remaining gap.

## 2. Check cross-document scope

- [ ] 2.1 Confirm the resulting edits preserve dates, named references, and unresolved claims; run Markdown/link checks for touched docs.
- [ ] 2.2 Confirm no `docs/platform-current-state.md` audit persistence text is changed; that item is handled by `reconcile-platform-current-state`.
