# Backlog

Forward-looking work items for agent-guardrails-template that are agreed but
not yet scheduled into a sprint. One entry per item; move to a sprint doc under
`docs/sprints/` when picked up.

---

## BL-001 — Add OpenSpec official documentation links

**Added:** 2026-10-11 (owner request, Roger)
**Type:** docs
**Status:** open

### Why

This repo uses the OpenSpec framework for its spec corpus (`docs/specs/*`), but
the existing references are vague and carry no links to the official sources.
`docs/standards/api-specifications.md` §"OpenSpec Overview" even reads as a
placeholder — *"OpenSpec is a specification format that may offer different
capabilities… (Note: If referring to a specific OpenSpec project, details would
go here.)"* — and offers a "hypothetical" YAML example. Contributors and agents
have no canonical place to learn the framework's real conventions.

### What

Add the official OpenSpec documentation links to the repo's docs, and replace
the placeholder prose in `docs/standards/api-specifications.md` with a pointer
to the official sources instead of invented detail.

Official sources (verified live 2026-10-11):

- Project site / docs: https://openspec.dev
- Documentation: https://openspec.dev/docs
- Source (GitHub): https://github.com/Fission-AI/OpenSpec

### Acceptance criteria

- [ ] `docs/standards/api-specifications.md` "OpenSpec Overview" section links
      the official site, docs, and GitHub source; the "hypothetical" example and
      "details would go here" placeholder are removed or clearly marked as a
      local illustration.
- [ ] Whatever index file governs specs (e.g. `docs/specs/INDEX.md`) or the
      repo README gains a short "OpenSpec" reference pointing at the official
      docs.
- [ ] No invented OpenSpec versions or URLs — links must resolve.

### Notes

- `docs/standards/INDEX.md` already lists "OpenSpec overview and use cases" as a
  topic; wire it to the real links rather than duplicating.
- Keep it a docs-only change; no code or gate impact.
