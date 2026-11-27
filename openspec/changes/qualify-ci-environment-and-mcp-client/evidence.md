# Evidence

## Hosted runner selector default

- Proof repository: [TheArchitectit/ci-runner-selector-proof-20261127](https://github.com/TheArchitectit/ci-runner-selector-proof-20261127) (public, isolated, contains only the selector proof workflow).
- Proof commit: `e5e6fa21d7f9877a5e99051cce96d3fb638754cb`.
- Exact workflow selector: `runs-on: ${{ fromJSON(vars.CI_RUNNER_LABELS || '["ubuntu-latest"]') }}`.
- Repository variable list: empty at verification time. The workflow's guard step for a non-empty `CI_RUNNER_LABELS` value was skipped.
- Run: [46069282587](https://github.com/TheArchitectit/ci-runner-selector-proof-20261127/actions/runs/46069282587), completed successfully at `2026-11-27T21:17:13Z`.
- Job: `selector`, id `140742066264`, completed successfully. Logs report `RUNNER_ENVIRONMENT=github-hosted`, `RUNNER_OS=Linux`, `RUNNER_ARCH=X64`, runner name `GitHub Actions 1000032561`, Ubuntu image `ubuntu-26.04`, and the explicit assertion that the exact selector fallback resolved to GitHub-hosted `ubuntu-latest`.
- Scope: proves GitHub evaluation and assignment for this exact selector when the repository variable is absent. It does not replace testing the template repository's configured UCS03 selector, fork guard, workflow-specific commands, or any image/client acceptance scenario below.
- The full candidate suite subsequently passed on GitHub-hosted Ubuntu 26.04 with the variable absent: [run 46071211496](https://github.com/TheArchitectit/ci-runner-selector-proof-20261127/actions/runs/46071211496), [job 140748324034](https://github.com/TheArchitectit/ci-runner-selector-proof-20261127/actions/runs/46071211496/job/140748324034). It checked out candidate `4b5af0b9a38a959ef5c7df693439794d237f5290` and approved AIGGP `7d17d6f1baa4357110e57d3a6ffd55fbacdd77f7`, then ran the canonical local CI entry point. Its log SHA-256 is `d9b4503ab142f6f0e364cdf21110feeec8157c0dd8d1229f3e1369418af3ec12`. The opt-in engine-backed context test was skipped; the Docker CLI transport test ran.

## Configured UCS03 fleet and source-test comparison

- Template main commit: [`e8949f5fed43bbeda75343bd48b9e68985a1eb9f`](https://github.com/TheArchitectit/agent-guardrails-template/commit/e8949f5fed43bbeda75343bd48b9e68985a1eb9f/checks).
- The 59 main and tag checks completed successfully. The runner banners identified 32 jobs on the primary UCS03 runner, 24 on spare2, and 3 on spare1. The nine workflows identified approved AIGGP revision `7d17d6f1baa4357110e57d3a6ffd55fbacdd77f7`.
- An AI05 local run and the UCS03 Python job reported the same outcomes for the 370 distinct pytest node IDs captured by the comparison. This is a partial log comparison, not an exhaustive proof for parameterized IDs or GitHub-only event behavior.
- The new parity job runs the exact local suite entry point on its configured runner. Earlier main results alone do not establish candidate, hosted, fork, or MCP acceptance.

### PR candidate `4b5af0b9a38a959ef5c7df693439794d237f5290`

- All 54 jobs across the six PR workflows completed successfully, and each job's GitHub runner assignment named the UCS03 Guardrails primary, spare1, or spare2. [Candidate checks](https://github.com/TheArchitectit/agent-guardrails-template/commit/4b5af0b9a38a959ef5c7df693439794d237f5290/checks).
- [Local CI parity job](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/46070612419/job/140746405119) ran on `ucs03-guardrails-spare2`: the PR event-range checks and `scripts/run_local_ci.sh` both passed; coverage was uploaded as an Actions artifact. The manual-dispatch explanation step was skipped on this PR as intended.
- Six workflow-specific AIGGP verification logs reported `7d17d6f1baa4357110e57d3a6ffd55fbacdd77f7`. The `validate-teams` export and upload steps were skipped because this was not a main push; the regression-test-presence step was skipped because no bug-fix commit was detected. These skips are not tests of their conditional paths.
- Runtime fork isolation was not exercised by this same-repository PR. A green result here does not attest to fork behavior.
- The same candidate's local entry point completed on AI05 in an isolated builder in 7 minutes 39 seconds, exit 0. Its successful-run log SHA-256 is `aeaf72d32a6a681aea5f2e95acd263c8dab6f8ca410cace19416d6c12ff29a3f`; Go discovered counts were 907, 12, and 11, all floors and race/build checks passed. The optional engine-backed context fixture was skipped; the CLI transport check ran. Three earlier local setup attempts stopped before tests because their isolated builder lacked prerequisites or started in the wrong directory; the full private log retains those attempts.

## Remaining evidence

The complete local/hosted/UCS03 parity matrix, AI03-AI05 synthetic isolation, and automated per-build image and Codex acceptance still require evidence. Tasks remain unchecked until all associated scenarios pass.

## Real MCP image and Codex client on AI05

- The production `mcp-server/deploy/Dockerfile` was built with rootless Podman 5.6.0 on AI05 from main `e8949f5fed43bbeda75343bd48b9e68985a1eb9f`. The `mcp-server` source tree is also identical in candidate `4b5af0b9a38a959ef5c7df693439794d237f5290` (`2e8d7593be68ad1186755fe8381b499bcafacc9f`).
- Image manifest digest: `sha256:5b43e9aee686160814dc4fb2de96df7a570e90ae8eee78110849af9fc2214bda`. Image ID: `sha256:b176adaef5db453ecb9bc6b89491f89001cf7c11d547499de1844f15948f4efa`.
- The unique scratch stack used synthetic credentials, temporary PostgreSQL and Redis, and a loopback-only endpoint. Nineteen bundled migrations applied. Direct MCP initialization negotiated `2025-03-26`, `tools/list` returned 50 tools, and `guardrail_advisor_list` returned nine advisors.
- A real Codex CLI `0.171.0-alpha.13.2` connected through a temporary SSH tunnel and completed `guardrail_advisor_list`, with nine advisors in its MCP result. The first attempt was refused by Codex's approval policy; the successful attempt explicitly enabled and authorized only that read-only test tool in temporary config. Raw Codex JSONL SHA-256: `a1da6fc6e3356bebd387b5f2bda49a4fe0f1005f8161febd304746d0b8d029a4`. The synthetic key was absent from the retained evidence.
- This is a one-time lab acceptance, not a per-build GitHub workflow gate. No AI05 credential or image-engine socket is available to UCS03 jobs. The user chose to skip model-backed testing on unattended builds; deterministic image/protocol testing still needs trusted, isolated per-build wiring. Do not report this as automated coverage.
