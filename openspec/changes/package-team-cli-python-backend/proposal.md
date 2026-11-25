# Proposal

## Why

The team CLI install target and installer copy only the Go executable, but commands also invoke `scripts/team_manager.py`. The executable searches the source checkout and then falls back to a working-directory-relative path, so an installed binary used outside the checkout cannot reliably find its required backend.

## What Changes

- Install or otherwise locate the Python backend as part of the CLI distribution.
- Preserve `TEAM_MANAGER_PATH` as an explicit override.
- Verify installed commands from a directory outside the source checkout.

## Capabilities

### New Capabilities
- `team-cli-backend-distribution`: locating the team manager backend for installed CLI commands.

### Modified Capabilities
- None.

## Impact

`cmd/team-cli/Makefile`, `cmd/team-cli/install.sh`, CLI path resolution, and distribution layout. The behavior of source-checkout development workflows remains supported.
