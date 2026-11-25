# TPL-41 verification

The focused regression passed with mixed ordinary and conventional fix
subjects. The helper emitted `bug_fix_count=2` as the only GitHub output
record and printed only the scalar count for the workflow diagnostic. Review
confirmed no workflow step consumes the removed `commits` output; the existing
numeric downstream gate is preserved. Strict OpenSpec 2.2.0 change validation
passed.

Hosted Regression Guard at `228ebf7` passed the corrected output step.
Its downstream check requires a new test under `tests/regression/`.
The focused test now follows that layout and references `FAIL-b7cf877c`.
[Hosted run](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/45657162229)
