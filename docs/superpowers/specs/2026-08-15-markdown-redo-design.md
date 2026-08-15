# Markdown Redo — Design Spec

**Date:** 2026-08-15
**Status:** Draft (awaiting review)
**Scope:** All tracked `.md` files except `examples/` language samples

---

## 1. Goal

A complete redo of the repository's 164 markdown files: consistent naming, a clean
directory taxonomy, no duplicates or contradictions, human-voiced prose under the
500-line limit, and a lean navigation system that doesn't break on every edit.

## 2. Locked Decisions (from brainstorming)

- **Depth:** structure + prose rewrite (all surviving files re-voiced, trimmed, facts reconciled).
- **Stale point-in-time docs:** move to `docs/archive/`, exclude from live nav.
- **Nav:** lean rebuild — one `INDEX_MAP.md` + one `INDEX.md` per dir; drop `HEADER_MAP.md` and `RULES_INDEX_MAP.md`; trim `TOC.md` to a flat listing.
- **3D game files:** all move to the private companion repo (honor the README's "moved to private repo" note).
- **`.openclaw/`:** remove as orphan (after its sole game-skill file moves to private repo).
- **pi-extension:** already restored to main (commit 9946f28) — out of scope for the redo except its README.
- **Reorg style:** clean reorg (new taxonomy; old links break; noted in CHANGELOG).

## 3. Naming Convention

- **`lowercase-kebab-case.md`** for all docs.
- Exceptions kept uppercase (tools/GitHub expect them): `README.md`, `CHANGELOG.md`, `CONTRIBUTING.md`, `LICENSE`, `CLAUDE.md`.
- All index files: `INDEX.md` (fixes current `INDEX.md`/`index.md` mix).
- Drop stale `**Version:**` / `**Last Updated:**` headers — version lives in CHANGELOG. Each file gets a one-line purpose under the H1 instead.

## 4. Target Taxonomy

```
docs/
├── getting-started/   quick-setup, how-to-apply, agent-guardrails (concepts), four-laws
├── architecture/      system architecture (rewritten, transport fixed)
├── mcp-server/        tools-reference, migration, python-to-go, api, observability, deployment-guide
├── integrations/      overview + claude-code, opencode, cursor, windsurf, copilot, agents-and-skills-setup
├── teams/             team-structure, team-tools (split if >500)
├── rules/             writing-rules, extracting-rules, rule-patterns
├── standards/         (kept, consolidated, renamed kebab)
├── workflows/         (kept, renamed kebab)
├── security/          (kept)
├── advisors/          (kept)
├── ai-dev/            (kept)
├── state/             (kept)
├── generative/        (kept)
├── monetization/      (kept)
├── multiplayer/       (kept)
├── analytics/         (kept)
├── deployment/        (kept)
├── ui-ux/             (kept)
├── accessibility/     (kept)
├── spatial/           (kept)
├── ethical/           (kept)
├── enterprise/        (kept, renamed kebab)
├── troubleshooting.md  (top-level, split to ≤500, wired into nav)
├── status.md
├── releases/          v3.3.0, v3.2.0, archive/
└── archive/           NEW — plans/, reviews/, gap-analyses/, sprints/, clean-cqrs/
```

Root: `README.md`, `CHANGELOG.md`, `CONTRIBUTING.md`, `CLAUDE.md`, `INDEX_MAP.md`, `TOC.md`.

## 5. File Disposition

### 5a. Move to private companion repo (game content — 7 files)
- `skills/shared-prompts/3d-game-dev.md`
- `.opencode/skills/3d-game-dev/` (SKILL.md)
- `.claude/skills-3d/3d-game-dev.json`
- `.cursor/rules-3d/3d-game-dev.md`
- `.openclaw/skills/3d-game-dev/SKILL.md` (then remove empty `.openclaw/`)
- `docs/standards/GAME_BUILD_VALIDATION.md` (unimplemented game-engine tool)
- `docs/vision-pipeline.md` — **AMBIGUOUS:** documents a real MCP server feature
  (env vars + code exist) but framed entirely around 3D game dev. Decision needed:
  (A) move to private repo as game content, or (B) keep and de-game the prose.
  Default if no answer: **B** (keep — it documents public server infra).

### 5b. Archive to `docs/archive/` (stale point-in-time — ~18 files)
- `docs/plans/MCP_SERVER_PLAN.md` (2093 lines, "Status: Planning" for a live server)
- `docs/plans/PROJECT_PLAN.md` (Project Sentinel, released)
- `docs/plans/GAP_REMEDIATION_BRANCH.md`
- `docs/reviews/AUDIT-REPORT.md` (April audit, claims Python still present)
- `docs/reviews/PLATFORM_REVIEW_2026-06-14.md`
- `docs/reviews/IMPLEMENTATION_REPORT_2026-06-14.md`
- `docs/GAP_ANALYSIS_TEAM_REPORT.md`
- `docs/TEAM_TOOLS_GAP_REMEDIATION_PLAN.md`
- `docs/ARCHITECTURE_CLEAN_CQRS.md` (references non-compiling `internal/adapters/`)
- `docs/sprints/*` (all 13 files — SPRINT_*, SPRINT_GUIDE, SPRINT_TEMPLATE, v1.x)
- `docs/releases/archive/*` (already archived — stays)

### 5c. Delete (duplicates / orphans / nav rebuild — ~6 files)
- `.guardrails/pre-work-check.md` OR `mcp-server/.guardrails/pre-work-check.md` (byte-identical; keep one)
- `docs/CONTRIBUTING.md` (fold into root `CONTRIBUTING.md`)
- `docs/OPCODE_INTEGRATION.md` (merged into integrations/opencode.md)
- `HEADER_MAP.md` (nav rebuild)
- `RULES_INDEX_MAP.md` (folded into INDEX_MAP)
- `.openclaw/` (after game file moved out)

### 5d. Merge (4 clusters → 4 files)
- OpenCode: `OPENCODE_INTEGRATION.md` + `OPCODE_INTEGRATION.md` → `integrations/opencode.md`
- Python migration: `PYTHON_TO_GO_MIGRATION.md` + `PYTHON_MIGRATION.md` → `mcp-server/python-to-go-migration.md`
- Contributing: root `CONTRIBUTING.md` + `docs/CONTRIBUTING.md` → root `CONTRIBUTING.md`
- Rules meta: `RULES_FROM_MD.md` + `RULE_PATTERNS_GUIDE.md` → `rules/` (split if >500)

### 5e. Keep + rewrite + rename (~120 files)
All remaining files: renamed to kebab, moved to target dir, prose re-voiced, trimmed
under 500 lines (split if over), facts reconciled. Oversized files needing real
decomposition: `TEAM_TOOLS.md` (1199), `TROUBLESHOOTING.md` (767), `MIGRATION.md`
(785), `API.md` (962), `PROMPTING_GUIDE.md` (984).

## 6. Prose & Length Standards

- Human voice, no AI-hype ("seamless", "powerful", "comprehensive", "leveraging").
  Match the tone of the v3.3.0 release notes.
- 500-line hard limit. Oversized files split into focused sub-documents with their
  own INDEX, not just trimmed.
- One-line purpose under every H1. No stale version/date headers.
- Reconcile all facts to the live server in one pass: **35 tools, 11 resources,
  Go 1.25+, BSD-3-Clause**. Fix the `.claude/skills/` count to 8.

## 7. Nav System (lean rebuild)

- Root `INDEX_MAP.md`: keyword → file, trimmed to current files only.
- One `INDEX.md` per content directory (purpose + file list).
- **Drop `HEADER_MAP.md`** (1033 lines of line-number refs that break on every edit).
- **Drop `RULES_INDEX_MAP.md`** (fold rule keywords into INDEX_MAP).
- `TOC.md`: flat file listing, no annotations.
- Wire current orphans into nav: `troubleshooting.md`, `architecture/`, the
  integration docs.

## 8. Execution Phases (commit per phase)

1. **Game-content removal** — move 7 game files out (note: transfer to private repo is
   manual; here we delete from public + record in CHANGELOG). Remove `.openclaw/`.
2. **Archive** — move ~18 stale docs to `docs/archive/` subdirs.
3. **Deletions & merges** — delete duplicates, merge the 4 clusters, drop nav files.
4. **Restructure + rename** — move files to target taxonomy, rename to kebab, fix
   internal links.
5. **Prose rewrite** — re-voice survivors, split oversized files, reconcile facts.
6. **Nav rebuild** — lean INDEX_MAP + per-dir INDEX, trim TOC.
7. **Verify** — dead-link sweep, count/version consistency, 500-line audit, gitleaks.

## 9. Verification Criteria

- `find docs -name '*.md' | xargs wc -l` — no file over 500 lines (CHANGELOG exempt).
- Dead-link sweep across all `.md` — zero broken internal links.
- `git grep` for stale counts (5/6/8/14/27/30 tools, Go 1.21/1.23 as current) — zero.
- gitleaks — clean (allowlist in place).
- Every `docs/` subdir has an `INDEX.md`; root has `INDEX_MAP.md` + `TOC.md` only.

## 10. Open Decisions Within Scope

1. `vision-pipeline.md` — move to private repo (game) or keep + de-game? Default: keep.
2. `GAME_BUILD_VALIDATION.md` — private repo vs archive as unimplemented? Default: private repo (it's game content).
3. `.teams/` runtime artifacts — stay gitignored/uncommitted throughout (as before).
