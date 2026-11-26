# TPL-35 local release rehearsal

On 2026-11-25, added `tests/test_release_coverage_workflow.py`. It extracts and
executes the actual `Verify release coverage` block from
`.github/workflows/release-coverage.yml` in isolated temporary repositories.
The template test follows the existing local workflow-test convention; when
run against AIGGP, it used AIGGP's existing structural `tests/workflow_read.py`
reader via the harness setting.
The fixtures have no remotes, and the test verifies their refs are unchanged
by the gate. It exercises an annotated tag at HEAD, missing tag, lightweight
tag, stale annotated tag at parent, missing changelog heading, generated-only
skip, and a mixed generated-plus-product change with stale tag.

Command: `python3 -m pytest -q tests/test_release_coverage_workflow.py`, run in
a disposable `aiggp-pr41-lead:20261125` test container with transient
`py3-pytest`. Result: **7 passed**. Matching annotated tag and generated-only
fixtures passed; all five negative controls failed as expected. The missing-notes case commits its changelog before tagging, and fixture commands isolate inherited Git overrides.
The tested step contains no publication command; the test invokes only this
coverage block and uses temporary repositories without remotes.

Strict validation command: `openspec validate
release-tpl35-publish-approved-main --strict`, run with Node 22.23.5 and
`@fission-ai/openspec@2.2.0` in a disposable container. Result: **valid**.

The worktree remained detached at
`db0603c927c32b63124aa944c264200b45cd3af2`. Its existing annotated
`refs/tags/v3.8.11` remained object
`429f0b418684fc98eb2c1b7d81419a1e1a14b8ee`, peeling to
`c7c40443ef06b8a8cacfdc52c34384d6d0eef134`; `VERSION`, `CHANGELOG.md`, and
both release workflows have no diff. No repository tag, remote, or published
release was changed. Approval and release metadata preparation remain open.
