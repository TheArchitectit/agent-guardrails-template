# Template QA Re-audit

**Review date:** 2026-11-25 UTC
**Template main snapshot:** `943bcf65055128e338b696dc40d74e79ac33449e`
**Prior baseline:** `c7c40443ef06b8a8cacfdc52c34384d6d0eef134`

This is a point-in-time review. It distinguishes successful checks from unavailable or queued results and treats absent evidence as unknown rather than a pass.

## Hosted workflow state at current main

| Workflow | Run | Observed result |
|---|---:|---|
| Trusted Main Validation | [45556758751](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/45556758751) | Success |
| Regression Guard | [45556758686](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/45556758686) | Success; review found a clean-checkout committed-diff blind spot, tracked as TPL-32 |
| Documentation Standards | [45556758670](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/45556758670) | Success |
| Release Coverage | [45556758733](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/45556758733) | Failed by configured tag-at-HEAD check: current `943bcf6` is not release `v3.8.11`, which points to `c7c4044` |
| Team Validation | [45556758721](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/45556758721) | Queued at refreshed observation; no result claimed |
| Secret Validation | [45556758743](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/45556758743) | Failed during scanner installation; scan did not run and no leak was reported |
| Guardrails Compliance | [45556758692](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/45556758692) | Success |

Release Coverage's result is consistent with the current explicit contract. The review did not create release metadata or a tag. Do not weaken the gate to make the current non-release commit pass; TPL-35 tracks approved release metadata and publication.

## Verified source and QA findings

| ID | Finding and source evidence | Tracked proposal |
|---|---|---|
| TPL-32 | `.github/workflows/regression-guard.yml` runs `python scripts/regression_check.py --all`. In the script, `--all` discovers paths from both staged and unstaged changes, but patch content is read from `git diff --cached`; an unstaged-only added file can therefore be discovered without its added content being evaluated. In the hosted clean-checkout job both worktree diffs are empty, so the committed HEAD patch is also not evaluated. | `regression-tpl32-committed-diff-coverage` |
| TPL-33 | No `.dockerignore` is present. The documented build uses `mcp-server` as context and `mcp-server/deploy/Dockerfile`, whose builder runs `COPY . .`. Eligible local context files are copied into a builder layer. The final image separately copies selected artifacts; this is a build-context hygiene finding, not a claim that arbitrary files enter the runtime image. | `docker-tpl33-bound-build-context` |
| TPL-34 | `mcp-server/Makefile` has a manual `govulncheck ./...` target; no workflow invokes it. QA-05 remediated the dated source-reachable findings, but recurring automated advisory review is not wired into CI. | `go-tpl34-automate-source-advisory-checks` |
| TPL-35 | Release Coverage fails at `943bcf6` because it is not the `v3.8.11` tagged commit at `c7c4044`. The current check is working as configured; approved release metadata is pending. | `release-tpl35-publish-approved-main` |
| TPL-38 | Secret Validation run `45556758743`, job `139010560787`, failed in “Run Gitleaks” while installing v8.24.3 because `/tmp/gitleaks.tmp` already existed. Installation failed before scanning; no leak result exists. | `isolate-gitleaks-runner-install` |
| IDE-01 | `ide/vscode-extension/src/extension.ts` calls `testConnection()` during activation without first calling `updateConfiguration()`. `GuardrailClient` initializes config in `src/utils/client.ts::updateConfiguration`, while `fetch()` dereferences it. | `ide01-initialize-vscode-client` |
| IDE-02 | IDE clients/docs default to port 8095; MCP REST/health/IDE routes use web port 8081, while MCP protocol uses 8080 (`mcp-server/internal/config/config.go`, `mcp-server/README.md`). | `ide02-align-ide-rest-endpoint` |
| IDE-03 | `mcp-server/internal/web/handlers_ide.go` binds `selection`; VS Code, Vim, Neovim, and JetBrains clients send `code`. | `ide03-align-selection-payload` |
| PKG-01 | Pi bridge imports/types `@modelcontextprotocol/sdk` but `pi-extension/package.json` and lockfile do not declare it. Locked test baseline: 196/197; adding documented SDK in a temporary copy: 197/197. | `pkg01-declare-pi-mcp-sdk-dev-dependency` |
| PKG-02 | Manifest/package lock name is `@architectit/pi-guardrails`; README and installer help use `@thearchitectit/pi-guardrails`. | `pkg02-align-pi-package-name` |

The eight historical findings assigned for remediation in the review worktree are QA-02, QA-03, QA-04, CI-01, CI-02, DIST-01, INSTALL-01, and INSTALL-02. Their OpenSpec packages are proposals and remain unverified until their tasks are implemented and tests pass. QA-05 is complete under the [official archived evidence](../changes/archive/2026-11-25-qa05-source-reachable-go-toolchain-remediation/evidence.md).

## Historical audit reclassification

- QA-05 is superseded by the post-baseline Go 1.26.11 evidence. The latest scan records zero source-reachable findings and four required-module-only findings; these are not equivalent to four source-path findings.
- The old c7 hosted workflow success does not describe current `943bcf6` release status. Current Release Coverage correctly reports a tag mismatch; several current workflow runs were still queued when observed.
- Secret Validation's failure is a scanner installation failure, not a leak finding and not a clean scan. The new TPL-38 proposal tracks unique job-owned scratch, integrity/version verification, and fail-closed behavior.
- INV-01, ARC-01, QA-01, DEP-01/02, OPS-01..04, INV-02, and RUN-01..06 remain inventory, runtime evidence, or deployment readiness proposals. See the [published gap register](2026-11-25-template-gap-register.md), [Open/closed/coded review](2026-11-25-template-open-closed-coded.md), and the [44 published + 27 active requirement trace](2026-11-25-template-requirement-trace.md).
- Older AIGGP workflow failures are not promoted to current findings without runs at current AIGGP HEAD. Only Drift Scan run `45431717255` and Fleet App Enrollment run `45431618054` were observed succeeding at `fd2c11f`; broader current-head workflow statuses remain unknown/not observed.

## Commands and limits

- Official OpenSpec CLI 2.2.0, `openspec validate --all --strict` at template main: 12 pass, 7 fail. Failures are seven unindexed short-spec files with strict scenario-format warnings.
- Current-main hosted evidence is limited to the run states shown above; queued results are intentionally not inferred.
- Source checks used `rg`, `sed`, and `git ls-files` for the named workflow, script, IDE, package, and Docker paths. No exploit reproduction was performed.
