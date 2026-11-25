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
| `INV-01` | Verified and archived | OpenSpec maintainers | The index matches all canonical paths and uses the status evidence in each specification. The 15-spec historical snapshot stays distinct. | [Archived INV-01 evidence](../changes/archive/2026-11-25-inv01-canonical-openspec-inventory/evidence.md) |
| `ARC-01` | Verified and archived | OpenSpec maintainers | Eight historical deltas map to seven published paths. The manifest includes isolation levels and records the unavailable retirement reference. | [Archived ARC-01 evidence](../changes/archive/2026-11-25-arc01-archive-reference-provenance/evidence.md) |
| `QA-01` | Verified and archived | QA/OpenSpec maintainers | Structured 25 scenarios across the seven failing specs. Strict validation passes for the complete tree. | [Archived QA-01 evidence](../changes/archive/2026-11-25-qa01-openspec-strict-scenario-structure/evidence.md) |
| `QA-02` | Verified and archived | Go QA maintainers | Four direct permission fixtures set and verify mode; auth/config/secretfile/mutationkill passed fresh under Go 1.26.11 Linux with umasks 0022 and 0077 | [make-permission-tests-umask-independent](../changes/archive/2026-11-25-make-permission-tests-umask-independent/evidence.md) |
| `QA-03` | Verified locally and archived | Python QA maintainers | Python utilities compare UTC-aware timestamps and preserve JSON fields and terminal `Z`. All 276 script tests passed with deprecations promoted to errors. | [Archived QA-03 evidence](../changes/archive/2026-11-25-use-timezone-aware-python-utc/evidence.md) |
| `QA-04` | Open | JS package/QA maintainers | npm audit snapshot: Pi 9 dev entries (3 moderate/4 high/2 critical); VS Code 14 dev (4 moderate/10 high); production-only zero | [audit-javascript-dev-dependencies](../changes/audit-javascript-dev-dependencies/proposal.md) |
| `QA-05` | Verified and archived | Go platform/QA maintainers | Merged in #12 at `943bcf6`. Source paths fell from 45 to 0 on the same advisory database. Four module-only findings remain. The official archive publishes the evidence contract. | [qa05-source-reachable-go-toolchain-remediation](../changes/archive/2026-11-25-qa05-source-reachable-go-toolchain-remediation/proposal.md) |
| `CI-01` | Verified locally and archived | CI/Python maintainers | The tracked root dependency manifest is installed by the GitLab Python jobs and Jenkins Python tests. Clean local replay passed 268 tests and produced coverage and export artifacts. | [Archived CI-01 evidence](../changes/archive/2026-11-25-fix-gitlab-requirements-inputs/evidence.md) |
| `CI-02` | Open | CI/Go maintainers | GitLab MCP tests and Jenkins MCP tests and Team CLI build still rely on a Go 1.21 declaration. The MCP module requires 1.26.11 and the Team CLI requires 1.23.2. | [align-legacy-ci-go-toolchains](../changes/align-legacy-ci-go-toolchains/proposal.md) |
| `DIST-01` | Open | CLI maintainers | install copies Go binary only; default Python backend is checkout/cwd-relative; verify outside-checkout install and override | [package-team-cli-python-backend](../changes/package-team-cli-python-backend/proposal.md) |
| `IDE-01` | Verified and archived | IDE extension maintainers | Initial activation loads the configured URL and SecretStorage before the first request. The regression covers successful readiness, refresh, and rejected requests. | [Archived activation verification](../changes/archive/2026-11-25-ide01-initialize-vscode-client/tasks.md) |
| `IDE-02` | Open | IDE/server maintainers | IDE URLs default 8095, REST server defaults 8081; verify defaults target REST listener | [ide02-align-ide-rest-endpoint](../changes/ide02-align-ide-rest-endpoint/proposal.md) |
| `IDE-03` | Open | IDE/server maintainers | server binds `selection`, clients send `code`; verify request succeeds through each supported client | [ide03-align-selection-payload](../changes/ide03-align-selection-payload/proposal.md) |
| `IDE-04` | Verified and archived | IDE extension maintainers | The package’s normal test entry point now compiles, runs TypeScript lint, and runs the activation regression. Parser and rule negative controls fail as expected. | [Archived lint verification](../changes/archive/2026-11-25-restore-vscode-extension-lint-config/evidence.md) |
| `INSTALL-01` | Verified locally and archived | Installer maintainers | The installer preserves `.claude`, refuses path and symlink escapes before writing, and handles repeated symlink installs. | [Archived INSTALL-01 verification](../changes/archive/2026-11-25-preserve-hidden-agent-install-paths/tasks.md) |
| `INSTALL-02` | Verified locally and archived | Installer maintainers | A named Windsurf install writes the root `.windsurfrules` file. Focused tests also exercised bulk Windsurf and directory-based Claude installs. | [Archived INSTALL-02 verification](../changes/archive/2026-11-25-fix-windsurf-single-skill-target/tasks.md) |
| `PKG-01` | Verified and archived | Pi package maintainers | SDK 1.32.2 is locked for development. The clean package suite passed all 199 tests and the packed production-only install retained standalone loading. | [Archived PKG-01 evidence](../changes/archive/2026-11-25-pkg01-declare-pi-mcp-sdk-dev-dependency/evidence.md) |
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
| `TPL-38` | Verified locally and archived | Secret scanning and CI maintainers | Checksum-verified Gitleaks 8.24.3 uses private scratch and scans without a hosted Action license. The local 12-commit draft scan returned zero findings and the synthetic finding control exited 2. | [Archived TPL-38 evidence](../changes/archive/2026-11-25-isolate-gitleaks-runner-install/evidence.md) |
| `TPL-39` | Verified and archived | Deployment maintainers | Isolated startup, migrations, teardown and restart passed. The exact published launcher upgraded and reran on the operator laptop with all services healthy, both loopback ports, HTTP 200 and original volumes preserved. | [Archived TPL-39 evidence](../changes/archive/2026-11-25-repair-laptop-docker-compose/evidence.md) |
| `TPL-40` | Verified and archived | QA/CI maintainers | Registered the root Python suite exposed by the hosted discovery gate. Local collection meets the 11-case floor. | [Archived TPL-40 evidence](../changes/archive/2026-11-25-register-root-python-test-suite/evidence.md) |
| `TPL-41` | Verified locally and archived | CI/QA maintainers | Regression Guard writes only its numeric bug-fix count. The laptop replay passed the exact output and enforcement steps, and Team Validation ran every enumerated Python suite. | [Archived TPL-41 evidence](../changes/archive/2026-11-25-fix-regression-guard-step-output/evidence.md) |
| `TPL-42` | Verified and archived | MCP configuration maintainers | PostgreSQL URL userinfo now preserves reserved characters. Configuration tests and isolated Windows deployment reached readiness using synthetic credentials. | [Archived TPL-42 evidence](../changes/archive/2026-11-25-encode-postgres-connection-credentials/evidence.md) |
| `TPL-43` | Open | Deployment/pilot QA maintainers | The published launcher contract needs separate-host cold Windows evidence that Docker Desktop starts from stopped/unavailable state, then first-run credentials and readiness persist across an isolated restart; do not touch the original pilot. | [tpl43-windows-cold-start-acceptance](../changes/tpl43-windows-cold-start-acceptance/proposal.md) |
| `TPL-44` | Open | Pi consumer/security maintainers | Pi handler-level Four Laws, classifier, permission and output-validation behavior needs consumer-path evidence; MCP tests do not establish Pi event enforcement or actual redaction. | [tpl44-pi-consumer-runtime-evidence](../changes/tpl44-pi-consumer-runtime-evidence/proposal.md) |
| `TPL-45` | Open | Policy-pack/runtime maintainers | Resolver and signature tests exist, but integrated runtime evidence must bind out-of-band trust, locked pack bytes, activation digest and an actual enforced rule; signature validity is not approval. | [tpl45-policy-pack-activation-evidence](../changes/tpl45-policy-pack-activation-evidence/proposal.md) |
| `TPL-46` | Open | Multi-agent/MCP maintainers | Multi-agent chain and conflict code/tests need runtime evidence through registered MCP requests; explicitly exclude unimplemented agent-registry YAML and LLM review validator claims. | [tpl46-multi-agent-enforcement-evidence](../changes/tpl46-multi-agent-enforcement-evidence/proposal.md) |
| `TPL-47` | Open | Compliance/runtime evidence owners | Measured compliance probe/report code needs authorized live observations tied to report rows and honest unknowns; do not confuse declarations/simulated collection with evidence or semantic release-conformance reports. | [tpl47-compliance-live-probe-evidence](../changes/tpl47-compliance-live-probe-evidence/proposal.md) |
| `TPL-48` | Verified locally and archived | CI/QA maintainers | Commit attribution reads each complete event-scoped Git message and fails on unreadable records. All six Linux Git fixtures passed, including large messages. | [Archived TPL-48 evidence](../changes/archive/2026-11-25-fix-commit-attribution-record-parsing/evidence.md) |
| `TPL-49` | Verified locally and archived | QA/OpenSpec maintainers | Seven section-sign glyphs in three proposals failed the pinned DevGate charset gate. The corrected citations preserve their section ranges. The clean candidate and all 13 charset controls passed. | [Archived TPL-49 evidence](../changes/archive/2026-11-25-normalize-audit-proposal-text/evidence.md) |
| `TPL-50` | Verified locally and archived | CI maintainers | All 45 visible GitHub jobs and the dependent summary passed their applicable laptop checks. Hosted runner assignment and reporting integrations are tracked separately. | [Archived TPL-50 evidence](../changes/archive/2026-11-25-verify-draft-workflows-locally/evidence.md) |
| `TPL-51` | Verified and archived | MCP validation maintainers | The local legacy Jenkins command exposed simultaneous regex cache writes. The native regression failed on the old cache and passed repeatedly after immutable publication and in-place clearing. | [Archived concurrency evidence](../changes/archive/2026-11-25-repair-validation-regex-cache-concurrency/evidence.md) |

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
