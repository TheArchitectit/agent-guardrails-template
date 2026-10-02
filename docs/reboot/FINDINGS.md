# Reboot findings (Oct 1, 2026)

Recovery: branch `reboot/from-v3.7.1` from tag v3.7.1 (commit 67ae13dec309). 816 files at the tag. Main is a one-file retirement notice (c973296). No history rewritten.

## Verified here
- `go build ./...` in mcp-server: passes (Go 1.26.6 toolchain).
- `go test ./...`: all packages with tests pass after one fix (below). Run in a sandbox with no container runtime, so L2 sandbox paths ran their fallback branches only.
- site: `vitest` 4 tests pass, `tsc` clean, `vite build` OK, scrub scan clean, home page rendered and inspected at 1280px.

## Fixed
- sandbox: `TestSandboxManager_FailClosedNoDowngrade/InvalidMountPathRejected` failed. Cause: bind-mount path validation ran inside the L2 path, after the container-runtime lookup. With no runtime, execution fell back to L1 and the malformed path was never rejected. Fix: validate before any level runs (commit 6de9ddc). Real bug, not only a test issue.

## Open, not verified
- **MCP endpoint auth (found, fixed, not live-tested).** The web/API port applies APIKeyAuth (internal/web/server.go), but the `/mcp` endpoint is served by a separate StreamableHTTP listener with no auth, bound to 0.0.0.0. Config validated MCP_API_KEY yet never checked it on `/mcp`. Fix: `/mcp` is wrapped in a constant-time bearer check that fails closed on an empty key (internal/mcp/server.go, tests in internal/mcp/auth_test.go incl. a real initialize request: 401 without key, 200 with it). Still unverified against a live deployment with Postgres and Redis. Because it changes access behavior for existing clients, it needs Roger's verdict before merge.
- **Live server not run.** Needs Postgres and Redis (docker-compose). Not available in this sandbox. Run on the Dell fleet to verify `tools/list` and tool calls.
- **Copilot/VS Code behavior.** Advisory only; no hook test exists, so compat matrix is UNKNOWN for all hosts.
- gofmt reports many pre-existing unformatted files; left alone to keep the diff small.
- Site stack deviation: plain CSS variables instead of Tailwind (spec allowed either). No Framer Motion, MDX or Playwright yet.
