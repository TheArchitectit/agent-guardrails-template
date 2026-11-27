# Active plans: platform and client compatibility

Scope: seven active OpenSpec changes involving client platforms, plugin/runtime
compatibility, developer CLI contracts, or client-facing CI evidence. Platform
support statements below are bounded by the proposal/task evidence at the base
revision, not inferred from repository presence.

The lead agent owns design choices and final signoff. The team assignments
below are recommended implementation and independent-validation lanes, not
additional approval permissions; optional consultations are identified as such.

## Platform coverage map

| Platform | Active work | Current evidence boundary at base |
|---|---|---|
| Pi | audit + `tpl44-pi-consumer-runtime-evidence` | Lockfile audit exists; registered Pi lifecycle evidence is still required. |
| VS Code | dependency audit | Runtime audit is explicitly separated from dev-only findings; no active hook-runtime qualification is included in these 28 changes. |
| JetBrains | `ide03` follow-up + compatibility matrix | One IDE 2023.3.2 runtime load is recorded; declared 223–241 compatibility is not yet closed. |
| Codex MCP client | CI/image qualification | Proposal requires the real built image, `initialize`, `tools/list`, and a benign `tools/call`; fixture-only evidence is insufficient. |
| Windows Docker Desktop pilot | `tpl43` | Cold-start and persisted-state acceptance is still required on a separate authorized Windows host/VM. |
| Team CLI | output contract | Selected status/query/history output work exists; additional command support and independent acceptance remain open. |

<a id="audit-javascript-dev-dependencies"></a>
### 1. `audit-javascript-dev-dependencies`

- **OpenSpec:** [proposal](../../../openspec/changes/audit-javascript-dev-dependencies/proposal.md) · [tasks](../../../openspec/changes/audit-javascript-dev-dependencies/tasks.md) · [delta spec](../../../openspec/changes/audit-javascript-dev-dependencies/specs/javascript-dependency-advisory-review/spec.md) · [current audit receipt](../../../openspec/changes/audit-javascript-dev-dependencies/evidence/2026-11-27/audit.md)
- **Remaining work:** `[ ] 1.2` record a maintainer disposition for each reported development-package entry; keep development and production audit results separate and avoid inferred reachability. Tasks `1.1` and `2.1` are checked at the base.
- **Dependencies:** the existing Pi and VS Code lockfiles and pinned package-manager/tool versions. Package-owner input is optional where needed; do not turn this evidence-only disposition into dependency upgrades.
- **Owner / independent check:** Team 5 Platform Engineering (CI/CD or developer advocate) with the owning extension maintainer; Team 9 AppSec reviews risk language; Team 10 QA checks lockfile/tool/date reproducibility; Team 7 Technical Writer checks the release-process link.
- **Platform boundary:** covers Pi and VS Code dependency inventories, not runtime permission hooks or vulnerability exploitability.
- **Acceptance / evidence:** every reported package has a dated disposition with package/version, lockfile identity, tool version, and separate `npm audit` versus `npm audit --omit=dev` outcomes; no reachability claim without proof.
- **Model / effort:** Luna for the disposition table and repeatable audit edits; Sol only if lockfile tooling must change. **2–4 person-hours** for remaining review/write/check work; excludes security-owner response time.

<a id="ide03-jetbrains-runtime-followup"></a>
### 2. `ide03-jetbrains-runtime-followup`

- **OpenSpec:** [proposal](../../../openspec/changes/ide03-jetbrains-runtime-followup/proposal.md) · [tasks](../../../openspec/changes/ide03-jetbrains-runtime-followup/tasks.md) · [delta spec](../../../openspec/changes/ide03-jetbrains-runtime-followup/specs/ide03-jetbrains-runtime/spec.md) · [evidence](../../../openspec/changes/ide03-jetbrains-runtime-followup/evidence.md)
- **Source task state:** `1.1`–`1.3` are checked at the base. Do not invent additional implementation tasks or call this a matrix qualification.
- **Remaining work:** independently inspect the pinned 2023.3.2 compile/instrumentation and compiled `GuardrailService.validateSelection` evidence; run strict OpenSpec and repository documentation/code gates; resolve any validator finding before archive. The proposal is a narrow follow-up, not a platform upgrade.
- **Dependencies:** the merged IDE-03 selection-field work and pinned IntelliJ 2023.3.2 artifacts.
- **Owner / independent check:** Team 7 Senior Backend/Technical Lead (plugin service request path) with Team 5 platform/build owner; Team 10 QA independently reruns compile/probe. Team 2 architecture is an optional payload-contract consultation.
- **Platform boundary:** JetBrains only, pinned to the proposal's IntelliJ build. Full 223–241 support belongs to the separate compatibility change below.
- **Acceptance / evidence:** compiled service sends the canonical selection field to the fixture; retain exact IDE/platform/JBR, dependency, and test-command versions from source evidence.
- **Model / effort:** Luna for evidence reconciliation and docs; Sol for any bounded build repair. Lead agent owns design and final signoff after the independent check. **1–3 person-hours** for remaining closure; assumes artifacts remain obtainable.

<a id="qualify-ci-environment-and-mcp-client"></a>
### 3. `qualify-ci-environment-and-mcp-client`

- **OpenSpec:** [proposal](../../../openspec/changes/qualify-ci-environment-and-mcp-client/proposal.md) · [tasks](../../../openspec/changes/qualify-ci-environment-and-mcp-client/tasks.md) · [CI parity spec](../../../openspec/changes/qualify-ci-environment-and-mcp-client/specs/ci-environment-parity/spec.md) · [Codex image spec](../../../openspec/changes/qualify-ci-environment-and-mcp-client/specs/mcp-image-codex-client-acceptance/spec.md) · [evidence](../../../openspec/changes/qualify-ci-environment-and-mcp-client/evidence.md)
- **Source task state:** `1.2`, `2.1`, `2.2`, and `3.1` are checked; `1.1`, `1.3`–`1.5`, `2.3`–`2.5`, and `3.2` remain unchecked. Check the linked task file and evidence for subsequent changes before execution.
- **Work sequence:** inventory deterministic checks across all nine workflows and event/artifact-only gates; test configured runner selection and fork rejection; exercise synthetic context isolation on AI03–AI05 without host sockets; document parity limitations; run the exact image/client qualification manually when requested; add a candidate-build job that builds the real image and runs deterministic protocol initialize/list/call checks without any Codex model invocation; independently review receipts and strict validation.
- **Dependencies:** supported runner registration and authorized UCS03 access; AI03–AI05 disposable paths; Go/Podman and pinned toolchains; synthetic secrets. The automated job needs a deterministic MCP protocol client, not model credentials. The [design](../../../openspec/changes/qualify-ci-environment-and-mcp-client/design.md) describes a proposed short-lived identity and isolated worker; neither that worker nor a trusted CI-to-AI05 endpoint exists yet. A Codex client is needed only for the manual/on-demand Codex task. A missing prerequisite becomes an explicit unavailable result.
- **Owner / independent check:** Team 5 CI/CD Architect owns parity and runner-selection work; Team 7 MCP owner owns image/protocol behavior; Team 4 infrastructure supports disposable hosts; Team 10 QA checks the matrix; Team 9 AppSec validates isolation and secret-leak evidence; Team 12 Release Manager is the workflow documentation lane.
- **Platform boundary:** actual Codex MCP client against the server image. It does not substitute for Pi or JetBrains extension-runtime evidence.
- **Acceptance / evidence:** remaining local/hosted/UCS03 parity, configured runner/fork guard, and AI03–AI05 synthetic context isolation must be evidenced. Existing receipts prove a single manual AI05 image/Codex run, selected hosted/UCS03 comparisons, and only a partial Python node-ID comparison; they do not prove full environment parity or per-build gating. The manual Codex run negotiated `2025-03-26`, listed 50 tools, and returned 9 advisors from a benign call. **Unattended candidate builds must not invoke Codex or a model, consume model tokens, or hold model credentials.** The per-build gate should build the real image and use a deterministic MCP protocol client for initialize/list/call, binding the result to source SHA and image digest. Codex calls stay manual/on-demand. Review build context/logs for secrets and unintended files; retain SHA/digest-bound results.
- **Model / effort:** Luna for docs/matrix and bounded scripts; Sol for deterministic image/protocol harness and workflow integration; Astra may advise on isolation design. Lead agent owns design and signoff after independent Team 9/10 validation. **18–30 person-hours** for remaining work, excluding host queueing and private runner administration.

<a id="qualify-jetbrains-plugin-compatibility"></a>
### 4. `qualify-jetbrains-plugin-compatibility`

- **OpenSpec:** [proposal](../../../openspec/changes/qualify-jetbrains-plugin-compatibility/proposal.md) · [tasks](../../../openspec/changes/qualify-jetbrains-plugin-compatibility/tasks.md) · [compatibility spec](../../../openspec/changes/qualify-jetbrains-plugin-compatibility/specs/jetbrains-plugin-compatibility/spec.md) · [evidence](../../../openspec/changes/qualify-jetbrains-plugin-compatibility/evidence.md)
- **Source task state:** `1.4` is checked and records one 2023.3.2 runtime load; `1.1`–`1.3` and `2.1` remain unchecked.
- **Work sequence:** establish the actual IntelliJ product/build/JBR matrix from plugin source APIs; align Java/Kotlin bytecode and generated plugin metadata to the verified runtime floor; run pinned wrapper and JetBrains verification for the declared builds; test the exact packaged ZIP in a load/service fixture; independently review and run strict OpenSpec.
- **Dependencies:** `ide03-jetbrains-runtime-followup`; official artifacts for the oldest and newest declared builds; an exact packaged plugin ZIP; verifier/runtime access. Keep product gaps separate when a product is not available.
- **Owner / independent check:** Team 5 platform/build owner with Team 7 plugin maintainer; Team 10 QA validates packaging and load. Team 2 architecture is an optional support-floor consultation; lead agent owns the support claim and signoff.
- **Platform boundary:** JetBrains IDE/plugin only. The single 2023.3.2 load does not prove the advertised 223–241 range; a failed or unavailable product/version remains explicitly unsupported or unverified.
- **Acceptance / evidence:** packaged metadata agrees with tested builds; Java/Kotlin classes load on each declared endpoint; verifier results bind to exact artifact/product build; canonical selection request remains correct; one-build versus full-range evidence is stated accurately.
- **Model / effort:** Luna for matrix/doc updates and pinned verification scripts; Sol for Gradle/bytecode changes; lead agent chooses and signs the support floor after independent verification. **12–20 person-hours**, excluding download/licensing/VM waits.

<a id="reconcile-team-cli-output-contract"></a>
### 5. `reconcile-team-cli-output-contract`

- **OpenSpec:** [proposal](../../../openspec/changes/reconcile-team-cli-output-contract/proposal.md) · [tasks](../../../openspec/changes/reconcile-team-cli-output-contract/tasks.md) · [output contract spec](../../../openspec/changes/reconcile-team-cli-output-contract/specs/team-cli-output-contract/spec.md) · [evidence](../../../openspec/changes/reconcile-team-cli-output-contract/evidence.md)
- **Source task state:** `1.1`, `1.2`, and `2.1` are checked; `2.2` remains unchecked.
- **Remaining work:** reviewer compares the command-by-command support matrix with actual Go wrapper, Python backend, and installed-prefix behavior; reruns the disposable CLI/backend regression and pinned strict OpenSpec; only then archive if accepted. Additional serializers are not implied by the checked implementation.
- **Dependencies:** Go Team CLI distribution and Python backend installed-prefix fixture.
- **Owner / independent check:** Team 7 Technical Lead and Senior Backend Engineer; Team 8 Integration Engineer reviews argument/format forwarding; Team 10 QA independently reruns the supported/unsupported output cases.
- **Platform boundary:** Team CLI / Go-to-Python command integration, not Pi, VS Code, or JetBrains runtime acceptance.
- **Acceptance / evidence:** supported JSON commands emit one structured stdout result and retain text compatibility; unsupported global output is rejected before side effects; existing query/history paths remain supported only as the matrix records.
- **Model / effort:** Luna for matrix/review and docs; Sol for a reproducible command-wrapper regression if needed. **2–4 person-hours** for closure.

<a id="tpl43-windows-cold-start-acceptance"></a>
### 6. `tpl43-windows-cold-start-acceptance`

- **OpenSpec:** [proposal](../../../openspec/changes/tpl43-windows-cold-start-acceptance/proposal.md) · [tasks](../../../openspec/changes/tpl43-windows-cold-start-acceptance/tasks.md) · [pilot spec](../../../openspec/changes/tpl43-windows-cold-start-acceptance/specs/local-docker-compose-pilot/spec.md)
- **Remaining work:** all tasks `1.1`–`1.2`, `2.1`–`2.3` are unchecked at the base. Select a separate authorized Windows host and unique user-local state; prove the fixed project name is unoccupied without inspecting the original pilot; predeclare stop conditions and secret-safe comparison; execute stopped-Desktop first run and restart with volume-preserving teardown; independently verify redacted receipts.
- **Dependencies:** separate authorized Windows VM/host; local Docker Desktop Linux engine and named-pipe context; pinned launcher, synthetic secrets and unique fixture paths; no access to the original pilot's stack or secrets.
- **Owner / independent check:** Team 5 platform owner for launcher/runtime; Team 4 infrastructure for the isolated VM; Team 10 QA independently checks behavior and cleanup; Team 9 AppSec validates secret-safe evidence. Use only the separate authorized host required by the proposal.
- **Platform boundary:** Windows Docker Desktop launcher lifecycle only; it does not qualify general Linux deployment or other coding assistants.
- **Acceptance / evidence:** cold start from stopped Desktop and unavailable local engine reaches app readiness within five minutes; first-run generated credential properties remain private; rerun reuses credentials and named volumes after teardown without `--volumes`; wrong engine, conflict, or unverified secret continuity fails safely.
- **Model / effort:** Luna for bounded launcher docs and receipt templates; Sol for any test harness scripts; host actions require an operator with the authorized Windows environment. Lead agent owns final signoff after independent validation. **8–14 person-hours** excluding Windows VM provisioning and reboot/window delays.

<a id="tpl44-pi-consumer-runtime-evidence"></a>
### 7. `tpl44-pi-consumer-runtime-evidence`

- **OpenSpec:** [proposal](../../../openspec/changes/tpl44-pi-consumer-runtime-evidence/proposal.md) · [tasks](../../../openspec/changes/tpl44-pi-consumer-runtime-evidence/tasks.md) · [Pi consumer spec](../../../openspec/changes/tpl44-pi-consumer-runtime-evidence/specs/pi-consumer-runtime-evidence/spec.md)
- **Remaining work:** tasks `1.1`–`1.4` and `2.1`–`2.2` are unchecked. Build a synthetic host lifecycle fixture through production handler registration/dispatch; exercise allowed/denied read/edit/bash, classifier pass/block/unavailable, and permission `auto`/`ask`/`blocked`; run synthetic secret/PII through real output validation; independently confirm production event-path use and classify warning versus actual redaction.
- **Dependencies:** Pi extension source revision and matching SDK/toolchain; synthetic host APIs and effect counters; configured classifier identity or an explicit unavailable case.
- **Owner / independent check:** Team 7 Pi extension maintainer; Team 8 Integration Engineer reviews event/handler dispatch; Team 9 AppSec checks deny-before-effect and sensitive-output handling; Team 10 QA independently reviews fixtures and receipts.
- **Platform boundary:** Pi extension lifecycle only. Explicit MCP tools are not evidence of automatic Pi integration. VS Code's active proposal here is dependency audit only.
- **Acceptance / evidence:** outcomes tie the registered hook, decision order, and no-effect denial; redaction evidence shows returned content changed or is labeled warning-only; record extension/config/classifier revisions and fixture IDs but not fixture payloads or secrets.
- **Model / effort:** Luna for fixture scaffolding and evidence table; Sol for event-dispatch test harness; lead agent owns semantic claims and signoff after independent validation. **8–14 person-hours** assuming the SDK lifecycle is available locally.
