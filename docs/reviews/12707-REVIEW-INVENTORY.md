# Agent 12707 — Review or Clean Inventory

**Date:** 2026-11-26
**Status:** review-only preservation branches, not accepted changes

## Canonical refs

These conclusions use the `review-12707/*-snapshot` branches. Any earlier
`review/12707/*` short-name refs created during this operation were a failed
publishing attempt and must not be treated as evidence-bearing snapshots.

Canonical names for agent 12707:

- `review-12707/legacy-main-snapshot` = f43b139 (223 ahead / 574 behind origin/main)
- `review-12707/forge-rescue-snapshot` = 7c7d54d (224 ahead / 574 behind)
- `review-12707/feature-3d-game-development-snapshot` = 2acb0eb (171 ahead / 576 behind)
- `review-12707/feature-3d-game-development-rescue-snapshot` = d35afb0 (318 ahead / 567 behind)
- `review-12707/team-manager-snapshot` = dddc796 (144 ahead / 576 behind)
- `review-12707/team-manager-rescue-snapshot` = f09b88b (318 ahead / 567 behind)
- `review-12707/pr3-snapshot` = 4acd891 (145 ahead / 576 behind)
- `review-12707/pr3-rescue-snapshot` = f789677 (318 ahead / 567 behind)

These counts were computed against origin/main at 20d27af on 2026-11-26. A
subsequent origin/main run may move the behind count; the snapshot commit IDs
are the binding reference. Multiple merge bases are present, so `A...B` diff
stats alone are an unsafe representation of the total content.

## Review order and assessment

### 1. team-manager / pr3: most likely useful feature work

`team-manager` contains a large Go port (`scripts/team_manager.py`, 3507 added
lines; `mcp-server/internal/mcp/tools_extended.go`, 2267 added lines), the MCP
team tool handlers, tests, docs, OpenAPI, UI components, and an OpenAPI/Swagger
control plane. `pr3` is the same history plus a targeted SSE keepalive / HTTP
fallback fix. Evaluate these together before importing anything else.

### 2. legacy-main / forge-rescue: assess piece by piece

The top commit is f43b139 (“release: fix release date to 2026-06-16”), whose
preceding history includes security fixes, lifecycle state machine, token
budget, CI policy check, OpenAPI, and a large set of “v3.2.0” additions. The
forge-rescue commit captures a historical 181-commit unpushed batch on top of
that state. Treat the entire slate as *candidate features*, not one accepted
rollup. Likely value is present; provenance is too broad to fast-forward.

### 3. 3D game-development: mostly docs, promotional / instructional

2acb0eb adds a ~45k-word AI-Powered Development 2026 guide and 3D game guardrails.
The immediate commits concern referral-link formatting and game-development
guide prose, not runtime engine changes. Review for whether this instructive
material is desired in the project at all.

### 4. *-rescue snapshots: preservation, not additive features

The three rescue snapshots are file-upload snapshots whose trees substantially
overlap legacy-main and their corresponding feature line. Do not evaluate them
as more features. Verify a unique delta before retaining anything from a rescue
snapshot that is not already represented in team-manager, pr3, or legacy-main.
The forge-rescue top commit describes 181 unpushed commits; the three feature
rescues each describe a machinery snapshot, not a new conceptual function.

## Obvious cleanup candidates on legacy-main — remove before promotion

All ELF header verified **executable binaries** (`7f454c46`):

- `mcp-server/server`: 24,004,894 bytes.
- `mcp-server/bin/guardrail-mcp`: 15,307,044 bytes.
- `cmd/team-cli/team`: 7,293,012 bytes.
- `mcp-server/test_tools`: 2,255,408 bytes.

Other artifacts:

- `mcp-server/guardrail-mcp-v1.11.3.tar.gz`: 6,395,275 bytes release archive.
- `test_export.csv`: generated export fixture (50 added lines).
- `test_export.json`: 688 added lines of generated fixture data.
- The legacy-main `.gitignore` ignores both `mcp-server/server` and
  `mcp-server/bin/guardrail-mcp` — both are nevertheless already tracked and
  present in the snapshot. Them being ignored does not undo the history leak.
- The same `.gitignore` ignores `mcp-server/internal/mcp/*_test.go`. New tracked
  tests already exist; do not make that ignore rule the way to “clean up” tests
  that should run.

## What agent 12707 must actually decide

For each useful candidate: whether its implementation belongs in a fresh
feature branch cut from current origin/main, with a separately written scoped
commit and verified gate evidence. The preserved snapshots serve as the source
of truth; they are not merge candidates in this shape.

1. **Keep/drop team manager Go port** — actual compile, behaviour, and test
   comparison versus current main.
2. **Keep/drop pr3's SSE fix** — independently reproduce the timeout assertion
   and verify the HTTP fallback instead of relying on the commit message.
3. **Keep/drop v3.2.0 feature slices** — individually, not “all seven features”.
4. **Decide fate of referral / 3D-guide content** by editorial preference.
5. **If a rescue snapshot is retained, record why** — identical overlap is not
   a unique feature.
6. **Before pushing anything to origin/main**: verify no ELF/binary/tarball or
   generated fixture is staged; ensure the same tests run that would have
   passed on the snapshot.

## Safety / proof notes

- Nothing in this branch-documentation pass has merged into origin/main.
- Exact snapshot commits were preserved without rewriting their contents or
  authors. Do not rebase the snapshots; copy desired content out in scoped
  commits against the real current main.
- Huge binary / test-artifact cleanup is a precondition before any branch
  promotion, not a side-effect of a merge.
