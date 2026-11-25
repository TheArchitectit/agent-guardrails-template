# Design

## Context

The legacy templates set Go 1.21. Existing GitLab Go tests build the MCP module; Jenkins Go tests build MCP and its CLI build targets the team CLI module. Their directives are MCP 1.26.11 and CLI 1.23.2. The Go example module declares 1.25.0 but neither template currently has an example job.

## Decisions

- GitLab uses the official `golang:<mcp-server/go.mod version>` image, sets `GOTOOLCHAIN=local`, and asserts `go env GOVERSION` before the existing MCP test command. The exact image provides the toolchain; `local` disables automatic downloads.
- Jenkins selects each module directive with `GOTOOLCHAIN=go<version>+path` before its existing MCP test or CLI build, then asserts `go env GOVERSION`. The `+path` selector requires a preinstalled matching version and disables download fallback.
- Require future jobs to select the `go.mod` directive for the module they build; do not add an example Go job in this change.
- Keep existing test and build commands, race flags, runner controls, and artifact locations unchanged unless a toolchain selector requires a local path adjustment.
- Fail clearly when the requested version is unavailable or the selected version differs. GitLab uses its exact image and Jenkins uses a matching preinstalled toolchain; neither route downloads or silently falls back.
- Give the MCP race suite an explicit 30-minute test timeout. A bounded mutation-kill control took 17 minutes 15.7 seconds with Go 1.26.11 in the local Linux image, exceeding Go's default 10-minute per-test timeout. Preserve all package, race, and coverage flags while allowing this existing control to finish on slower runners.

## Compatibility and Rollback

GitLab uses the official image tag matching the MCP module directive. Jenkins agents must provide each required versioned Go executable on `PATH` or a bundled toolchain that matches the requested module version; setup fails with an explicit diagnostic when it is unavailable. Local Docker shell replays validate the selectors and commands, while acceptance on live GitLab/Jenkins runner platforms remains a separate deployment check. Preserve current tests, build steps, runner controls, and artifact locations.

## Verification

Run each relevant template job and record `go version` in the job context; verify MCP tests still use `-race` and CLI build artifacts remain unchanged.