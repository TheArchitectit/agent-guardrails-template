# OpenSpec index

> **Generated, not typed.** Status is derived from each spec's own closure
> table (the per-requirement `EXERCISED` / `WIRED` / `ABSENT` / `NOT_RUN`
> cells) or its explicit `Status:` line. A spec with no closure evidence is
> reported **UNKNOWN** rather than guessed at.

**8 specs** — CLOSED 3 · PARTIAL 5 · OPEN 0 · UNKNOWN 0

| Spec | Title | Status | Evidence |
|---|---|---|---|
| [`prompt-injection-classification-and-mediation/spec.md`](specs/prompt-injection-classification-and-mediation/spec.md) | Prompt Injection Classification and Mediation | **CLOSED** | Status: CLOSED (EXERCISED 3) |
| [`sandbox-isolation-levels/spec.md`](specs/sandbox-isolation-levels/spec.md) | Sandbox Isolation Levels | **CLOSED** | Status: CLOSED (EXERCISED 3) |
| [`semantic-filtering-categories-and-actions/spec.md`](specs/semantic-filtering-categories-and-actions/spec.md) | Semantic Filtering Categories and Actions | **CLOSED** | Status: CLOSED (EXERCISED 3) |
| [`indirect-injection-provenance-records/spec.md`](specs/indirect-injection-provenance-records/spec.md) | Provenance Records | **PARTIAL** | Status: PARTIAL (EXERCISED 2, WIRED 1) |
| [`prompt-injection-evidence-and-conformance/spec.md`](specs/prompt-injection-evidence-and-conformance/spec.md) | Prompt Injection Evidence and Conformance | **PARTIAL** | Status: PARTIAL (EXERCISED 2, WIRED 1) |
| [`provenance-policy-and-conformance/spec.md`](specs/provenance-policy-and-conformance/spec.md) | Provenance Policy and Conformance | **PARTIAL** | Status: PARTIAL (EXERCISED 1, WIRED 1) |
| [`sandbox-conformance-and-evidence/spec.md`](specs/sandbox-conformance-and-evidence/spec.md) | Sandbox Conformance and Evidence | **PARTIAL** | Status: PARTIAL (EXERCISED 1, WIRED 1) |
| [`semantic-filtering-evidence-and-conformance/spec.md`](specs/semantic-filtering-evidence-and-conformance/spec.md) | Semantic Filtering Evidence and Conformance | **PARTIAL** | Status: PARTIAL (EXERCISED 1, WIRED 2) |

## Change packages (`openspec/changes/`)

**4 packages**

| Change | Tasks | Spec deltas | Notes |
|---|---|---|---|
| [`aiggp-03-prompt-injection-defense`](changes/aiggp-03-prompt-injection-defense/proposal.md) | **4/5** | 2 | L1/L2/L3 pipeline, MCP tools, and Ed25519 evidence wired; standalone third-party conformance gate open |
| [`aiggp-04-semantic-content-filtering`](changes/aiggp-04-semantic-content-filtering/proposal.md) | **3/5** | 2 | S1–S15 taxonomy, backends, config loader, and fail-closed MCP tools wired; labelled corpus runner and drift monitor open |
| [`aiggp-05-runtime-sandbox-isolation`](changes/aiggp-05-runtime-sandbox-isolation/proposal.md) | **4/5** | 2 | L1/L2 confinement, live capability probe, and MCP tools wired; four-level container-escape certification harness open |
| [`aiggp-06-indirect-injection-provenance`](changes/aiggp-06-indirect-injection-provenance/proposal.md) | **4/5** | 2 | Sanitization-before-scan, trust-class policy, and MCP provenance tools wired; multi-hop append-only ancestry query open |

## Reading the status

- **CLOSED** — every requirement row is `EXERCISED`: implementation present
  and a test drives it.
- **PARTIAL** — some rows `EXERCISED`, at least one `WIRED`/`ABSENT`/
  `NOT_RUN`/`DIFFERS`. The gap is named in that spec.
- **OPEN** — no requirement reaches `EXERCISED`.
- **UNKNOWN** — no closure table and no `Status:` line. A documentation gap,
  not a claim the work is unfinished. Fix by adding the closure table.
