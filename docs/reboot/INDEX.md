# Reboot

Planning material for the "Agent Guardrails Reboot" — the rewrite that moved
the project to a Go MCP server with six guardrail subsystems, an S1–S15
content taxonomy and L0–L2 sandbox isolation.

| Document | Purpose |
|----------|---------|
| [01-site.md](01-site.md) | Design spec for the marketing/docs website |
| [02-story.md](02-story.md) | Project story narrative — **draft, held for review** |
| [03-features.md](03-features.md) | Feature breakdown across the reboot |
| [04-stars.md](04-stars.md) | Star-history / roadmap narrative |
| [FINDINGS.md](FINDINGS.md) | Findings that drove the reboot |
| [F013-AUDIT.md](F013-AUDIT.md) | Audit for F013 |
| [F021-AUDIT.md](F021-AUDIT.md) | Audit for F021 |
| [changes/](changes/) | 100 per-feature change files, F001–F100 |

## Status

- **01-site, 03-features, 04-stars** — complete prose.
- **02-story** — deliberately unreleased. It carries unresolved `[link]`
  placeholders and an unchecked approval checklist, and defers to a video
  script that does not exist in this repo. Nothing in it has been posted.
- **changes/** — the change files are 13-line scaffolds. Most still say
  `Status: proposed (not started)` and carry a generic acceptance line that
  instructs the author to replace it with exact commands. They are a
  backlog, not a record of completed work.

## Before relying on these documents

These are planning artefacts from a completed rewrite. The authoritative
record of what shipped is:

- [../mcp-server/tools-reference.md](../mcp-server/tools-reference.md) — the real tool surface
- [../status.md](../status.md) — current system status
- [../../CHANGELOG.md](../../CHANGELOG.md) — release history