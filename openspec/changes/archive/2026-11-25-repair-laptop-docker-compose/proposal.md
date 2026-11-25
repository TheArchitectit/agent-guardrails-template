# Proposal

## Why

The root Compose entry point points at the repository root even though its Dockerfile expects `mcp-server` as the build context. This blocks the Windows laptop pilot and leaves the existing local profile difficult to use safely without a verified runbook.

## What Changes

- Point the root stack at the MCP module context and its existing deployment Dockerfile while preserving pinned images, the explicit local deployment profile and attestation, loopback-only publications, and database/cache health ordering.
- Add a short Windows PowerShell and Docker Desktop pilot runbook with synthetic disposable credentials, a private env file outside the build context, preflight, readiness, and non-destructive teardown steps.
- Add only the documentation links needed to make the laptop path discoverable.

## Capabilities

### New Capabilities

- `local-docker-compose-pilot`: Defines the root Compose build path and safe, isolated Windows laptop pilot behavior.

### Modified Capabilities

None. The broader build-context hygiene finding remains tracked separately as TPL-33.

## Impact

Affects the root `docker-compose.yml`, deployment documentation and its index links. It does not change the existing deployment profile contract or claim a production-readiness/OPS closure. Container startup on Windows remains a pilot prerequisite and is not verified in this Linux worktree.
