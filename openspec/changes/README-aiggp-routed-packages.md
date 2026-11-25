# Routed packages (aiggp-03/04/05/06)

Moved 2026-10-11 from `TheArchitectit/repo-brainstorming` `devgate-open-changes/`
per the owner-approved routing (Roger 2026-10-11, QUEUE A6/A10): these are
Guardrails-family specs -- they name this product's Go MCP server as their calling
system, not DevGate's. Verbatim copies (IDs, cross-references, draft text
preserved); staged as native `openspec/changes/` packages here.

Relationship to the pre-reboot drafts in `docs/specs/guardrail-gaps-2026/`:
those are the earlier proposals reconciled in `STATUS.md` (261 rows). These
packages are the newer imported drafts; any reconcile/edit is future work.
Removal from the hub repo happened in the same change in
`TheArchitectit/repo-brainstorming` (README rule 2).

**Archived 2026-11-06.** The four historical packages contain eight delta
files mapping to seven unique published spec paths. The table links each
archived delta to its corresponding published file; the shared
`evidence-and-conformance` destination retains both package mappings.

| Archived package | Archived delta | Published spec |
|---|---|---|
| [aiggp-03 prompt injection](archive/2026-11-06-aiggp-03-prompt-injection-defense/) | [classification-and-mediation](archive/2026-11-06-aiggp-03-prompt-injection-defense/specs/classification-and-mediation/spec.md) | [classification-and-mediation](../specs/classification-and-mediation/spec.md) |
| [aiggp-03 prompt injection](archive/2026-11-06-aiggp-03-prompt-injection-defense/) | [evidence-and-conformance](archive/2026-11-06-aiggp-03-prompt-injection-defense/specs/evidence-and-conformance/spec.md) | [evidence-and-conformance](../specs/evidence-and-conformance/spec.md) |
| [aiggp-04 semantic filtering](archive/2026-11-06-aiggp-04-semantic-content-filtering/) | [categories-and-actions](archive/2026-11-06-aiggp-04-semantic-content-filtering/specs/categories-and-actions/spec.md) | [categories-and-actions](../specs/categories-and-actions/spec.md) |
| [aiggp-04 semantic filtering](archive/2026-11-06-aiggp-04-semantic-content-filtering/) | [evidence-and-conformance](archive/2026-11-06-aiggp-04-semantic-content-filtering/specs/evidence-and-conformance/spec.md) | [evidence-and-conformance](../specs/evidence-and-conformance/spec.md) |
| [aiggp-05 runtime isolation](archive/2026-11-06-aiggp-05-runtime-sandbox-isolation/) | [conformance-and-evidence](archive/2026-11-06-aiggp-05-runtime-sandbox-isolation/specs/conformance-and-evidence/spec.md) | [conformance-and-evidence](../specs/conformance-and-evidence/spec.md) |
| [aiggp-05 runtime isolation](archive/2026-11-06-aiggp-05-runtime-sandbox-isolation/) | [isolation-levels](archive/2026-11-06-aiggp-05-runtime-sandbox-isolation/specs/isolation-levels/spec.md) | [isolation-levels](../specs/isolation-levels/spec.md) |
| [aiggp-06 indirect provenance](archive/2026-11-06-aiggp-06-indirect-injection-provenance/) | [policy-and-conformance](archive/2026-11-06-aiggp-06-indirect-injection-provenance/specs/policy-and-conformance/spec.md) | [policy-and-conformance](../specs/policy-and-conformance/spec.md) |
| [aiggp-06 indirect provenance](archive/2026-11-06-aiggp-06-indirect-injection-provenance/) | [provenance-records](archive/2026-11-06-aiggp-06-indirect-injection-provenance/specs/provenance-records/spec.md) | [provenance-records](../specs/provenance-records/spec.md) |

The four archive package headers still cite
`openspec/changes/AIGGP-RETIREMENT-2026-10-02.md`. That file is absent from
this checkout, and no replacement retirement record was found. The dated
reference is recorded as unavailable; archived package files remain unchanged
to preserve their historical bytes.
