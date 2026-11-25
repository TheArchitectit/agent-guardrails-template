# legacy-ci-go-toolchain-selection Specification

## Purpose

Defines explicit Go toolchain selection for legacy CI jobs that build repository Go modules.

## Requirements

### Requirement: Legacy CI selects each module's declared Go toolchain

Legacy CI Go jobs SHALL select an explicit toolchain compatible with the `go` directive of the module they build.

#### Scenario: run MCP module tests

- **WHEN** a legacy CI job runs Go tests from `mcp-server`
- **THEN** it installs or selects the toolchain declared by `mcp-server/go.mod` before invoking the existing test command

#### Scenario: build the existing CLI module

- **WHEN** Jenkins runs its existing team CLI build
- **THEN** it selects a toolchain compatible with `cmd/team-cli/go.mod` and does not assume that an example module job exists

#### Scenario: add a future module job

- **WHEN** a future legacy CI job builds a different Go module
- **THEN** its setup selects a toolchain compatible with that module's own `go.mod` directive

### Requirement: Legacy toolchain setup does not depend on implicit downloads

Legacy CI jobs SHALL make toolchain selection explicit before build or test execution.

#### Scenario: runner starts with an older default Go version

- **WHEN** the runner's preinstalled Go version differs from the module's declared version
- **THEN** the job installs or selects the required toolchain explicitly and reports a clear setup failure if it is unavailable
