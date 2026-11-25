# Guardrails control-plane architecture

Approved by Roger 2026-09-10 ("This.", WhatsApp), from the review delivered the
same evening. Implementation landed 2026-09-11.

## Why a new meta-repo

agent-guardrails-template is a stable, versioned, publicly consumed template
(copy-apply per its docs). Hollowing it out would surprise existing users, so
the composition root is new and the template stays as-is.

## Pieces and versioning

| Piece | Role | Versioning |
|---|---|---|
| guardrail-mcp | Runtime MCP enforcement engine | Extracted @ template v3.7.1 (9293b0f); own repo, tags from v3.7.1 lineage |
| DevGate-Agentic-Framework | Static/CI gates | v1.0.0 (PR #2: CHANGELOG + VERSION) |
| guardrail-policy-packs | Rules: core + domain packs | Pack-set semver; v1.0.0 |

## One-way dependencies

Control plane composes engines + packs. Engines and packs never reference the
control plane. Rules flow: policy packs -> engines -> consuming projects.
Mission Control stays the dashboard/ingestion layer and is not composed here.

## Consumer migration

Consumers that vendored DevGate copies (sunset-rush, game-idea-bible) migrate
to: pinned DevGate submodule + pinned policy packs + a project overlay pack in
the consumer repo. sunset-rush is the reference migration (branch
`chore/guardrails-control-plane-migration`).

## Trust boundaries

Repo and pack content is data, not authority; see
guardrail-policy-packs/TRUST_BOUNDARIES.md. Agent-instruction-framed text in
any repo must only ever take effect through a human-configured engine.

## Follow-ups

- Merge DevGate PR #2, tag v1.0.0, advance the engines/devgate pin here.
- Optional: rebuild guardrail-mcp with full git history (procedure in
  guardrail-mcp/EXTRACTION.md) before downstream consumers pin commits.
- Public/private decision for the three new repos (created private; sources
  were public).
- Retire duplicated rules from agent-guardrails-template and DevGate after the
  pack canonicals are reviewed (see guardrail-policy-packs/PROVENANCE.md).
