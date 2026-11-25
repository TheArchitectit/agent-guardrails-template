# Template Open, Closed, and Coded Review

**Review snapshot:** `943bcf65055128e338b696dc40d74e79ac33449e`
**Review date:** 2026-11-25
**Purpose:** reconcile current OpenSpec inventory and historical closure claims with source, test, and workflow evidence.

## Summary

At this snapshot, `openspec/specs/` contains 15 specs and 44 requirement rows. All 15 files declare `CLOSED`, while `openspec/INDEX.md` lists only eight. The seven omitted short-slug specs contain 22 additional, semantically nonidentical requirement rows. The indexed 22 requirement rows claim `EXERCISED`; the companion traceability review maps all 44 rows to code and named tests, with explicit limits where component tests do not establish runtime, consumer, signing, or release behavior.

The pinned baseline has four active OpenSpec changes, all four strict-valid, and four archived AIGGP changes with five checked task boxes each (20/20 historical checkboxes). The checked archive boxes are metadata, not a local test result. Four archive deltas yield seven unique short-slug specs; `openspec/changes/README-aiggp-routed-packages.md` says six and omits `isolation-levels`. Each archived proposal also references the absent `openspec/changes/AIGGP-RETIREMENT-2026-10-02.md`.

Official OpenSpec CLI 2.2.0 strict validation at the pinned baseline accepts the eight indexed specs and four active changes, and rejects the seven unindexed specs for scenario format. Thus no active-change validation failures were observed at this baseline. There is no repository-wide validation gate for all specs and active changes. Current source confirms the principal runtime evidence gaps: static rather than runtime sandbox conformance, small marker-based conformance fixtures, caller-invoked provenance scans, no universal action mediation, incomplete evidence envelope binding, and no Go semantic redaction action with span-digest evidence.

## Inventories and status claims

| Inventory | Count | Evidence and interpretation |
|---|---:|---|
| Spec files | 15 | `openspec/specs/**/spec.md`; all declare CLOSED. |
| Indexed specs | 8 | `openspec/INDEX.md:8-19`; seven files omitted. |
| Requirement rows | 44 | 22 in indexed long slugs and 22 in seven short slugs. |
| Indexed requirement statuses | 22 EXERCISED | Status label is a claim, not proof of full SHALL behavior. See traceability companion. |
| Archived AIGGP packages | 4 | `aiggp-03` through `aiggp-06`; five checked boxes each, 20 total. |
| Unique short-slug archive outputs | 7 | Four archive delta trees share `evidence-and-conformance`; README incorrectly counts six and omits `isolation-levels`. |
| Active changes at baseline | 4 | All four active changes pass strict validation. Review-worktree proposals are separate. |

The archive task counts should not be restated as test counts or independent proof. PR #8 is historical evidence that its closure notes explicitly bounded verification to the older Go test suite. The GitHub snapshot at the earlier c7 review showed zero open PRs and nine closed PRs (#1–#8 and #10), with open issue #9 separate from OpenSpec; these historical GitHub counts are not asserted as current at `943bcf6`.

## Requirement-to-code traceability

The [current OpenSpec requirement trace](2026-11-25-template-requirement-trace.md) covers all 44 published rows plus 27 rows across the four active changes. It records status claims, code mappings, test symbols, and evidence limits. Test symbols were checked against the source tree; the matrix does not claim those tests were run during this review. The prior [c7-pinned traceability companion](2026-11-25-openspec-traceability.md) remains unchanged.

Key findings in that matrix:

- `RUN-01`: `mcp-server/internal/guardrails/sandbox_conformance.go` evaluates configured helper decisions and rendered plans; it does not run a conformance workload in a supported runtime.
- `RUN-02` / `RUN-06`: semantic positives use marker tokens recognized by the reference classifier; benign and injection corpora are small/repetitive; no required version-bound report or release promotion gate is evidenced.
- `RUN-03`: injection verdict APIs and provenance scan tools require caller invocation; no command/edit/git/send action hook or automatic file/web/email/tool ingress instrumentation is evidenced.
- `RUN-04`: the evidence envelope omits classifier/category/runtime fields, hashes a fixed policy identity rather than active config, and permits unsigned output when no signer is configured.
- `RUN-05`: `content_filter_config.go` supports no `ActionRedact`; no Go span removal or span-digest result was found. The Pi output validator is a separate optional feature and does not supply the specified evidence digest.
- `INV-01` / `ARC-01`: index and archive manifest discrepancies remain reconciliation work.

## QA and status deltas since earlier snapshot

QA-05 changed after the earlier c7 baseline. The official [archived QA-05 evidence](../changes/archive/2026-11-25-qa05-source-reachable-go-toolchain-remediation/evidence.md) records Go 1.26.11 remediation, source-reachable findings reduced from 45 to zero, and four required-module-only residual findings. This does not close recurring CI advisory automation, tracked separately as TPL-34. Do not repeat the earlier 45-source-reachable baseline as current.

The current template also has a dated hosted workflow update: [Release Coverage run `45556758733`](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/45556758733) failed because current `HEAD 943bcf6` is not the tagged release `v3.8.11` at `c7c4044`; this matches the configured tag-at-HEAD contract. [Trusted Main Validation `45556758751`](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/45556758751), [Regression Guard `45556758686`](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/45556758686), [Documentation Standards `45556758670`](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/45556758670), and [Guardrails Compliance `45556758692`](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/45556758692) succeeded. [Secret Validation `45556758743`](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/45556758743) failed during scanner installation before scanning; this is neither a leak finding nor a clean scan. [Team Validation `45556758721`](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/45556758721) was queued at the latest verified observation; no result is claimed.

## OpenSpec change crosswalk

The [published gap register](2026-11-25-template-gap-register.md) maps each ID to its native OpenSpec proposal and state. The [current requirement trace](2026-11-25-template-requirement-trace.md) contains the row-level mappings. Existing historical audits remain unchanged. Remediation changes remain proposals until their unchecked implementation tasks are completed and evidence is reviewed.

## Evidence limits

No exploit procedures or malicious payloads were run. Suggested future sandbox evidence can use benign permitted and denied operations. A test or code symbol establishes only its exercised path; it does not establish universal ingress coverage, production action enforcement, real runtime escape resistance, signature policy, or release gating unless that path is directly evidenced.
