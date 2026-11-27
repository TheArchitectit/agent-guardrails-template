# Canonical sprint plan: Quality Evidence

> Planning baseline only. CLOSED means the canonical spec records exercised closure; this plan schedules regression/evidence review, not reimplementation. UNKNOWN means no status or closure table is recorded: first inspect implementation and evidence, then propose a narrowly scoped change only if a gap is confirmed. Do not infer status from archived task checkboxes.

## Operating model

- **Models:** use Luna (`gpt-6-luna`) for bounded implementation, test, and evidence preparation; use Astra (`gpt-6-astra`) for design decisions, security-sensitive review, and final sign-off. These are recommendations, not execution evidence.
- **Time:** per-spec ranges below are person-hours for the listed scope. Estimates exclude external approvals, credentials, unavailable services, and unrelated repair.
- **Platform applicability:** names identify where a requirement applies. Applicability is not a claim of current runtime compatibility. Source-only, CI, and server requirements are editor-neutral unless named.
- **Acceptance:** retain the canonical SHALL clauses and named scenarios as the acceptance source. Record exact commit, command/tool versions, output hashes where useful, and failures/skips.

## Planned work

<a id="audit-proposal-text-integrity"></a>
### [audit-proposal-text-integrity](../../../openspec/specs/audit-proposal-text-integrity/spec.md) — UNKNOWN

- **Disposition:** baseline status unknown; inspect code, tests, and current workflow evidence first; create a narrowly scoped implementation change only for a reproduced gap.
- **Scope:** Audit proposal section citations pass the existing charset policy.
- **Steps:** map each requirement to current source and focused tests; run or add bounded checks for every scenario below. Keep fixes scoped to a confirmed gap, preserve interfaces, and retain fail-closed behavior.
- **Scenario acceptance:** A proposal cites a section range — the candidate audit proposals are checked with the pinned charset gate; the scan finds no disallowed citation characters; the citations retain the original section numbers and range boundaries.
- **Evidence:** record exact revision, commands, tool versions, scenario outcomes, and logs or hashes in the owning change/evidence file; explicitly mark unavailable environments and failed checks. UNKNOWN remains UNKNOWN until source-backed implementation and test evidence justify a status update.
- **Ownership:** QA (lead); source-domain maintainer (review). **Coding-platform scope:** Editor-neutral source/CI behavior; no platform runtime compatibility claim.
- **Models/time:** Execution and evidence: Luna (1–2h); lead decision and sign-off: Astra (0.5–1h). Estimate excludes unavailable approvals or infrastructure.

<a id="numbered-design-status-evidence"></a>
### [numbered-design-status-evidence](../../../openspec/specs/numbered-design-status-evidence/spec.md) — UNKNOWN

- **Disposition:** baseline status unknown; inspect code, tests, and current workflow evidence first; create a narrowly scoped implementation change only for a reproduced gap.
- **Scope:** Reconciled claims retain dates and evidence.
- **Steps:** map each requirement to current source and focused tests; run or add bounded checks for every scenario below. Keep fixes scoped to a confirmed gap, preserve interfaces, and retain fail-closed behavior.
- **Scenario acceptance:** Dated claim retains evidence and residuals — a numbered design status claim is reconciled; its historical date and supporting source or test references remain visible and unresolved requirements or verification limits are stated..
- **Evidence:** record exact revision, commands, tool versions, scenario outcomes, and logs or hashes in the owning change/evidence file; explicitly mark unavailable environments and failed checks. UNKNOWN remains UNKNOWN until source-backed implementation and test evidence justify a status update.
- **Ownership:** QA (lead); source-domain maintainer (review). **Coding-platform scope:** Editor-neutral source/CI behavior; no platform runtime compatibility claim.
- **Models/time:** Execution and evidence: Luna (1–2h); lead decision and sign-off: Astra (0.5–1h). Estimate excludes unavailable approvals or infrastructure.

<a id="openspec-archive-provenance"></a>
### [openspec-archive-provenance](../../../openspec/specs/openspec-archive-provenance/spec.md) — UNKNOWN

- **Disposition:** baseline status unknown; inspect code, tests, and current workflow evidence first; create a narrowly scoped implementation change only for a reproduced gap.
- **Scope:** Archive manifest matches package outputs.
- **Steps:** map each requirement to current source and focused tests; run or add bounded checks for every scenario below. Keep fixes scoped to a confirmed gap, preserve interfaces, and retain fail-closed behavior.
- **Scenario acceptance:** Archive manifest matches package outputs — an archived package is listed in the manifest; its enumerated output paths match the package files and every historical reference resolves or is marked unavailable..
- **Evidence:** record exact revision, commands, tool versions, scenario outcomes, and logs or hashes in the owning change/evidence file; explicitly mark unavailable environments and failed checks. UNKNOWN remains UNKNOWN until source-backed implementation and test evidence justify a status update.
- **Ownership:** QA (lead); source-domain maintainer (review). **Coding-platform scope:** Editor-neutral source/CI behavior; no platform runtime compatibility claim.
- **Models/time:** Execution and evidence: Luna (1–2h); lead decision and sign-off: Astra (0.5–1h). Estimate excludes unavailable approvals or infrastructure.

<a id="platform-state-evidence"></a>
### [platform-state-evidence](../../../openspec/specs/platform-state-evidence/spec.md) — UNKNOWN

- **Disposition:** baseline status unknown; inspect code, tests, and current workflow evidence first; create a narrowly scoped implementation change only for a reproduced gap.
- **Scope:** Audit persistence claims remain source bounded.
- **Steps:** map each requirement to current source and focused tests; run or add bounded checks for every scenario below. Keep fixes scoped to a confirmed gap, preserve interfaces, and retain fail-closed behavior.
- **Scenario acceptance:** Audit store statement is source bounded — the platform inventory describes server audit persistence; it cites the inspected audit-store attachment and does not claim that every log event or deployment is durably covered..
- **Evidence:** record exact revision, commands, tool versions, scenario outcomes, and logs or hashes in the owning change/evidence file; explicitly mark unavailable environments and failed checks. UNKNOWN remains UNKNOWN until source-backed implementation and test evidence justify a status update.
- **Ownership:** QA (lead); source-domain maintainer (review). **Coding-platform scope:** Editor-neutral source/CI behavior; no platform runtime compatibility claim.
- **Models/time:** Execution and evidence: Luna (1–2h); lead decision and sign-off: Astra (0.5–1h). Estimate excludes unavailable approvals or infrastructure.

<a id="python-utc-timestamps"></a>
### [python-utc-timestamps](../../../openspec/specs/python-utc-timestamps/spec.md) — UNKNOWN

- **Disposition:** baseline status unknown; inspect code, tests, and current workflow evidence first; create a narrowly scoped implementation change only for a reproduced gap.
- **Scope:** UTC time comparisons use aware values; serialized UTC timestamp shape stays compatible.
- **Steps:** map each requirement to current source and focused tests; run or add bounded checks for every scenario below. Keep fixes scoped to a confirmed gap, preserve interfaces, and retain fail-closed behavior.
- **Scenario acceptance:** filter timestamped records by a UTC time window — a time-window filter compares a generated UTC boundary with a; both values are interpreted as UTC-aware instants and the comparison; emit a structured log or metric timestamp — a script emits a timestamp for a structured log, metric, audit; it retains the existing field name and serializes the UTC value as; parse an emitted UTC timestamp — an existing consumer parses an emitted timestamp with the current; it continues to interpret the value as UTC without a schema or key.
- **Evidence:** record exact revision, commands, tool versions, scenario outcomes, and logs or hashes in the owning change/evidence file; explicitly mark unavailable environments and failed checks. UNKNOWN remains UNKNOWN until source-backed implementation and test evidence justify a status update.
- **Ownership:** QA (lead); source-domain maintainer (review). **Coding-platform scope:** Editor-neutral source/CI behavior; no platform runtime compatibility claim.
- **Models/time:** Execution and evidence: Luna (2–4h); lead decision and sign-off: Astra (0.5–1h). Estimate excludes unavailable approvals or infrastructure.

<a id="test-fixture-permissions"></a>
### [test-fixture-permissions](../../../openspec/specs/test-fixture-permissions/spec.md) — UNKNOWN

- **Disposition:** baseline status unknown; inspect code, tests, and current workflow evidence first; create a narrowly scoped implementation change only for a reproduced gap.
- **Scope:** permission test fixtures have deterministic modes; production secret mode checks remain restrictive.
- **Steps:** map each requirement to current source and focused tests; run or add bounded checks for every scenario below. Keep fixes scoped to a confirmed gap, preserve interfaces, and retain fail-closed behavior.
- **Scenario acceptance:** permissive fixture is created under restrictive umask — a permission test needs a group-readable fixture while running; the fixture has the intended permissive mode and the test evaluates; fixture is created under ordinary umask — the same permission test runs under umask `0022`; it evaluates the same mode and produces the same pass or fail result; private secret is created — production code writes a credential or secret file on a POSIX host; its mode remains `0600` and a group/world-readable file is rejected.
- **Evidence:** record exact revision, commands, tool versions, scenario outcomes, and logs or hashes in the owning change/evidence file; explicitly mark unavailable environments and failed checks. UNKNOWN remains UNKNOWN until source-backed implementation and test evidence justify a status update.
- **Ownership:** QA (lead); source-domain maintainer (review). **Coding-platform scope:** Editor-neutral source/CI behavior; no platform runtime compatibility claim.
- **Models/time:** Execution and evidence: Luna (2–4h); lead decision and sign-off: Astra (0.5–1h). Estimate excludes unavailable approvals or infrastructure.

<a id="validation-regex-cache"></a>
### [validation-regex-cache](../../../openspec/specs/validation-regex-cache/spec.md) — UNKNOWN

- **Disposition:** baseline status unknown; inspect code, tests, and current workflow evidence first; create a narrowly scoped implementation change only for a reproduced gap.
- **Scope:** Concurrent cache operations are race-free; Regex behavior remains compatible.
- **Steps:** map each requirement to current source and focused tests; run or add bounded checks for every scenario below. Keep fixes scoped to a confirmed gap, preserve interfaces, and retain fail-closed behavior.
- **Scenario acceptance:** Concurrent callers compile the same pattern and clear the cache — multiple callers compile and match the same valid pattern while another caller clears the regex cache; all successful matches SHALL preserve the standard matching result and the race detector SHALL report no data race; Invalid patterns and timed matching — a caller compiles an invalid pattern or performs a match with a configured timeout; the existing error and timeout behavior SHALL remain unchanged.
- **Evidence:** record exact revision, commands, tool versions, scenario outcomes, and logs or hashes in the owning change/evidence file; explicitly mark unavailable environments and failed checks. UNKNOWN remains UNKNOWN until source-backed implementation and test evidence justify a status update.
- **Ownership:** QA (lead); source-domain maintainer (review). **Coding-platform scope:** Editor-neutral source/CI behavior; no platform runtime compatibility claim.
- **Models/time:** Execution and evidence: Luna (1–2h); lead decision and sign-off: Astra (0.5–1h). Estimate excludes unavailable approvals or infrastructure.
