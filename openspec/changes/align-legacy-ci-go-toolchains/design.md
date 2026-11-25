# Design

## Context

The legacy templates set Go 1.21 globally. Existing GitLab Go tests build the MCP module; Jenkins Go tests build MCP and its CLI build targets the team CLI module. Their directives are MCP 1.26.11 and CLI 1.23.2. The Go example module declares 1.25.0 but neither template currently has an example job.

## Decisions

- Set up the MCP test job with the exact version declared by `mcp-server/go.mod`, using an explicit supported installer or image.
- Set up the existing Jenkins CLI build from `cmd/team-cli/go.mod`, rather than inheriting the MCP version or relying on auto-download.
- Require future jobs to select the `go.mod` directive for the module they build; do not add an example Go job in this change.
- Keep existing test and build commands, race flags, runner controls, and artifact locations unchanged unless a toolchain selector requires a local path adjustment.
- Fail clearly when the requested toolchain cannot be installed; do not silently fall back to the runner's Go version.

## Compatibility and Rollback

Older legacy runners may need network access to fetch the specified toolchain or a pre-provisioned matching image. Keep the current job semantics while adding setup. If a runner cannot install the required version, pin a compatible image/toolchain mirror and verify it before reverting to any older Go version.

## Verification

Run each relevant template job and record `go version` in the job context; verify MCP tests still use `-race` and CLI build artifacts remain unchanged.
