# Tasks

## 1. Normalize and constrain paths

- [x] 1.1 Change target normalization to remove only a literal `./` prefix and add a temporary-root test proving `.claude` remains in the installed path.
- [x] 1.2 Resolve and constrain destination paths to the configured root and add traversal test verifying no outside write occurs.

## 2. Verify installer compatibility

- [x] 2.1 Run the focused installer test suite and verify existing non-hidden platform destinations still resolve as before.

## Verification evidence

- `pytest -q tests/test_setup_agents.py` in `aiggp-pr41-lead:20261125`: 7 passed (2 parametrized subtests).
- Covered hidden `.claude` preservation with and without `./`, parent traversal rejection without an outside write, and symlink escape rejection for both clone and named-skill installs.
- Repeated named and bulk symlink installs against temporary roots; both report existing links as occupied without rejecting the source-facing leaf link.
- `openspec validate preserve-hidden-agent-install-paths --strict`: valid.
- `python scripts/regression_check.py --all` in the pinned image: no potential regressions detected.
- Independent replay on the laptop passed all seven installer tests and two
  subtests in a disposable Linux checkout. The regression scan passed.
- The command-line copy install also produced the expected hidden
  `.claude/skills/guardrails-enforcer.json` destination.
