# Tasks

## 1. Correct the workflow output

- [x] 1.1 Replace the multiline commit-subject output with a helper that writes
  only `bug_fix_count` and prints its scalar value for diagnostics; preserve
  the current matching and downstream gate.
- [x] 1.2 Add a focused regression using mixed ordinary/fix subjects and
  validate every emitted output record is valid and only contains the consumed
  numeric count.

## 2. Verify

- [x] 2.1 Run the focused regression and validate the OpenSpec change strictly.
- [x] 2.2 Confirm the workflow still gates regression-test checks on the
  unchanged `bug_fix_count` output and that no step consumes `commits`.
- [x] 2.3 Add `tests/regression` to `.guardrails/python-test-floor.json` with
  its explicit pytest command, setup dependency, positive floor, and CI
  applicability; run the discovery gate and suite on Linux.
- [x] 2.4 Run the exact Regression Guard output and enforcement steps for
  draft PR #13 at the fixed revision. Record the run evidence from its
  assigned GitHub runner or the maintainer-authorized laptop replay.
