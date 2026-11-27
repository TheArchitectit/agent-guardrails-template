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

Event-specific parity and synthetic isolation across AI03, AI04 and AI05 remain separate qualification tasks. Model-backed Codex testing runs only on operator request. Tasks remain unchecked until all associated scenarios pass.

## Real MCP image and Codex client on AI05

- The production `mcp-server/deploy/Dockerfile` was built with rootless Podman 5.6.0 on AI05 from main `e8949f5fed43bbeda75343bd48b9e68985a1eb9f`. The `mcp-server` source tree is also identical in candidate `4b5af0b9a38a959ef5c7df693439794d237f5290` (`2e8d7593be68ad1186755fe8381b499bcafacc9f`).
- Image manifest digest: `sha256:5b43e9aee686160814dc4fb2de96df7a570e90ae8eee78110849af9fc2214bda`. Image ID: `sha256:b176adaef5db453ecb9bc6b89491f89001cf7c11d547499de1844f15948f4efa`.
- The unique scratch stack used synthetic credentials, temporary PostgreSQL and Redis, and a loopback-only endpoint. Nineteen bundled migrations applied. Direct MCP initialization negotiated `2025-03-26`, `tools/list` returned 50 tools, and `guardrail_advisor_list` returned nine advisors.
- A real Codex CLI `0.171.0-alpha.13.2` connected through a temporary SSH tunnel and completed `guardrail_advisor_list`, with nine advisors in its MCP result. The first attempt was refused by Codex's approval policy; the successful attempt explicitly enabled and authorized only that read-only test tool in temporary config. Raw Codex JSONL SHA-256: `a1da6fc6e3356bebd387b5f2bda49a4fe0f1005f8161febd304746d0b8d029a4`. The synthetic key was absent from the retained evidence.
- This is a one-time lab acceptance, not a per-build GitHub workflow gate. No AI05 credential or image-engine socket is available to UCS03 jobs. Model-backed Codex testing runs only on demand. The automated workflow uses the deterministic MCP protocol probe.

## Token-free image harness qualification on AI05

- Candidate source `821cb8e28d542f018751321d29d8168f9b14e916` ran the image harness through `bash` on AI05. It built image ID `sha256:9f318656eeca69051bf870e3b0f83aa21b41910d56d9d21deb21d8fda572e505`, manifest digest `sha256:2b175602c0a92be3d85d33b7ec43572c507266f66c8f70bc8097fe171228761d`, applied migrations, and passed deterministic MCP initialize, tool listing and the nine-advisor read-only call. The receipt recorded `codex=not-run`, so no model was invoked.
- Direct invocation first failed before any container was created because the initial commit recorded the script as non-executable. The file mode is corrected in the follow-up. This earlier run exercised the standalone harness. The per-build worker uses its own fixed guest entry point and signed receipts.
- Direct invocation was repeated successfully on AI05 for `28febdf2beed26b079d58a9d2dfccd26638c9516`, using the corrected executable mode, bounded resources, and fail-closed protocol checks. The receipt records image ID `sha256:d197d32c3073c08a7fc13924a3f5b7b55a361e026aa5f27ef91a5796caa64465`, manifest digest `sha256:2f017a37133044984bc4e27003150508eea8bcd28533732dc469a9092aca8f2a`, `mcp_protocol=passed`, and `codex=not-run`.

## UCS03 and AI05 worker qualification

The initial [branch push](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/46080207476/job/140777179526) for `3de91ef24937c5374e07610a25b809014e9e753a` reached AI05. Its offline guest failed at the production Dockerfile's Go module download. The independently verified Ed25519 receipt recorded the correct commit, check ID, AI05 host, offline VM profile, failure and `codex=not-run`. The first [PR job](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/46080211523/job/140777192832) exposed a missing runner-local service map. After the map was installed on all three UCS03 runners, the [PR retry](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/46080211523/job/140778448109) reached AI05 and reproduced the dependency failure on synthetic merge commit `cb23d4dffa847350f573f211d8764cfb13ba481a`.

After the sealed guest gained its own verified Go module cache, the [branch
push job](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/46082099235/job/140783170514)
passed for `a39ebc26a708575d670ec516318d5dc7d811a9f1`. The GitHub job and
runner banner agreed on `ucs03-guardrails-spare2`. Independent Ed25519
verification bound the check to AI05, its offline KVM guest and image
`sha256:6d7e15d972da0a354e062190711e223897f792c47933bb28e7138dad949976d8`.
MCP initialized, listed 50 tools and returned nine advisors. The worker
removed the named run directory before signing. The receipt recorded
`codex=not-run`; its file SHA-256 was
`1e5b13400a811531dd1f7d9b8af21a6a73a0acc5a4ec5029d6b5f87626964b31`.

The same Guardrails commit also passed the canonical local CI command in the
[default hosted job](https://github.com/TheArchitectit/ci-runner-selector-proof-20261127/actions/runs/46082172369/job/140783403023).
The proof repository had no `CI_RUNNER_LABELS` variable. Its job banner
identified `GitHub Actions 1000032572`; the hosted-environment assertion and
candidate/AIGGP identity checks passed. The guard for an unexpected selector
override was skipped as intended. Proof commit
`81bd5c01da652c78465339f9e9ace27534337c25`; full log SHA-256
`9eb1d4c9ec5f799d578763e87c33af009303223f1fd7aaf29fe8da696dcf0d95`.

The [PR job on the UCS03 primary](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/46082103594/job/140783183592)
passed on its own signed merge commit
`b901eaf3abb0e09ae896ec94d141883cb97f4588`. The Git commit object named
`a39ebc26a708575d670ec516318d5dc7d811a9f1` as its second parent.
AI05 returned digest
`sha256:c8285e182c2091708d60fd9cee27457b47fcc66393edcf310fd4976dccad48e2`,
50 tools, nine advisors, scoped cleanup and `codex=not-run`. The downloaded
receipt's SHA-256 was
`da64e69c2eb32c31a3f1dfa5373fb1cf4c92469ae5e657f030c59284ecd88264`.

The same source and approved AIGGP pin ran locally on AI05, in a fresh
two-CPU builder, from `2026-11-27T23:25:45Z` to `23:35:48Z`. It exited
zero. The local and hosted logs recorded the same four Python suite outcomes:
68 passed and one skipped, 12 passed, 289 passed and 32 passed. Both ran
the Docker CLI context-transport test and skipped the optional live-engine
context fixture. Both ran the Go inventories and their builds. The separate
worker receipts above cover the live MCP image. The full local log SHA-256 is
`279a62d6d8c31769fbb6c229c36f93a7978718c28a6d20bf233ef31fc1db5db8`.

A [non-release tag push](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/46082853450/job/140785604355)
exercised the `refs/tags/ci-proof/ai05-worker-a39ebc2` path on
`ucs03-guardrails-spare1`. Its signed commit was
`a39ebc26a708575d670ec516318d5dc7d811a9f1`, with image digest
`sha256:920a8a48697c185e88add2100bd9b64a01d85e320495a273453c3abc2dafb7b9`.
It passed the same MCP assertions and recorded `codex=not-run`.
The branch, PR and tag jobs together exercised all three UCS03 fleet members.
Each digest identifies the particular per-run image, which carries a unique
ownership label. No persistent application container is deployed by this check.
