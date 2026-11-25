# TPL-41 verification

## Regression Guard behavior

The focused regression passed with mixed ordinary and conventional fix
subjects. The helper emitted `bug_fix_count=2` as the only GitHub output
record and printed only the scalar count. Review confirmed no workflow step
consumes the removed `commits` output; the existing numeric downstream gate is
preserved. Strict OpenSpec 2.2.0 change validation passed before the additional
Python-suite discovery finding.

Hosted Regression Guard at `228ebf7` passed the corrected output step. The
hosted downstream check requires a new test under `tests/regression/`; the
focused test now lives there and references `FAIL-b7cf877c`.
[Hosted run](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/45657162229)

## Python suite discovery follow-up

A local Linux replay of the workflow against published `30b63f6` found that the newly placed regression
suite had no `.guardrails/python-test-floor.json` row. Added the exact suite
path `tests/regression`, setup `python3 -m pip install pytest`, test command
`python3 -m pytest tests/regression/ -v`, floor 1, and required applicability
to the inventory.

Linux Docker validation against this worktree:

- `python3 scripts/check_python_tests.py --repo .` — exit 0; 4 discovered
  suites, all covered.
- `python3 -m pytest tests/regression/ -v` — 1 passed.
- `python3 -m pytest tests/regression/ --collect-only -q -o addopts=""` —
  1 collected, meeting the floor.

Pytest emitted a cache-write warning because the checkout was mounted read-only;
the test itself passed. No repository cache or source file was written by the
container run.

## Remaining verification

Hosted rerun on draft PR #13 after this inventory correction is still pending.
The tasks remain open until the hosted Python discovery/test-floor and
Regression Guard checks pass at the corrected revision.
