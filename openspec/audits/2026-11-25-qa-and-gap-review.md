# QA and OpenSpec gap review

**Review snapshot:** `c7c40443ef06b8a8cacfdc52c34384d6d0eef134`
**Review date:** 2026-11-25
**Scope:** repository QA evidence, OpenSpec inventory and closure claims, and
spec-to-code traceability. This is a read-only assessment of the pinned
snapshot; proposals below do not claim fixes or closure.

## Summary

The pinned revision has successful repository validation evidence, but the
OpenSpec catalog and the strength of several runtime claims need reconciliation.
Seven hosted workflow runs on this exact commit succeeded. Four archived
packages contain 20/20 checked task boxes; those are historical completion
metadata, not a local test count. The pinned revision has zero active OpenSpec
changes. The official OpenSpec CLI 2.2.0
strictly validates the eight indexed specs and rejects seven additional spec
files because their scenarios do not meet the CLI's Given/When/Then structure.
All 15 files declare CLOSED, but declared status alone does not resolve the
catalog gap or establish runtime coverage.

Four archived change packages are present (aiggp-03 through aiggp-06). Their
archive note says the packages were completed and their deltas published; the
review records this as historical traceability, not independent runtime proof.
GitHub review context at the snapshot showed zero open PRs and nine closed PRs
(#1–#8 and #10), plus open issue #9, a resource suggestion outside the
OpenSpec inventory. PR #8 is relevant historical evidence; its notes explicitly
bound closure claims to the old Go tests.

## QA evidence

| Check | Result | Evidence and limit |
|---|---|---|
| Python repository suites | 300/300 passed | 268 tests in `scripts/` plus 32 example tests passed; 68 subtests passed. Coverage was 55.24% (4,240/7,675 lines). Python 3.12 emitted 363 deprecation warnings, primarily from naive `datetime.utcnow()` use. |
| Go builds | 3/3 passed | MCP server, team CLI, and example builds passed on the pinned source. |
| Team CLI/examples tests | Passed | The team CLI and example test targets passed. |
| File-permission QA fixtures | Environment-sensitive | Four direct auth/config/secretfile fixtures request mode 0644 and are created as 0600 under default umask 0077. A downstream mutation baseline also depends on one fixture. All direct mode fixtures pass under scoped umask 0022. This is fixture portability evidence; the private 0600 mode is not a production secret permission defect. |
| actionlint | Passed | Version 1.7.7 accepted the workflow files with the custom private-runner label recognized. |
| Trusted Main Validation | Success | [Run 43432386404](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/43432386404) on the pinned commit. |
| Team Validation | Success | [Run 43432386908](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/43432386908) on the pinned commit. |
| Regression Guard | Success | [Run 43432386449](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/43432386449) on the pinned commit. |
| Documentation Standards | Success | [Run 43432386501](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/43432386501) on the pinned commit. |
| Secret Validation | Success | [Run 43432386593](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/43432386593) on the pinned commit. |
| Guardrails Compliance | Success | [Run 43432386444](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/43432386444) on the pinned commit. |
| Release Coverage | Success | [Run 43432386765](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/43432386765) on the pinned commit. |
| OpenSpec CLI strict validation | 8 pass, 7 fail | `@fission-ai/openspec` 2.2.0, strict mode. The seven unindexed files fail scenario-format warnings promoted by `--strict`; see `QA-01`. |
| Pi extension unit suite | 196 pass, 1 fail | `npm ci` succeeded in a temporary copy. One MCP bridge test cannot resolve `@modelcontextprotocol/sdk`, which is not declared in the package manifest. A temporary writable `HOME` resolved five session-store environment failures. |
| Pi extension with documented SDK supplied | 197 pass | Adding the documented SDK version in a temporary copy allowed all tests to pass; it appeared as extraneous because the package manifest does not declare it. |
| npm dependency advisory snapshot | Pi: 9 dev package entries (3 moderate, 4 high, 2 critical); VS Code: 14 dev package entries (4 moderate, 10 high) | Results from unchanged lockfiles against advisory data checked 2026-11-25. `npm audit --omit=dev` reported zero for both manifests. No CI advisory enforcement was found. Counts are package entries, not unique vulnerabilities or production reachability claims. |
| VS Code extension compile | Passed | Locked `npm ci` and `npm run compile` in a temporary copy. |
| MCP web static checks | Passed | `node --check` over `mcp-server/web/js/**/*.js`; Python stdlib HTMLParser parsed both static HTML entry points. No MCP web package manifest or lockfile exists. |

The hosted workflow results show those jobs passed for the pinned revision.
They do not establish that every conformance harness is invoked as a release
gate or that its fixtures exercise a real runtime.

## OpenSpec inventory and closure

The index at the pinned revision lists eight specs, all marked CLOSED. A
recursive inventory finds 15 `spec.md` files: eight indexed verbose slugs and
seven unindexed short slugs. The files declare CLOSED, while seven short-slug
files fail strict OpenSpec scenario-format validation. The table records the
declared status and index membership; it does not certify the underlying
closure claims.

| Spec path | Indexed | Declared status at snapshot |
|---|---:|---|
| `openspec/specs/indirect-injection-provenance-records/spec.md` | Yes | CLOSED |
| `openspec/specs/prompt-injection-classification-and-mediation/spec.md` | Yes | CLOSED |
| `openspec/specs/prompt-injection-evidence-and-conformance/spec.md` | Yes | CLOSED |
| `openspec/specs/provenance-policy-and-conformance/spec.md` | Yes | CLOSED |
| `openspec/specs/sandbox-conformance-and-evidence/spec.md` | Yes | CLOSED |
| `openspec/specs/sandbox-isolation-levels/spec.md` | Yes | CLOSED |
| `openspec/specs/semantic-filtering-categories-and-actions/spec.md` | Yes | CLOSED |
| `openspec/specs/semantic-filtering-evidence-and-conformance/spec.md` | Yes | CLOSED |
| `openspec/specs/categories-and-actions/spec.md` | No | CLOSED; strict validation failed |
| `openspec/specs/classification-and-mediation/spec.md` | No | CLOSED; strict validation failed |
| `openspec/specs/conformance-and-evidence/spec.md` | No | CLOSED; strict validation failed |
| `openspec/specs/evidence-and-conformance/spec.md` | No | CLOSED; strict validation failed |
| `openspec/specs/isolation-levels/spec.md` | No | CLOSED; strict validation failed |
| `openspec/specs/policy-and-conformance/spec.md` | No | CLOSED; strict validation failed |
| `openspec/specs/provenance-records/spec.md` | No | CLOSED; strict validation failed |

Short and verbose slugs overlap in subject but are not interchangeable copies.
For example, `evidence-and-conformance` contains six requirements while
`prompt-injection-evidence-and-conformance` contains three. Four archived
changes are recorded under `openspec/changes/archive/2026-11-06-aiggp-03-*`
through `aiggp-06-*`; their checked task boxes total 20/20. Their deltas yield
seven unique short-slug specs because `evidence-and-conformance` is shared.
The routing README says six and omits `isolation-levels`. Package headers also
reference the absent `openspec/changes/AIGGP-RETIREMENT-2026-10-02.md`. These
manifest issues are tracked as `ARC-01`. The archive records do not settle
which short or verbose variant is canonical.

| Archived change | Delta paths under `openspec/changes/archive/2026-11-06-*` |
|---|---|
| `aiggp-03-prompt-injection-defense` | `specs/classification-and-mediation`; shared `specs/evidence-and-conformance` |
| `aiggp-04-semantic-content-filtering` | `specs/categories-and-actions`; shared `specs/evidence-and-conformance` |
| `aiggp-05-runtime-sandbox-isolation` | `specs/conformance-and-evidence`; `specs/isolation-levels` |
| `aiggp-06-indirect-injection-provenance` | `specs/policy-and-conformance`; `specs/provenance-records` |

## Gap register

These IDs are unique within this review and link to the tracked proposals
below. They describe evidence or process gaps, not confirmed production
vulnerabilities.

| ID | Finding | Tracked proposal |
|---|---|---|
| `INV-01` | Index omits seven of 15 spec files; overlapping short and verbose slugs have distinct requirements, and archive-to-spec mapping needs reconciliation. | [`reconcile-openspec-inventory`](../changes/reconcile-openspec-inventory/proposal.md) |
| `ARC-01` | Archive README reports six unique published specs and omits `isolation-levels`; package headers reference a retirement file absent at this snapshot. | [`reconcile-openspec-inventory`](../changes/reconcile-openspec-inventory/proposal.md) |
| `QA-01` | Strict OpenSpec validation passes 8 indexed specs and fails 7 unindexed files on scenario structure; repository-wide inventory/validation is not demonstrated by the index. | [`strengthen-repository-qa`](../changes/strengthen-repository-qa/proposal.md) |
| `QA-02` | Four direct auth/config/secretfile mode fixtures request 0644 and become 0600 under 0077; all direct mode fixtures pass under scoped 0022. A downstream mutation baseline also depends on one fixture. | [`strengthen-repository-qa`](../changes/strengthen-repository-qa/proposal.md) |
| `QA-03` | Python 3.12 test run emits 363 deprecation warnings, primarily for naive `datetime.utcnow()` usage. | [`strengthen-repository-qa`](../changes/strengthen-repository-qa/proposal.md) |
| `QA-04` | Current npm advisories identify affected development dependency entries in Pi and VS Code lockfiles; no CI advisory enforcement is evidenced. Runtime dependency audit reported zero for both. | [`strengthen-repository-qa`](../changes/strengthen-repository-qa/proposal.md) |
| `CI-01` | GitLab jobs invoke `pip install -r requirements.txt`, but the referenced root manifest is absent. | [`strengthen-repository-qa`](../changes/strengthen-repository-qa/proposal.md) |
| `CI-02` | GitLab and Jenkins set Go 1.21 while `mcp-server/go.mod` declares 1.25.5; automatic toolchain acquisition may affect reproducibility, but mismatch alone does not prove failure. | [`strengthen-repository-qa`](../changes/strengthen-repository-qa/proposal.md) |
| `DIST-01` | Team CLI installation places only the Go binary; its Python backend path is checkout-relative unless explicitly configured. | [`strengthen-repository-qa`](../changes/strengthen-repository-qa/proposal.md) |
| `IDE-01` | VS Code extension tests its initial connection before initializing client configuration, so initial connection testing enters its error path. | [`strengthen-repository-qa`](../changes/strengthen-repository-qa/proposal.md) |
| `IDE-02` | IDE docs/config default REST URL to port 8095 while the REST server defaults to 8081; MCP protocol uses separate port 8080. | [`strengthen-repository-qa`](../changes/strengthen-repository-qa/proposal.md) |
| `IDE-03` | VS Code, Neovim, and JetBrains send `code`; the REST selection-validation handler expects `selection`, so requests fail schema validation. | [`strengthen-repository-qa`](../changes/strengthen-repository-qa/proposal.md) |
| `INSTALL-01` | `scripts/setup_agents.py` strips leading dots from a documented `.claude/...` destination, changing the selected install path. | [`strengthen-repository-qa`](../changes/strengthen-repository-qa/proposal.md) |
| `INSTALL-02` | `install_skill` treats `.windsurfrules` as a destination directory and appends the filename, rather than writing the platform rules file. | [`strengthen-repository-qa`](../changes/strengthen-repository-qa/proposal.md) |
| `PKG-01` | Pi extension's MCP bridge test fails because SDK used by the test is absent from its manifest; locked install run is 196/197, while adding documented SDK version in a temp copy passes 197/197. | [`strengthen-repository-qa`](../changes/strengthen-repository-qa/proposal.md) |
| `PKG-02` | Pi extension package metadata uses `@architectit/pi-guardrails` while install instructions and CLI help name `@thearchitectit/pi-guardrails`. | [`strengthen-repository-qa`](../changes/strengthen-repository-qa/proposal.md) |
| `LEGACY-01` | Several numbered design/status docs retain dated problem statements that current source and named tests supersede or conflict with; reconcile them with owner acceptance, retaining dates and open residuals. | [`strengthen-repository-qa`](../changes/strengthen-repository-qa/proposal.md) |
| `DEP-01` | Kubernetes manifest has placeholders and no render/apply procedure; it omits non-loopback binds/profile while Service and probes target the Pod IP. | [`verify-platform-rollout-readiness`](../changes/verify-platform-rollout-readiness/proposal.md) |
| `DEP-02` | Production deployment guide predates the current profile, preflight, secret, staging, and migration controls. | [`verify-platform-rollout-readiness`](../changes/verify-platform-rollout-readiness/proposal.md) |
| `OPS-01` | No enterprise consumer/topology inventory, supported profile decision, or named operating owners are recorded for a rollout. | [`verify-platform-rollout-readiness`](../changes/verify-platform-rollout-readiness/proposal.md) |
| `OPS-02` | Local producer/verifier mechanics do not evidence acceptance by a real external consumer; applicable external Gate 4 remains a rollout question. | [`verify-platform-rollout-readiness`](../changes/verify-platform-rollout-readiness/proposal.md) |
| `OPS-03` | Revocation propagation evidence covers one process; multi-instance behavior for a selected replica topology is unproven. | [`verify-platform-rollout-readiness`](../changes/verify-platform-rollout-readiness/proposal.md) |
| `OPS-04` | Repository has migration/readiness/recovery design evidence but no rehearsal record for a selected deployment topology, restore, and service objectives. | [`verify-platform-rollout-readiness`](../changes/verify-platform-rollout-readiness/proposal.md) |
| `INV-02` | `docs/platform-current-state.md` says audit persistence is unattached, while current server code attaches `database.NewAuditStore(db)`. | [`verify-platform-rollout-readiness`](../changes/verify-platform-rollout-readiness/proposal.md) |
| `RUN-01` | Sandbox escape conformance evaluates policy and configuration helpers but does not exercise a workload against a supported runtime; closure evidence does not demonstrate runtime containment. | [`verify-runtime-guardrail-conformance`](../changes/verify-runtime-guardrail-conformance/proposal.md) |
| `RUN-02` | Semantic conformance uses marker-based reference fixtures; evidence does not establish representative category detection and false-positive performance for configured production classifiers. | [`verify-runtime-guardrail-conformance`](../changes/verify-runtime-guardrail-conformance/proposal.md) |
| `RUN-03` | Inspected injection and provenance paths are explicit tools; this tree does not demonstrate automatic interception at every claimed action or ingestion boundary. | [`verify-runtime-guardrail-conformance`](../changes/verify-runtime-guardrail-conformance/proposal.md) |
| `RUN-04` | Injection evidence fields and optional signing do not substantiate all classifier, category, active-policy, and verification claims stated by the spec. | [`verify-runtime-guardrail-conformance`](../changes/verify-runtime-guardrail-conformance/proposal.md) |
| `RUN-05` | The Go semantic action enum does not implement redaction/span removal despite the spec claim; Pi output filtering is a separate warning/log path. | [`verify-runtime-guardrail-conformance`](../changes/verify-runtime-guardrail-conformance/proposal.md) |
| `RUN-06` | Versioned conformance harnesses have no evidenced required release gate or version-bound report artifact. | [`verify-runtime-guardrail-conformance`](../changes/verify-runtime-guardrail-conformance/proposal.md) |

The complete 44-row requirement, code, and test mapping is in the [OpenSpec traceability companion](2026-11-25-openspec-traceability.md).

## Traceability and interpretation

- The eight indexed specs state CLOSED with closure tables; those tables cite
  implementation and test paths. Citation of a test file does not by itself
  show the test was run in a required release gate.
- The archived aiggp packages preserve proposals, task lists, designs, and
  deltas. Their checked tasks are historical completion claims. The archive
  README miscounts the seven unique outputs and omits a slug; reconciliation is
  tracked as `INV-01` and `ARC-01`.
- Historical PR #8: [pull/8](https://github.com/TheArchitectit/agent-guardrails-template/pull/8).
  Its notes bound the relevant closure to old Go tests; treat it as historical
  context, not as evidence for additional runtime behavior.
- Open issue [#9](https://github.com/TheArchitectit/agent-guardrails-template/issues/9)
  is a resource suggestion and was not counted as an OpenSpec change.
- The separate historical `docs/specs/guardrail-gaps-2026/STATUS.md` ledger
  dates its 261-row recount to 2026-10-10; it is not a current recount of these
  OpenSpecs. Its `tools-reference.md` link resolves locally and is not a finding.
- The 261-row legacy ledger includes historical residuals; they are not
  relabeled as confirmed current failures here. The numbered design inventory
  separately contains status drift tracked as `LEGACY-01`.
- The separate numbered design inventory is not the OpenSpec tree. Examples of
  current documentation drift include Spec 12's stale open-routes statement,
  Spec 16's 2026-10-04 baseline on authorization/config secrecy contradicted by
  current code and named tests, and Spec 23's opening problem statement
  contradicted by its later test-closure section. Specs 07 and 17 also need
  their earlier acceptance/headline claims bounded by later implementation
  evidence and remaining requirements (`LEGACY-01`).
- No conclusion here assumes behavior outside the repository. Runtime or
  consumer integration may exist elsewhere but was not evidenced in this
  snapshot.

## Branch and change inventory

At this dated snapshot, the two live upstream refs observed were `main` at
`c7c4044` and `chore/external-audit` at `d7a0264`. The audit branch is
divergent (25 commits only on main and 27 only on the audit branch) and adds
26 QA digest Markdown files without OpenSpec or implementation changes. It is
not evidence for merging its content into this review. Historical PR #8
(`feature/guardrail-gaps-2026`) and PR #10
(`docs/submodule-and-consumer-wiring`) are merged; they provide provenance,
not open work. Tags and historical/fork refs are outside this live branch
inventory.

## Platform rollout readiness

The passing hosted workflows and Python suite, successful IDE compile, and
implemented authorization, signed-evidence, migration, readiness, and CI
controls support further controlled evaluation. They do not establish an
  approved consumer deployment, organization-wide operational readiness, or
  compliance certification. The repository has no rollout inventory or
  consumer acceptance evidence. Before a broad rollout, the tracked evidence gaps need
resolution in order:

1. Reconcile the full 15-spec inventory, short/verbose authority, and archive
   manifest (`INV-01`, `ARC-01`, `QA-01`).
2. Close reproducibility, dependency review, and integration issues in CI, installed CLI, IDE
   clients, package setup, and deployment (`CI-01`, `CI-02`, `DIST-01`,
   `QA-04`, `IDE-01` through `IDE-03`, `INSTALL-01`, `INSTALL-02`, `PKG-01`, `PKG-02`,
   `DEPLOY-01`, `LEGACY-01`).
3. Produce version-bound runtime, classifier, action-boundary, evidence, and
   release-gate verification (`RUN-01` through `RUN-06`).
4. Select and document the enterprise deployment topology and operating
   owners, then validate rendered deployment, consumer acceptance, multi-
   instance revocation, and operational recovery (`DEP-01`, `DEP-02`,
   `OPS-01` through `OPS-04`, `INV-02`).

The audit records evidence needed for a pilot and later rollout decision. It
does not issue compliance certification or a production approval.

## Scope

This review records QA and evidence gaps and proposes acceptance criteria. It
does not alter production code, change any spec's closure status, assert that
the proposals are complete, or reproduce attack procedures.
