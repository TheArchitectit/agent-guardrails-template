# Design

## Context

`make install` and `install.sh` place only the Go binary in the installation directory. `getTeamManagerPath` honors `TEAM_MANAGER_PATH`, probes paths near the executable, then returns `scripts/team_manager.py` relative to the current working directory.

## Decisions

- Define a stable install-owned data location for `team_manager.py` and make the installed executable resolve that location relative to its own installation, independent of the caller's working directory.
- Keep `TEAM_MANAGER_PATH` as the first-priority override.
- Ensure both the Makefile install and shell installer copy the backend and any runtime files it requires, and make uninstall remove only files owned by this package.
- Preserve source-tree lookup for development if it remains safe and documented.

## Compatibility and Rollback

Existing source checkouts and the explicit environment override continue to work. The installation layout is additive. Roll back by removing the added install payload and resolver path together; do not remove user-provided override files.

## Verification

Install into a temporary prefix, invoke CLI commands from an unrelated working directory, verify backend execution, verify `TEAM_MANAGER_PATH` override precedence, and verify uninstall removes the owned payload.
