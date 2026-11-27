# Active plans: release and toolchain integrity

Three active OpenSpec changes govern version selection, conformance-report
retention, and release metadata. The source proposals keep real publication
operations separate from local tests; this plan preserves that boundary.

The lead agent owns design choices and final signoff. Team assignments below
are implementation and independent-validation lanes, not extra approval
permissions. Release-owner decisions remain separate only where the source
proposal explicitly requires them.

<a id="reconcile-devgate-pin-ownership"></a>
### 1. `reconcile-devgate-pin-ownership`

- **OpenSpec:** [proposal](../../../openspec/changes/reconcile-devgate-pin-ownership/proposal.md) · [tasks](../../../openspec/changes/reconcile-devgate-pin-ownership/tasks.md) · [pin ownership delta](../../../openspec/changes/reconcile-devgate-pin-ownership/specs/devgate-pin-ownership/spec.md)
- **Tasks:** all four `[TPL-37]` tasks are unchecked: state the ordinary template root gitlink pin; keep the AIGGP import and proposed engine pins branch-scoped; validate affected scripts/tests and an authorized clean recursive or standalone fetch before any ownership/pin change; preserve DG-08 as the reusable-bootstrap item.
- **Dependencies:** source declares current template `.devgate` pin `8ccb4467c70614c37f4a533428d2aa1e09a80517`, import commit `abc8c730e066a3377d4cac83d6605d249c6bf5b2`, and branch-only proposed engine pin `17b3369908a2e1f4b7356fde0486d179c81eedcd`. A real pin change requires authorized access to the source and a successful clean fetch; absent access means record unavailable.
- **Owner / independent check:** Team 5 Platform/CI owner owns template CI pin behavior; Team 8 Integration Engineer owns consumer bootstrap pin propagation; Team 2 architecture resolves pin ownership semantics; Team 10 QA validates scripts and clean fetch evidence.
- **Acceptance / evidence:** ordinary template source uses the declared compatible root pin; consumer builds select one immutable pin consistently through assembly/bootstrap; imported engine pin remains scoped to its branch; no private topology is recorded.
- **Model / effort:** Luna for ownership matrix/docs and pin assertions; Sol for bootstrap-script changes; lead agent decides if a source change is warranted. **4–8 person-hours**, excluding private repository access approval.

<a id="release-gate-conformance-reports"></a>
### 2. `release-gate-conformance-reports`

- **OpenSpec:** [proposal](../../../openspec/changes/release-gate-conformance-reports/proposal.md) · [tasks](../../../openspec/changes/release-gate-conformance-reports/tasks.md) · [prompt injection delta](../../../openspec/changes/release-gate-conformance-reports/specs/prompt-injection-evidence-and-conformance/spec.md) · [semantic evidence delta](../../../openspec/changes/release-gate-conformance-reports/specs/semantic-filtering-evidence-and-conformance/spec.md)
- **Tasks:** all three `[RUN-06]` tasks are unchecked: produce immutable content-addressed conformance reports bound to source/corpus/classifier/policy/release/runtime identities; add a required freshness/threshold/artifact-completeness gate; fail verified promotion for absent, stale, altered, unavailable, incomplete, or failing evidence and retain misses immutably.
- **Dependencies:** reliable prompt-injection and semantic conformance corpus results (RUN-02); bound policy/classifier context (RUN-04); version-bump invalidation/freshness rules; protected release workflow identity and retention store.
- **Owner / independent check:** Team 5 CI/CD Architect owns the required job and artifact transport; Team 7 classification/evidence owners supply report schema; Team 9 AppSec reviews tamper/freshness failure modes; Team 10 QA tests fail-closed cases; Team 12 Release Manager validates promotion integration.
- **Acceptance / evidence:** report digest can be retrieved and verified without overwrite; every report binds the declared inputs; stale/missing/tampered/unavailable/incomplete/failing reports block promotion; release job retains named misses.
- **Model / effort:** Sol for artifact identity and workflow logic; Luna for report formats and negative fixtures; Astra may assist with a design critique, but lead agent owns the contract/signoff. **12–20 person-hours**, excluding artifact-store policy changes.

<a id="release-tpl35-publish-approved-main"></a>
### 3. `release-tpl35-publish-approved-main`

- **OpenSpec:** [proposal](../../../openspec/changes/release-tpl35-publish-approved-main/proposal.md) · [tasks](../../../openspec/changes/release-tpl35-publish-approved-main/tasks.md) · [release metadata delta](../../../openspec/changes/release-tpl35-publish-approved-main/specs/release-metadata-delivery/spec.md)
- **Source task state:** isolated matching/stale-tag tests (`1.2`, `2.1`, `2.2`) are checked at the base; only `1.1` remains unchecked.
- **Remaining work:** after a release version/notes are approved under project policy, update `VERSION` and `CHANGELOG.md` for the reviewed commit; retain the existing tag-at-HEAD check; validate an isolated matching candidate. Actual tag creation and publication remain a separate release-owner operation and are not part of this plan.
- **Dependencies:** approved release version and notes; reviewed target commit; current release coverage implementation. Source explicitly prohibits weakening the tag-at-HEAD test or contacting a remote from its isolated test fixtures.
- **Owner / independent check:** Team 12 Release Manager owns metadata coordination and actual release operation; Team 7 Technical Writer prepares changelog; Team 10 QA reruns matching/stale/lightweight/missing-note fixtures; Team 5 validates the existing release workflow remains unchanged.
- **Acceptance / evidence:** `VERSION`, changelog, and annotated tag identify the same reviewed candidate in an isolated test; missing/lightweight/stale tag or missing notes fails; no real tag or publication is created by the test.
- **Model / effort:** Luna for metadata/changelog mechanics and test receipts; Sol only if a test fixture fails; lead agent reviews candidate alignment. **2–4 person-hours** for `1.1` and final evidence, excluding release decision and actual publication.
