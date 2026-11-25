# Template OpenSpec review and gap register

**Reviewed revision** `943bcf65055128e338b696dc40d74e79ac33449e` on `main`

**Review date** 2026-11-25 UTC

The review reconciles all 44 requirements in the 15 specs published at the
reviewed revision, four active plans, and four existing archive packages.
The [open, closed, and coded inventory](2026-11-25-template-open-closed-coded.md)
records implementation paths and test evidence by requirement. The
[QA re-audit](2026-11-25-template-qa-reaudit.md) records the current hosted
checks, integration findings, and release metadata status.

The 31 earlier findings and seven additional independently actionable findings
from the initial review each have a full OpenSpec package. The implementation
pass added TPL-39 for the Windows Docker pilot and TPL-40 for Python
test discovery. The [combined plans](superseded-changes/README.md)
preserve the earlier grouping and historical checkbox state. QA-05 was the sole
verified and archived item in the initial review. QA-01 and TPL-32 were
subsequently implemented and verified during the implementation pass.

## Review procedure

The item packages follow the official spec-driven workflow with the installed
OpenSpec 2.2.0 CLI. Each has scaffold metadata, a proposal, a design, normative
scenario deltas, and task checkboxes. Verify planning artifacts with
`openspec status --all` and `openspec validate --changes --strict`. Use the
full and archived strict checks to compare the published tree and historical
archives. Apply and verify each implementation against its acceptance tasks
before syncing and archiving the completed change. [Official workflow](https://github.com/Fission-AI/OpenSpec/blob/main/docs/workflows.md)

At the reviewed revision the full strict check passed 12 entries and failed
seven published short-slug specs for scenario formatting. The item proposals
preserve those unresolved baseline failures for QA-01. The completed QA-05
archive adds one sourced evidence specification. The published count in this
review branch is 16, and the published baseline under review was 15.

The initial review commit had 37 active changes with complete planning artifacts. All
37 pass strict change validation, and all five archives pass strict archive
validation. Full-tree strict validation passes 46 of 53 entries, with the same
seven baseline spec failures. Each new proposal was peer reviewed against the
finding, canonical requirement, acceptance scenarios, and implementation tasks.

## Item register

| ID | State | Owner role | Observed gap or evidence | Full OpenSpec |
|---|---|---|---|---|
| `INV-01` | Open | OpenSpec maintainers | index says 8 while tree has 15 specs; reconcile all requirement variants and test indexed count equals filesystem inventory | [inv01-canonical-openspec-inventory](../changes/inv01-canonical-openspec-inventory/proposal.md) |
| `ARC-01` | Open | OpenSpec maintainers | The four 2026-11-06 archive packages publish seven short-slug outputs. The archive README says six, omits `isolation-levels`, and cites a missing retirement record. | [arc01-archive-reference-provenance](../changes/arc01-archive-reference-provenance/proposal.md) |
| `QA-01` | Verified and archived | QA/OpenSpec maintainers | Structured 25 scenarios across the seven failing specs. Strict validation passes for the complete tree. | [Archived QA-01 evidence](../changes/archive/2026-11-25-qa01-openspec-strict-scenario-structure/evidence.md) |
| `QA-02` | Verified and archived | Go QA maintainers | Four direct permission fixtures set and verify mode; auth/config/secretfile/mutationkill passed fresh under Go 1.26.11 Linux with umasks 0022 and 0077 | [make-permission-tests-umask-independent](../changes/archive/2026-11-25-make-permission-tests-umask-independent/evidence.md) |
| `QA-03` | Open | Python QA maintainers | 363 Python deprecation warnings, primarily `datetime.utcnow`; aware timestamps and unchanged JSON keys/terminal Z | [use-timezone-aware-python-utc](../changes/use-timezone-aware-python-utc/proposal.md) |
| `QA-04` | Open | JS package/QA maintainers | npm audit snapshot: Pi 9 dev entries (3 moderate/4 high/2 critical); VS Code 14 dev (4 moderate/10 high); production-only zero | [audit-javascript-dev-dependencies](../changes/audit-javascript-dev-dependencies/proposal.md) |
| `QA-05` | Verified and archived | Go platform/QA maintainers | Merged in #12 at `943bcf6`. Source paths fell from 45 to 0 on the same advisory database. Four module-only findings remain. The official archive publishes the evidence contract. | [qa05-source-reachable-go-toolchain-remediation](../changes/archive/2026-11-25-qa05-source-reachable-go-toolchain-remediation/proposal.md) |
| `CI-01` | Open | CI/Python maintainers | GitLab invokes missing root `requirements.txt`; use tracked dependency metadata and verify clean-checkout Python jobs | [fix-gitlab-requirements-inputs](../changes/fix-gitlab-requirements-inputs/proposal.md) |
| `CI-02` | Open | CI/Go maintainers | GitLab MCP tests and Jenkins MCP tests and Team CLI build still rely on a Go 1.21 declaration. The MCP module requires 1.26.11 and the Team CLI requires 1.23.2. | [align-legacy-ci-go-toolchains](../changes/align-legacy-ci-go-toolchains/proposal.md) |
| `DIST-01` | Open | CLI maintainers | install copies Go binary only; default Python backend is checkout/cwd-relative; verify outside-checkout install and override | [package-team-cli-python-backend](../changes/package-team-cli-python-backend/proposal.md) |
| `IDE-01` | Open | IDE extension maintainers | activation calls `testConnection` before config initialization; verify successful initial configured connection | [ide01-initialize-vscode-client](../changes/ide01-initialize-vscode-client/proposal.md) |
| `IDE-02` | Open | IDE/server maintainers | IDE URLs default 8095, REST server defaults 8081; verify defaults target REST listener | [ide02-align-ide-rest-endpoint](../changes/ide02-align-ide-rest-endpoint/proposal.md) |
| `IDE-03` | Open | IDE/server maintainers | server binds `selection`, clients send `code`; verify request succeeds through each supported client | [ide03-align-selection-payload](../changes/ide03-align-selection-payload/proposal.md) |
| `INSTALL-01` | Open | Installer maintainers | `path.lstrip("./")` strips `.claude` hidden component; preserve hidden path and contain under configured root | [preserve-hidden-agent-install-paths](../changes/preserve-hidden-agent-install-paths/proposal.md) |
| `INSTALL-02` | Open | Installer maintainers | Windsurf `.windsurfrules` file treated as a directory; direct file target and verify bulk/directory installs unchanged | [fix-windsurf-single-skill-target](../changes/fix-windsurf-single-skill-target/proposal.md) |
| `PKG-01` | Open | Pi package maintainers | bridge imports MCP SDK absent from manifest/lock; locked test 196/197, with documented SDK 197/197; preserve optional host behavior | [pkg01-declare-pi-mcp-sdk-dev-dependency](../changes/pkg01-declare-pi-mcp-sdk-dev-dependency/proposal.md) |
| `PKG-02` | Open | Pi package maintainers | package metadata is `@architectit/pi-guardrails`, install docs/CLI say `@thearchitectit/pi-guardrails`; verify published command resolves actual package | [pkg02-align-pi-package-name](../changes/pkg02-align-pi-package-name/proposal.md) |
| `LEGACY-01` | Open | Documentation/spec owners | numbered design claims contradict later status/code/test evidence; reconcile dates and owner acceptance | [legacy01-stale-numbered-design-claims](../changes/legacy01-stale-numbered-design-claims/proposal.md) |
| `INV-02` | Open | Documentation/platform maintainers | platform-current-state says audit persistence unattached despite `database.NewAuditStore(db)` wiring; reconcile with current source | [reconcile-platform-current-state](../changes/reconcile-platform-current-state/proposal.md) |
| `RUN-01` | Open | Runtime/sandbox owners | sandbox evaluator tests plan helpers/flags, not supported-runtime workload; use benign permitted/denied probes and runtime identity evidence | [verify-sandbox-runtime-conformance](../changes/verify-sandbox-runtime-conformance/proposal.md) |
| `RUN-02` | Open | Classifier/conformance owners | marker positives/reference classifier, repetitive benign/injection corpora; build representative versioned corpus and budgets | [verify-semantic-conformance-evidence](../changes/verify-semantic-conformance-evidence/proposal.md) |
| `RUN-03` | Open | MCP/client integration owners | scanner is explicit caller API; no action interception or automatic ingress instrumentation evidenced; map/adopt enforcement call sites | [trace-action-and-ingress-boundaries](../changes/trace-action-and-ingress-boundaries/proposal.md) |
| `RUN-04` | Open | Evidence owners | envelope lacks classifier/category/isolation context; fixed policy digest; signing optional; bind actual decision context and verify | [bind-decision-evidence-context](../changes/bind-decision-evidence-context/proposal.md) |
| `RUN-05` | Open | Semantic filter owners | Go filter has no `ActionRedact`, span removal, or digest evidence; add redaction behavior and privacy-preserving evidence | [implement-semantic-redaction-evidence](../changes/implement-semantic-redaction-evidence/proposal.md) |
| `RUN-06` | Open | Release/conformance owners | no version-bound report or CI promotion gate; create artifact tied to versions and enforce release decision | [release-gate-conformance-reports](../changes/release-gate-conformance-reports/proposal.md) |
| `DEP-01` | Open | Deployment owners | K8s placeholders and loopback defaults conflict with Pod reachability; render supported private profile and verify probes/service reachability | [dep01-render-kubernetes-profile](../changes/dep01-render-kubernetes-profile/proposal.md) |
| `DEP-02` | Open | Deployment docs owners | legacy guide predates profile/preflight/secret/migration controls; align procedure and verify commands against selected deployment | [dep02-reconcile-production-guide](../changes/dep02-reconcile-production-guide/proposal.md) |
| `OPS-01` | Open | Platform operations | no approved consumer/topology/boundaries/owners; record generic rollout scope, accountable role and approval | [ops01-approve-rollout-inventory](../changes/ops01-approve-rollout-inventory/proposal.md) |
| `OPS-02` | Open | Consumer integration owners | local producer/verifier only; no real external consumer acceptance or Gate 4 applicability evidence | [ops02-verify-external-consumer](../changes/ops02-verify-external-consumer/proposal.md) |
| `OPS-03` | Open | Platform auth/operations | no multi-instance credential revocation propagation/failover record; verify selected replica topology | [ops03-verify-multi-instance-revocation](../changes/ops03-verify-multi-instance-revocation/proposal.md) |
| `OPS-04` | Open | Platform operations | no selected topology migration/readiness/restore/rollback rehearsal; record outcomes against approved topology | [ops04-rehearse-operational-recovery](../changes/ops04-rehearse-operational-recovery/proposal.md) |
| `TPL-32` | Verified and archived | QA/CI maintainers | Committed and local patch scans now use the correct ranges, with passing focused controls. | [Archived TPL-32 evidence](../changes/archive/2026-11-25-regression-tpl32-committed-diff-coverage/evidence.md) |
| `TPL-33` | Verified and archived | MCP build maintainers | Synthetic Docker context checks and the exact no-cache production Dockerfile build passed with a unique image tag and no fixture values displayed. | [docker-tpl33-bound-build-context](../changes/archive/2026-11-25-docker-tpl33-bound-build-context/evidence.md) |
| `TPL-34` | Open | MCP/CI maintainers | The Go advisory scanner has a manual Make target. Active CI lacks a recurring source-level advisory check. | [go-tpl34-automate-source-advisory-checks](../changes/go-tpl34-automate-source-advisory-checks/proposal.md) |
| `TPL-35` | Open | Release maintainers | Release Coverage rejects main `943bcf6` against the older `v3.8.11` tag. The gate applies the existing tag-at-HEAD contract. | [release-tpl35-publish-approved-main](../changes/release-tpl35-publish-approved-main/proposal.md) |
| `TPL-36` | Open (import proposal) | Assembly and MCP evidence owners | The import review found two separate composition contracts. Bind the approved assembly pins to the exact build and policy identity reported by Spec 08. | [bind-assembly-to-runtime-evidence](../changes/bind-assembly-to-runtime-evidence/proposal.md) |
| `TPL-37` | Open (import proposal) | Template and AIGGP CI owners | The import branch proposes a second divergent DevGate pin. Verify a single selected AIGGP revision and the consumer checkout path before changing CI ownership. | [reconcile-devgate-pin-ownership](../changes/reconcile-devgate-pin-ownership/proposal.md) |
| `TPL-38` | Partially verified | Secret scanning and CI maintainers | The Secret Validation job at `943bcf6` failed before scan execution because the installer found a shared Gitleaks temporary path already occupied. | [isolate-gitleaks-runner-install](../changes/isolate-gitleaks-runner-install/proposal.md) |
| `TPL-39` | Verified and archived | Deployment maintainers | Isolated startup, migrations, teardown and restart passed. The exact published launcher upgraded and reran on the operator laptop with all services healthy, both loopback ports, HTTP 200 and original volumes preserved. | [Archived TPL-39 evidence](../changes/archive/2026-11-25-repair-laptop-docker-compose/evidence.md) |
| `TPL-40` | Verified and archived | QA/CI maintainers | Registered the root Python suite exposed by the hosted discovery gate. Local collection meets the 11-case floor. | [Archived TPL-40 evidence](../changes/archive/2026-11-25-register-root-python-test-suite/evidence.md) |
| `TPL-41` | Implemented locally; hosted rerun pending | CI/QA maintainers | Regression Guard wrote multiline PR subjects as a single-line `GITHUB_OUTPUT` value; the step failed on multiple subjects. The unused output is removed and the numeric count has a focused regression. | [fix-regression-guard-step-output](../changes/fix-regression-guard-step-output/proposal.md) |
| `TPL-42` | Verified and archived | MCP configuration maintainers | PostgreSQL URL userinfo now preserves reserved characters. Configuration tests and isolated Windows deployment reached readiness using synthetic credentials. | [Archived TPL-42 evidence](../changes/archive/2026-11-25-encode-postgres-connection-credentials/evidence.md) |

## Pilot sequencing

Use OPS-01 to establish the deployment boundary and accountable roles. Apply
TPL-36 and TPL-37 when selecting the assembly or adopting a new DevGate pin.
DEP-01 and DEP-02 cover deployment procedures for the selected topology.
RUN-01 through RUN-06 cover runtime integration and the evidence that supports
closure claims. OPS-02 through OPS-04 capture adoption, revocation, and
recovery results from that topology. TPL-32, TPL-34, TPL-35, and TPL-38 cover the
checks and release identity that need to be observable for the chosen release.
This sequencing keeps an owner-approved optional deployment decision separate
from implementation and evidence.
