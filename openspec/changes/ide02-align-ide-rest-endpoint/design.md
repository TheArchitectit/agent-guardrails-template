# Design

## Context

The server config defaults `WEB_PORT` to 8081; its README identifies web UI, REST, health, and IDE endpoints on that listener. MCP transport defaults to 8080. IDE clients invoke REST routes. Existing clients expose a configurable server URL and must continue to honor it.

## Goals / Non-Goals

**Goals:**

- Align all supported IDE REST defaults and setup examples with the web listener.
- Preserve per-client explicit URL overrides and keep MCP transport distinct.

**Non-Goals:**

- Change server ports or transport behavior.
- Replace the MCP transport port with the REST port.

## Decisions

- Use `http://localhost:8081` as the REST default in VS Code, JetBrains, Vim, Neovim, and their setup examples.
- Keep each integration's explicit URL setting or variable authoritative when supplied.
- Add a static consistency test for all four clients plus a Go handler test proving the route is served. The static test must report each mismatch rather than pass on an empty file set.

## Risks / Trade-offs

- [One client retains a stale default] → enumerate all four clients and docs in a non-vacuous test.
- [Custom deployments lose their endpoint] → verify a non-default explicit URL still controls requests.

## Migration Plan

Existing saved overrides remain unchanged. Fresh configurations use port 8081. Run the contract, Go handler, and VS Code checks listed in `tasks.md`.
