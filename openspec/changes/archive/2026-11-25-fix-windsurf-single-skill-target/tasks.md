# Tasks

## 1. Resolve file destinations

- [x] 1.1 Update registry/target resolution to represent explicit file destinations and add a temporary-root test that `windsurf-rules` creates a regular `.windsurfrules` file at the root.
- [x] 1.2 Add regression coverage for representative directory-based skill destinations and verify their existing relative paths are preserved.

## 2. Verify installer flows

- [x] 2.1 Run focused single-skill and bulk platform installer tests and verify both flows retain their expected output paths.

## Verification evidence

- `pytest -q tests/test_setup_agents.py` in `aiggp-pr41-lead:20261125`: 7 passed (2 parametrized subtests).
- Confirmed individual Windsurf install creates a regular `<target>/.windsurfrules` file and a directory-based Claude skill retains `<target>/.claude/skills/...`.
- Confirmed bulk Windsurf install still creates `<target>/.windsurfrules`.
- Directory versus file semantics are explicit in the registry's trailing-separator convention; the full `.windsurfrules` destination remains a leaf.
- `openspec validate fix-windsurf-single-skill-target --strict`: valid.
- `python scripts/regression_check.py --all` in the pinned image: no potential regressions detected.
- Independent replay on the laptop passed all seven installer tests and two
  subtests in a disposable Linux checkout. The regression scan passed.
- The command-line single-skill copy produced the root rules file. The bulk
  command created a symlink and succeeded when run again on that target.
