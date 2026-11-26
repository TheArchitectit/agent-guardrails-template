# Design

## Context

`make install` and `install.sh` place only the Go binary in the installation directory. `getTeamManagerPath` honors `TEAM_MANAGER_PATH`, probes paths near the executable, then returns `scripts/team_manager.py` relative to the current working directory.

## Decisions

- Define `PREFIX/share/agent-guardrails-template/team-cli/scripts` as the install-owned backend location. The executable checks the sibling `../share/agent-guardrails-template/team-cli/scripts/team_manager.py` path before the existing source-checkout and working-directory fallbacks.
- Package `team_manager.py`, its sibling `batch_operations.py` and `encryption.py` modules, and the migrations directory together. Resolve migration resources beside `team_manager.py`; keep `.teams` project data relative to the caller's working directory.
- Keep `TEAM_MANAGER_PATH` as the first-priority override.
- Make the Makefile installer and shell installer honor the same `PREFIX` and install the identical payload. Uninstall removes the CLI binary, alias, and dedicated package data directory without deleting sibling user files or an external `TEAM_MANAGER_PATH` override.
- Preserve source-tree lookup for development if it remains safe and documented.

## Compatibility and Rollback

Existing source checkouts and the explicit environment override continue to work. The installation layout is additive. Roll back by removing the added install payload and resolver path together; do not remove user-provided override files.

## Verification

Install into a temporary prefix, invoke CLI commands from an unrelated working directory, verify backend execution, verify `TEAM_MANAGER_PATH` override precedence, and verify uninstall removes the owned payload.
