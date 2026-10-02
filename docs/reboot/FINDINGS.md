# Reboot findings (Oct 1, 2026)

Recovery: branch `reboot/from-v3.7.1` from tag v3.7.1 (commit 67ae13dec309). 816 files at the tag. Main is a one-file retirement notice (c973296). No history rewritten.

## Verified here
- `go build ./...` in mcp-server: passes (Go 1.26.6 toolchain).
- `go test ./...`: all packages with tests pass after one fix (below). Run in a sandbox with no container runtime, so L2 sandbox paths ran their fallback branches only.
- site: `vitest` 4 tests pass, `tsc` clean, `vite build` OK, scrub scan clean, home page rendered and inspected at 1280px.

## Fixed
- sandbox: `TestSandboxManager_FailClosedNoDowngrade/InvalidMountPathRejected` failed. Cause: bind-mount path validation ran inside the L2 path, after the container-runtime lookup. With no runtime, execution fell back to L1 and the malformed path was never rejected. Fix: validate before any level runs (commit 6de9ddc). Real bug, not only a test issue.

## Open, not verified
- **MCP endpoint auth.** Config requires MCP_API_KEY, but `cmd/server/main.go` and `internal/mcp/server.go` show no auth applied to `/mcp`, and the server binds 0.0.0.0. Not confirmed against a live server. Treat as unauthenticated: localhost only. Needs an OpenSpec change.
- **Live server not run.** Needs Postgres and Redis (docker-compose). Not available in this sandbox. Run on the Dell fleet to verify `tools/list` and tool calls.
- **Copilot/VS Code behavior.** Advisory only; no hook test exists, so compat matrix is UNKNOWN for all hosts.
- gofmt reports many pre-existing unformatted files; left alone to keep the diff small.
- Site stack deviation: plain CSS variables instead of Tailwind (spec allowed either). No Framer Motion, MDX or Playwright yet.
