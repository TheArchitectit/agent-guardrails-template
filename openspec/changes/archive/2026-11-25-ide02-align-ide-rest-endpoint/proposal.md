# Proposal

## Why

VS Code, JetBrains, Vim, and Neovim default to `http://localhost:8095`, while the Go server REST and web listener defaults to port 8081; MCP transport uses the separate port 8080. Fresh IDE installations target an unserved REST endpoint.

## What Changes

- Set the four IDE REST defaults and setup examples to `http://localhost:8081`.
- Preserve explicit user endpoint overrides and the separate MCP transport port.
- Add a cross-client consistency check and a server REST handler test.

## Capabilities

### New Capabilities

- `ide-rest-endpoint`: Keep IDE REST defaults aligned with the server while preserving overrides.

### Modified Capabilities

None.

## Impact

Affects VS Code, JetBrains, Vim, Neovim setup documentation, and REST contract tests. No user-configured endpoint is removed. No implementation is included in this planning package.
