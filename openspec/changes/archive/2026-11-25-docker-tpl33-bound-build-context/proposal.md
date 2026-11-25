# Proposal

## Why

The documented Docker build uses `mcp-server` as context and the builder runs `COPY . .`. No context-root `.dockerignore` is present, so eligible local credentials, VCS state, caches, and test artifacts are sent and copied into an intermediate builder layer. The final image separately copies selected artifacts; this does not claim arbitrary files currently enter the runtime image.

## What Changes

- Add `mcp-server/.dockerignore` that excludes local-only files while retaining all build inputs.
- Verify exclusions and required paths with safe synthetic context fixtures, then run a representative image build without displaying fixture contents.

## Capabilities

### New Capabilities

- `mcp-docker-context`: Keep the MCP builder context limited to required source and build inputs.

### Modified Capabilities

None.

## Impact

Affects Docker build context and test coverage. Runtime image assembly remains unchanged. No implementation is included in this planning package.
