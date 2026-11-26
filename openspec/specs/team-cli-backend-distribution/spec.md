# team-cli-backend-distribution Specification

Status: CLOSED (EXERCISED 2)

## Purpose

Defines how installed team CLI distributions locate their required Python backend.

## Requirements

### Requirement: Installed CLI locates its packaged backend

An installed team CLI SHALL locate an installation-owned `team_manager.py` backend independently of the caller's current working directory.

#### Scenario: invoke an installed command outside the checkout

- **WHEN** a user runs an installed team CLI command from a directory outside the source checkout
- **THEN** the CLI invokes the backend installed with the CLI without requiring a working-directory-relative source path

### Requirement: Explicit backend override is preserved

The team CLI SHALL honor `TEAM_MANAGER_PATH` as an explicit backend path override.

#### Scenario: provide a backend override

- **WHEN** `TEAM_MANAGER_PATH` names an available backend
- **THEN** the CLI uses that backend before its installation-owned default
