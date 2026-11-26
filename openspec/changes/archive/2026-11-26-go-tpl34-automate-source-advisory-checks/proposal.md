# Proposal

## Why

QA-05 records a one-time Go source-advisory remediation and scan. `mcp-server/Makefile` exposes a manual `govulncheck` target, but no workflow invokes it, so future source-reachable advisories can enter without recurring CI review.

## What Changes

- Add a PR and main-push advisory job using the toolchain in `mcp-server/go.mod` and a pinned scanner version.
- Fail on source-call-path findings and incomplete scans; report imported-only and module-only findings separately.
- Test classification with sanitized scan-output fixtures and document local invocation.

## Capabilities

### New Capabilities

- `go-advisory-automation`: Continuously classify and report Go advisories without conflating module presence with source reachability.

### Modified Capabilities

None.

## Impact

Affects the MCP Go CI workflow, advisory result handling, tests, and local documentation. This is separate from completed QA-05 remediation and does not claim the current dependency graph is advisory-free.
