# Active plans: deployment and operational qualification

Nine active proposals require an environment, rollout, consumer, operating
procedure, or authorization boundary. Select synthetic/isolated fixtures where
the proposal allows them. For pilot-specific work, obtain only the owner and
environment decisions explicitly named by the OpenSpec tasks; an unavailable
consumer or host remains a blocker, not a reason to substitute a mock.

The lead agent owns design choices and final signoff. The team assignments
below are implementation and independent-validation lanes, not extra approval
permissions. Only source tasks that explicitly call for an owner decision,
approved test identity, or authorized environment retain that prerequisite.

<a id="dep01-render-kubernetes-profile"></a>
### 1. `dep01-render-kubernetes-profile`

- **OpenSpec:** [proposal](../../../openspec/changes/dep01-render-kubernetes-profile/proposal.md) · [tasks](../../../openspec/changes/dep01-render-kubernetes-profile/tasks.md) · [Kubernetes profile delta](../../../openspec/changes/dep01-render-kubernetes-profile/specs/kubernetes-profile-evidence/spec.md)
- **Tasks:** all three tasks are unchecked: deterministic render for one explicitly supported private profile; validate render/profile/listener/Service/ingress/auth/probe consistency and reject unsafe mismatches before apply; document and exercise render/preflight/apply in a disposable supported cluster.
- **Dependencies:** choose the single supported private profile; pinned renderer and required Kubernetes client/cluster; exact source/image revision; synthetic credentials in protected test files.
- **Owner / independent check:** Team 5 Platform Engineering (Kubernetes Administrator/CI/CD Architect) with Team 4 IaC/Network Security; Team 10 QA checks render and reject cases; Team 9 AppSec inspects exposure/auth references.
- **Acceptance / evidence:** rendered output has no unresolved placeholder; probes/routes and bind profile agree; bad configurations fail before apply; runbook contains redacted commands, image/config revision, and observed disposable-cluster result.
- **Model / effort:** Sol for renderer/preflight validation; Luna for test cases and runbook; lead agent chooses supported-profile boundaries and signs the result. **10–18 person-hours**, excluding disposable cluster setup.

<a id="dep02-reconcile-production-guide"></a>
### 2. `dep02-reconcile-production-guide`

- **OpenSpec:** [proposal](../../../openspec/changes/dep02-reconcile-production-guide/proposal.md) · [tasks](../../../openspec/changes/dep02-reconcile-production-guide/tasks.md) · [production procedure delta](../../../openspec/changes/dep02-reconcile-production-guide/specs/production-deployment-procedure/spec.md)
- **Tasks:** all three tasks are unchecked: trace each current command to implementation/profile and record prerequisites/superseded path; update protected-secret, preflight, staging, migration, readiness and recovery instructions; execute the written sequence in isolated staging and record command-level results.
- **Dependencies:** stable supported profile/render flow from DEP-01; current migration and secret-source procedures; isolated staging access. Do not include private endpoint or credential values in examples.
- **Owner / independent check:** Team 7 Technical Writer owns the guide with Team 5 Platform Engineer for CI/deploy commands and Team 11 SRE for recovery steps; Team 10 QA executes the documented sequence; Team 9 AppSec reviews secret handling.
- **Acceptance / evidence:** every production command names its source revision/prerequisites; protected credentials and preflight are used; migration/readiness/recovery steps match current implementation; staging execution matches the guide; obsolete steps are clearly superseded.
- **Model / effort:** Luna for source-to-instruction reconciliation and edits; Sol if command smoke automation is needed; lead agent owns final technical wording. **8–14 person-hours** excluding staging provisioning.

<a id="ops01-approve-rollout-inventory"></a>
### 3. `ops01-approve-rollout-inventory`

- **OpenSpec:** [proposal](../../../openspec/changes/ops01-approve-rollout-inventory/proposal.md) · [tasks](../../../openspec/changes/ops01-approve-rollout-inventory/tasks.md) · [rollout inventory delta](../../../openspec/changes/ops01-approve-rollout-inventory/specs/rollout-inventory-and-ownership/spec.md)
- **Tasks:** all three tasks are unchecked. The source explicitly requires service, control, and operations owners to agree on a pilot and schema; then populate/approve a record, validate required fields/owners and publish a redacted stable reference; missing/unapproved fields keep readiness blocked.
- **Dependencies:** a real pilot consumer, topology, profile, boundaries, applicable gates, and named operating owners. Keep identities and environment details in the approved private system; source task requires a decision-owner disposition.
- **Owner / independent check:** Team 1 Product/Business Systems Analyst coordinates scope; Team 2 Solution Architect captures topology/profile; Team 3 GRC/Policy Manager defines controls and privacy; Team 11/12 operating roles supply service ownership. Team 10 QA checks required-field completeness and redaction.
- **Acceptance / evidence:** inventory records revision/date and redacted stable reference; every consumer/topology/profile/boundary/control/owner field exists; readiness remains blocked if approval or required data is absent.
- **Model / effort:** Luna for schema/checklist preparation and validation script; Sol only if a machine-readable validator is needed; lead agent coordinates the source-required owner decision. **6–10 person-hours** excluding owner response and private-system entry.

<a id="ops02-verify-external-consumer"></a>
### 4. `ops02-verify-external-consumer`

- **OpenSpec:** [proposal](../../../openspec/changes/ops02-verify-external-consumer/proposal.md) · [tasks](../../../openspec/changes/ops02-verify-external-consumer/tasks.md) · [external consumer delta](../../../openspec/changes/ops02-verify-external-consumer/specs/external-consumer-verification/spec.md)
- **Tasks:** all three tasks are unchecked: identify the pilot consumer and decide Gate 4 applicability with a decision owner; run isolated end-to-end signature/identity/policy/effect-boundary acceptance; publish redacted, revision-bound evidence and ensure observe-only receipts do not authorize effects.
- **Dependencies:** selected consumer and boundary; approved test identity; producer/verifier interface; image and policy revisions from the selected pilot. If the consumer is unavailable, mark blocked.
- **Owner / independent check:** Team 8 Integration Engineer owns the producer-consumer exchange; Team 3 GRC/Policy Manager scopes the evidence requirement; Team 9 AppSec verifies identity/signature; Team 10 QA checks the effect boundary and receipts.
- **Acceptance / evidence:** actual consumer verifies signature, identity and policy; effect boundary is observed; versioned evidence binds consumer/image/policy revisions; observe-only evidence is not used as authorization.
- **Model / effort:** Sol for the end-to-end fixture and interface adapter; Luna for redacted receipt/reporting; lead agent owns scope and signs evidence claims. **8–16 person-hours** excluding external consumer coordination.

<a id="ops03-verify-multi-instance-revocation"></a>
### 5. `ops03-verify-multi-instance-revocation`

- **OpenSpec:** [proposal](../../../openspec/changes/ops03-verify-multi-instance-revocation/proposal.md) · [tasks](../../../openspec/changes/ops03-verify-multi-instance-revocation/tasks.md) · [revocation delta](../../../openspec/changes/ops03-verify-multi-instance-revocation/specs/multi-instance-revocation-evidence/spec.md)
- **Tasks:** all three tasks are unchecked. Record selected replica/registry/failover path and propagation bound; obtain owner approval for the isolated drill; use a synthetic credential to observe every instance before/after failover; retain revision/timing evidence without live credential material.
- **Dependencies:** multi-instance pilot topology, registry distribution mechanism, failover path, bounded propagation target, and synthetic credential. If a supported topology is not selected, do not claim the process-local unit test proves it.
- **Owner / independent check:** Team 8 IAM/Integration owns registry distribution; Team 5 Platform configures isolated replicas; Team 11 SRE runs failover; Team 9 AppSec checks credential revocation semantics; Team 10 QA independently verifies per-instance deny and unaffected synthetic controls.
- **Acceptance / evidence:** every serving instance and failover path deny revoked access within the declared bound; unrelated synthetic credential remains unaffected; image/config IDs and timestamps are recorded with no token values.
- **Model / effort:** Sol for multi-replica harness and event recording; Luna for evidence matrix; lead agent coordinates source-required owner approval. **10–18 person-hours** excluding environment provisioning and approval wait.

<a id="ops04-rehearse-operational-recovery"></a>
### 6. `ops04-rehearse-operational-recovery`

- **OpenSpec:** [proposal](../../../openspec/changes/ops04-rehearse-operational-recovery/proposal.md) · [tasks](../../../openspec/changes/ops04-rehearse-operational-recovery/tasks.md) · [recovery delta](../../../openspec/changes/ops04-rehearse-operational-recovery/specs/operational-recovery-rehearsal/spec.md)
- **Tasks:** all three tasks are unchecked: select isolated staging topology and define measurable objectives/stop conditions/data/owners; rehearse ordered migration/readiness-drain/backup restore/recovery or rollback using synthetic or approved data; compare outcomes, assign remediation owners, and publish a redacted revision-bound reference.
- **Dependencies:** selected staging topology and recovery objectives; backup source/restore target; test data; operators responsible for database, app, and rollback steps. It is not closed by migration design docs alone.
- **Owner / independent check:** Team 11 SRE Lead owns rehearsal order and measurements; Team 12 Release Manager/Change Manager coordinates rollback window; Team 6 DBA verifies database integrity; Team 4 owns staging isolation; Team 10 QA independently checks objective/outcome alignment.
- **Acceptance / evidence:** restore integrity and readiness/drain behavior are measured against declared objectives; actual gaps/remediation owners are recorded; no production data or sensitive endpoints appear in published evidence.
- **Model / effort:** Luna for runbook and observation checklist; Sol for synthetic data/verification automation; lead agent coordinates execution and owns go/no-go evidence. **12–20 person-hours** excluding backup creation and external change windows.

<a id="tpl52-enforce-optional-surface-authorization"></a>
### 7. `tpl52-enforce-optional-surface-authorization`

- **OpenSpec:** [proposal](../../../openspec/changes/tpl52-enforce-optional-surface-authorization/proposal.md) · [tasks](../../../openspec/changes/tpl52-enforce-optional-surface-authorization/tasks.md) · [authorization delta](../../../openspec/changes/tpl52-enforce-optional-surface-authorization/specs/optional-mcp-surface-authorization/spec.md)
- **Source task state:** tasks `1.1`–`1.3` are checked; task `2.1` remains unchecked.
- **Remaining work:** independently verify the checked endpoint map and registered-handler allow/deny/disabled/redaction/effect tests; run pinned strict OpenSpec and relevant repository checks; archive only after the source's three scenarios are evidenced.
- **Dependencies:** per-operation caller identity, scope/resource rules and decision-audit path already present in the change; confirm optional MCP and REST paths remain configured as source expects.
- **Owner / independent check:** Team 8 IAM/API owner with Team 7 MCP/REST maintainers; Team 9 AppSec checks decision-before-handler and redaction; Team 10 QA reviews registered-handler evidence.
- **Acceptance / evidence:** authorized operations work; denied calls do not invoke handlers or effects; disabled exposure stays absent; decision audit is redacted; test transport/profile/revisions are recorded.
- **Model / effort:** Luna for acceptance/evidence reconciliation; Sol only for a concrete failure. **2–4 person-hours**.

<a id="tpl53-qualify-mcp-session-topology"></a>
### 8. `tpl53-qualify-mcp-session-topology`

- **OpenSpec:** [proposal](../../../openspec/changes/tpl53-qualify-mcp-session-topology/proposal.md) · [tasks](../../../openspec/changes/tpl53-qualify-mcp-session-topology/tasks.md) · [session topology delta](../../../openspec/changes/tpl53-qualify-mcp-session-topology/specs/mcp-session-topology/spec.md) · [session inventory evidence](../../../openspec/changes/tpl53-qualify-mcp-session-topology/evidence/2026-11-25-session-topology.md)
- **Source task state:** `1.1` is checked; `1.2`, `1.3`, and `2.1` remain unchecked.
- **Work sequence:** make render/preflight reject inconsistent profile/replica/routing/session modes; run isolated restart/reinitialize/revoke/identity-separation and mode-specific routing; independently verify expired/revoked/mismatched-principal scenarios and strict OpenSpec before archive.
- **Dependencies:** source inventory; deployment profile validator; session and identity state store; replica and routing configuration; a disposable topology. If state is process-local, qualify only the documented supported mode.
- **Owner / independent check:** Team 5 Platform owner for render/preflight; Team 7 MCP/backend owner for session handling; Team 8 IAM reviews principal separation; Team 11 SRE runs restart/failover; Team 10 QA validates each scenario; Team 9 AppSec checks revocation behavior.
- **Acceptance / evidence:** unsupported combinations fail preflight; supported restart reinitializes according to declared state mode; expired/revoked/mismatched identities deny; traffic routing and image/profile digests are recorded.
- **Model / effort:** Sol for config validation and lifecycle fixture; Luna for bounded topology matrix; lead agent owns support-mode decisions. **10–18 person-hours**, excluding topology provisioning.

<a id="tpl54-qualify-organization-scoped-documents"></a>
### 9. `tpl54-qualify-organization-scoped-documents`

- **OpenSpec:** [proposal](../../../openspec/changes/tpl54-qualify-organization-scoped-documents/proposal.md) · [tasks](../../../openspec/changes/tpl54-qualify-organization-scoped-documents/tasks.md) · [document access delta](../../../openspec/changes/tpl54-qualify-organization-scoped-documents/specs/organization-scoped-document-access/spec.md)
- **Tasks:** all four tasks are unchecked: inventory read/list/search/cache/ingestion/mutable paths; select approved permission source and access profile; enforce boundaries and record external retrieval interface/ownership; run paired-identity lookup/search/metadata/cache/deny/revoke/retire tests; independently inspect the three scenarios.
- **Dependencies:** actual organization identity and current source-permission authority; selected supported shared-access profile; document retrieval provider and cache; representative synthetic docs for at least two organizations. Do not infer permissions from caller-supplied organization strings.
- **Owner / independent check:** Team 6 Data Architect/DPO owns document classification and lifecycle; Team 8 IAM/API owner selects trusted permission input; Team 7 backend owns query/cache paths; Team 9 Privacy/AppSec checks cross-org isolation; Team 10 QA runs paired-principal and revocation tests.
- **Acceptance / evidence:** same-org permitted access works; cross-org or withdrawn authority cannot leak content, metadata, ranking or cache results; external profile is documented with source authority and bounded audit evidence.
- **Model / effort:** Sol for authorization/query/cache integration; Luna for path inventory, fixtures, and reports; lead agent owns design choice and final acceptance. **12–22 person-hours**, excluding external permission-source access.
