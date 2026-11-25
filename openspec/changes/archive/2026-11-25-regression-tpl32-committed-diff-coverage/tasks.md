# Tasks

## 1. Add committed-range evaluation

- [x] 1.1 Add explicit base/head arguments, canonicalize revisions, and use unique merge-base-to-head scope for PRs or ancestor before-to-head scope for pushes. Reject missing, shallow, ambiguous, divergent-push, or empty ranges. Preserve local staged/unstaged options and evaluate their pattern patches independently.
- [x] 1.2 Add temporary Git history coverage for a benign committed sentinel, clean nonempty range, missing/empty/shallow/ambiguous history, divergent PR base-only content, and push divergence. Cover staged-only, unstaged-only, and mixed local patches where forbidden context cannot suppress a separate patch's finding. Run the regression tests.

## 2. Wire event ranges and retain local behavior

- [x] 2.1 Update `.github/workflows/regression-guard.yml` to pass exact PR base/head or push before/head SHAs, enable merge-base mode only for PRs, and retain `fetch-depth: 0`; both paths reject missing or empty ranges.
- [x] 2.2 Run `/tmp/agentqa-venv/bin/pytest tests/test_regression_check.py -q` and exercise the CI range CLI in temporary fixture histories. Verify staged-only, unstaged-only, mixed-patch, and existing local-mode behavior.
