# Evidence

## Hosted runner selector default

- Proof repository: [TheArchitectit/ci-runner-selector-proof-20261127](https://github.com/TheArchitectit/ci-runner-selector-proof-20261127) (public, isolated, contains only the selector proof workflow).
- Proof commit: `e5e6fa21d7f9877a5e99051cce96d3fb638754cb`.
- Exact workflow selector: `runs-on: ${{ fromJSON(vars.CI_RUNNER_LABELS || '["ubuntu-latest"]') }}`.
- Repository variable list: empty at verification time. The workflow's guard step for a non-empty `CI_RUNNER_LABELS` value was skipped.
- Run: [46069282587](https://github.com/TheArchitectit/ci-runner-selector-proof-20261127/actions/runs/46069282587), completed successfully at `2026-11-27T21:17:13Z`.
- Job: `selector`, id `140742066264`, completed successfully. Logs report `RUNNER_ENVIRONMENT=github-hosted`, `RUNNER_OS=Linux`, `RUNNER_ARCH=X64`, runner name `GitHub Actions 1000032561`, Ubuntu image `ubuntu-26.04`, and the explicit assertion that the exact selector fallback resolved to GitHub-hosted `ubuntu-latest`.
- Scope: proves GitHub evaluation and assignment for this exact selector when the repository variable is absent. It does not replace testing the template repository's configured UCS03 selector, fork guard, workflow-specific commands, or any image/client acceptance scenario below.

## Configured UCS03 fleet and source-test comparison

- Template main commit: [`e8949f5fed43bbeda75343bd48b9e68985a1eb9f`](https://github.com/TheArchitectit/agent-guardrails-template/commit/e8949f5fed43bbeda75343bd48b9e68985a1eb9f/checks).
- The 59 main and tag checks completed successfully. The runner banners identified 32 jobs on the primary UCS03 runner, 24 on spare2, and 3 on spare1. The nine workflows identified approved AIGGP revision `7d17d6f1baa4357110e57d3a6ffd55fbacdd77f7`.
- An AI05 local run and the UCS03 Python job reported the same outcomes for the 370 distinct pytest node IDs captured by the comparison. This is a partial log comparison, not an exhaustive proof for parameterized IDs or GitHub-only event behavior.
- The new parity job runs the exact local suite entry point on its configured runner. Its candidate run, hosted run, fork behavior, and MCP acceptance remain to be verified; no pending task is marked complete by the earlier main results.

## Remaining evidence

The complete local/hosted/UCS03 parity matrix, AI03-AI05 synthetic isolation, and real Codex MCP client initialize/list/call still require evidence. Tasks remain unchecked until the associated scenarios pass.
