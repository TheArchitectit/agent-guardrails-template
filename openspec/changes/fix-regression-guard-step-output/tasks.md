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
- [ ] 2.3 Run the hosted Regression Guard on draft PR #13 at the fixed
  revision. Verify the output step passes and record the run evidence.
