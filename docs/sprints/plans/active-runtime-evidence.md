# Active plans: runtime conformance and decision evidence

Nine active changes define or verify runtime-bound evidence. Keep source-level,
unit-test, manager-level, registered-handler, and production-startup evidence
separate. Each proposal's spec scenarios govern acceptance; the cards below do
not broaden claims to a consumer platform that was not exercised.

The lead agent owns design decisions and final signoff. Team assignments below
are implementation/independent-validation lanes, not permission gates; Team 2,
Team 3, and Team 9 consultations are optional unless a source task explicitly
requires a named owner decision or authorized fixture.

<a id="bind-assembly-to-runtime-evidence"></a>
### 1. `bind-assembly-to-runtime-evidence`

- **OpenSpec:** [proposal](../../../openspec/changes/bind-assembly-to-runtime-evidence/proposal.md) · [tasks](../../../openspec/changes/bind-assembly-to-runtime-evidence/tasks.md) · [delta spec](../../../openspec/changes/bind-assembly-to-runtime-evidence/specs/assembly-runtime-evidence/spec.md)
- **Tasks:** all four `[TPL-36]` tasks are unchecked: define selected-assembly/build/policy/profile/check identity; bind runtime observations and reject missing or mismatched identity; test exact-match and controlled-mismatch fixtures; preserve the Spec 08 runtime contract and separate external-consumer acceptance.
- **Dependencies:** runtime-observable build and policy identity; composition/assembly manifest and the policy-pack activation evidence from TPL-45; do not infer external consumer acceptance from this binding.
- **Owner / independent check:** Team 7 backend owner for evidence schema/runtime observation; Team 2 Solution Architect for identity contract; Team 10 QA independently exercises match/mismatch; Team 9 AppSec reviews integrity claims.
- **Acceptance / evidence:** chosen assembly's MCP digest, policy digest, runtime profile, and check identities match live observation; missing or different inputs produce an explicit failed/incomplete result. Retain digest-bound fixtures and response.
- **Model / effort:** Luna for field wiring/docs; Sol for cross-component evidence code. Lead agent signs identity semantics after independent validation; Team 2 consultation is optional. **8–14 person-hours**.

<a id="bind-decision-evidence-context"></a>
### 2. `bind-decision-evidence-context`

- **OpenSpec:** [proposal](../../../openspec/changes/bind-decision-evidence-context/proposal.md) · [tasks](../../../openspec/changes/bind-decision-evidence-context/tasks.md) · [prompt-injection evidence spec](../../../openspec/changes/bind-decision-evidence-context/specs/prompt-injection-evidence-and-conformance/spec.md)
- **Tasks:** all three `[RUN-04]` tasks are unchecked: bind active policy digest, classifier/version, category, outcome, and subject; represent signature presence/verification; offline-verify signed evidence and label legacy records incomplete.
- **Dependencies:** policy/classifier identity source; signed evidence verifier and stable legacy envelope. Coordinate field names with release conformance reports.
- **Owner / independent check:** Team 7 backend/evidence owner; Team 2 architecture for canonical fields; Team 9 AppSec verifies signing claims; Team 10 QA tests signed, unsigned, altered, and incomplete records.
- **Acceptance / evidence:** blocked injection record names active policy and classifier context; unsigned is never called sealed; offline verification rejects alterations and preserves explicit incomplete state without upgrading it to verified.
- **Model / effort:** Luna for additive schema and tests; Sol for verifier-path changes; lead agent owns and signs trust semantics after independent validation. **6–10 person-hours**.

<a id="implement-semantic-redaction-evidence"></a>
### 3. `implement-semantic-redaction-evidence`

- **OpenSpec:** [proposal](../../../openspec/changes/implement-semantic-redaction-evidence/proposal.md) · [tasks](../../../openspec/changes/implement-semantic-redaction-evidence/tasks.md) · [action delta](../../../openspec/changes/implement-semantic-redaction-evidence/specs/semantic-filtering-categories-and-actions/spec.md) · [evidence delta](../../../openspec/changes/implement-semantic-redaction-evidence/specs/semantic-filtering-evidence-and-conformance/spec.md)
- **Tasks:** all three `[RUN-05]` items are unchecked: execute configured redaction before return; emit category/rule and removed-span digest without retaining raw content; fail closed for unsupported actions and test permitted/unsupported paths.
- **Dependencies:** declared semantic action schema and configured runtime; `verify-semantic-conformance-evidence` defines category fixtures, while `bind-decision-evidence-context` defines cross-cutting identity fields.
- **Owner / independent check:** Team 7 backend owner; Team 8 API/integration reviews output boundary; Team 9 privacy/AppSec reviews secret removal and digest handling; Team 10 QA checks original versus returned fixture bytes.
- **Acceptance / evidence:** permitted redaction changes returned text and records only safe digest/category/rule metadata; unsupported action fails closed; no removed raw span appears in output, logs, or evidence.
- **Model / effort:** Sol for action-path changes and tests; Luna for fixture/docs work; lead agent signs the privacy claim after independent Team 9/10 validation. **8–16 person-hours**.

<a id="trace-action-and-ingress-boundaries"></a>
### 4. `trace-action-and-ingress-boundaries`

- **OpenSpec:** [proposal](../../../openspec/changes/trace-action-and-ingress-boundaries/proposal.md) · [tasks](../../../openspec/changes/trace-action-and-ingress-boundaries/tasks.md) · [classification delta](../../../openspec/changes/trace-action-and-ingress-boundaries/specs/classification-and-mediation/spec.md) · [provenance delta](../../../openspec/changes/trace-action-and-ingress-boundaries/specs/provenance-records/spec.md)
- **Tasks:** all four `[RUN-03]` items are unchecked: inventory destructive call sites and prove guard-before-effect or narrow the claim; verify permitted/denied actions; inventory external-content paths and prove provenance before acceptance or mark unverifiable; test channel-derived/missing provenance separately from caller-invoked MCP scanning.
- **Dependencies:** current call-site inventory for each shipped consumer; production handler pathways and effect counters; do not substitute unit helpers for a registered consumer path.
- **Owner / independent check:** Team 7 feature/backend owner for MCP path; Team 8 Integration owner for consumer edges; Team 9 AppSec reviews deny-before-effect and trust edges; Team 10 QA checks positive/negative side effects.
- **Acceptance / evidence:** each claim is linked to an actual call site and runtime path; denial precedes effect; uninstrumented ingress is recorded as a gap; explicit scan tools are described as caller-invoked only.
- **Model / effort:** Sol for call-site/effect instrumentation tests; Luna for inventory and evidence matrix; lead agent owns claim boundaries and signs off after independent validation. **10–18 person-hours**.

<a id="verify-sandbox-runtime-conformance"></a>
### 5. `verify-sandbox-runtime-conformance`

- **OpenSpec:** [proposal](../../../openspec/changes/verify-sandbox-runtime-conformance/proposal.md) · [tasks](../../../openspec/changes/verify-sandbox-runtime-conformance/tasks.md) · [sandbox spec](../../../openspec/changes/verify-sandbox-runtime-conformance/specs/sandbox-conformance-and-evidence/spec.md)
- **Tasks:** all six `[RUN-01]` tasks are unchecked: exercise benign permitted and denied operations per supported runtime; record runtime/host/isolation; bind declared and actual isolation plus completion into signed evidence; hermetically verify fields/signature and interruption cases; reconcile safe capability checks versus escape-fixture conformance; mark only what was exercised.
- **Dependencies:** selected supported-runtime list and host profile; signed-envelope verifier; disposable, non-privileged fixtures. The task explicitly keeps escape-fixture conformance distinct from safe capability checks.
- **Owner / independent check:** Team 5 platform/runtime owner with Team 7 sandbox implementation owner; Team 9 AppSec reviews containment claims; Team 10 QA verifies actual observation and negative cases; Team 4 owns test-host isolation.
- **Acceptance / evidence:** benign allow succeeds, deny is observed blocked; signed evidence binds runtime/version/actual isolation and completion; fault/interruption cannot be reported as successful completion; only actually tested capabilities receive an exercised status. Escape testing stays explicitly unverified unless separately scoped and run.
- **Model / effort:** Sol for runtime/evidence harness; Luna for safe fixtures/docs; Astra may advise on containment assumptions. Lead agent retains scope and signoff after independent Team 9/10 validation. **12–20 person-hours**, excluding host/escape-fixture provisioning.

<a id="verify-semantic-conformance-evidence"></a>
### 6. `verify-semantic-conformance-evidence`

- **OpenSpec:** [proposal](../../../openspec/changes/verify-semantic-conformance-evidence/proposal.md) · [tasks](../../../openspec/changes/verify-semantic-conformance-evidence/tasks.md) · [semantic evidence spec](../../../openspec/changes/verify-semantic-conformance-evidence/specs/semantic-filtering-evidence-and-conformance/spec.md)
- **Tasks:** all three `[RUN-02]` items are unchecked: define/pin representative positives and benign negatives with per-category detection/false-positive thresholds; evaluate using configured classifier and fail either threshold; label reference-classifier output harness-only.
- **Dependencies:** selected classifier configuration and stable model/version identity; representative examples for every shipped category; redaction action work is tracked separately by RUN-05. Team 3/category owners may be consulted on category meaning.
- **Owner / independent check:** Team 7 backend owner for corpus runner; Team 10 QA owns measurement reproducibility; Team 9 checks fixtures for sensitive content. Lead agent selects threshold semantics; Team 2/3 domain consultation is optional.
- **Acceptance / evidence:** category-level rates, both thresholds, classifier/version, corpus digest, and misses are reproducible; failing/missing categories fail verification; reference-classifier data is never used as production performance evidence.
- **Model / effort:** Sol for measurement gate; Luna for corpus manifests and reports; Astra may advise on methodology. Lead agent decides thresholds and signs off after independent validation. **16–28 person-hours**, excluding classifier service access.

<a id="tpl45-policy-pack-activation-evidence"></a>
### 7. `tpl45-policy-pack-activation-evidence`

- **OpenSpec:** [proposal](../../../openspec/changes/tpl45-policy-pack-activation-evidence/proposal.md) · [tasks](../../../openspec/changes/tpl45-policy-pack-activation-evidence/tasks.md) · [activation spec](../../../openspec/changes/tpl45-policy-pack-activation-evidence/specs/policy-pack-runtime-activation/spec.md)
- **Tasks:** all five numbered tasks are unchecked: create disposable signed fixtures; test manager-level atomicity and label its scope; start production startup with valid trust/lock/pack; exercise bad trust/bytes/key/signature controls; independently compare observations and export redacted evidence.
- **Dependencies:** explicit trust key set and lock schema; production server config/startup; runtime identity and decision attribution. Manager-only results do not satisfy startup acceptance.
- **Owner / independent check:** Team 7 policy-pack owner; Team 9 AppSec checks ephemeral keys and invalid-candidate refusal; Team 10 QA starts the actual server and checks diagnostics. Team 2 architecture is an optional contract consultation.
- **Acceptance / evidence:** valid startup records signer, lock/content/effective digest, and rule attribution; invalid variants never become active; failed manager candidate preserves prior set; no private key or policy secret is retained.
- **Model / effort:** Sol for startup fixture/harness; Luna for manifest and evidence formats; lead agent signs trust decisions after independent validation. **10–18 person-hours**.

<a id="tpl46-multi-agent-enforcement-evidence"></a>
### 8. `tpl46-multi-agent-enforcement-evidence`

- **OpenSpec:** [proposal](../../../openspec/changes/tpl46-multi-agent-enforcement-evidence/proposal.md) · [tasks](../../../openspec/changes/tpl46-multi-agent-enforcement-evidence/tasks.md) · [multi-agent spec](../../../openspec/changes/tpl46-multi-agent-enforcement-evidence/specs/multi-agent-runtime-enforcement/spec.md)
- **Tasks:** all seven `[TPL-46]` items are unchecked: start production constructor with digest-bound config; call registered tool; verify config overrides defaults and validator ordering; exercise unavailable/error/conflict strategies; record refusal of unsupported YAML/LLM cases; produce redacted evidence; independently trace registry-to-dispatch and bound claims to supported MCP path.
- **Dependencies:** configured safety-chain and conflict-strategy schema; actual registered MCP tool path; explicit supported versus refused feature list.
- **Owner / independent check:** Team 7 multi-agent backend owner; Team 8 Integration Engineer traces registered tool; Team 9 AppSec checks failure-to-pass risk; Team 10 QA exercises ordering and strategy cases. Team 2 architecture is an optional scope consultation.
- **Acceptance / evidence:** request/config revisions bind outputs; configured order and conflict strategy are observed; unavailable validators never become pass; unsupported features remain explicitly refused; no generic host or Pi integration claim.
- **Model / effort:** Sol for integrated MCP fixture; Luna for bounded failure tests/evidence; lead agent owns scope and final signoff after independent validation. **8–14 person-hours**.

<a id="tpl47-compliance-live-probe-evidence"></a>
### 9. `tpl47-compliance-live-probe-evidence`

- **OpenSpec:** [proposal](../../../openspec/changes/tpl47-compliance-live-probe-evidence/proposal.md) · [tasks](../../../openspec/changes/tpl47-compliance-live-probe-evidence/tasks.md) · [probe evidence spec](../../../openspec/changes/tpl47-compliance-live-probe-evidence/specs/compliance-probe-report-evidence/spec.md)
- **Tasks:** all six `[TPL-47]` items are unchecked: select authorized isolated runtime and probe sources; collect positive, absent/unavailable, and empty-probe cases through real MCP/runtime; compare each report row to observations/config/interval; review privacy/integrity; independent source audit; list unobservable gaps without certification claims.
- **Dependencies:** an approved framework/config, concrete probe sources, collection interval, owner, runtime path, and redaction rules. Without these, hold the probe run rather than simulate a result.
- **Owner / independent check:** Team 7 implements/exports; Team 9 checks privacy/integrity; Team 10 validates each report row against actual probe output. Team 3 GRC and Team 6 Data Governance are optional framework/retention consultations unless named as source-task owners.
- **Acceptance / evidence:** at least one observed and one unknown control plus empty-source refusal; declarations or simulated collection are not counted as live probes; report digest/signature and runtime identity are captured; remaining controls are marked unmeasured.
- **Model / effort:** Luna for probe matrix/report templates; Sol for live-path instrumentation; lead agent signs control interpretation after independent validation. **8–16 person-hours**, excluding external probe setup.
