# Qualify CI environment parity and the built MCP image with Codex

## Why

The repository has a local validation entry point and GitHub workflows that can target either a hosted runner or an administrator-selected runner pool. Static workflow tests cannot prove how GitHub evaluates the hosted default when `CI_RUNNER_LABELS` is absent. The Docker client transport fixture only proves filtered build-context transmission; it explicitly does not build the production MCP image or exercise a real Codex MCP client against it.

## What changes

- Define and verify a check-by-check parity contract for local execution, GitHub-hosted execution, and the registered UCS03 self-hosted fleet without weakening any existing gate.
- Qualify selector behavior with the variable absent and configured, and preserve fork isolation for private runner pools.
- Build the real MCP server image and exercise it through an actual Codex MCP client, including initialization, tool listing, and a benign tool call.
- Repeat synthetic context and isolation checks on AI03, AI04, and AI05 using disposable fixtures and no host Docker socket.
- Document the workflow triggers, responsibility, runner selection, local equivalents, and evidence that remains GitHub-only.

## Capabilities

### New Capabilities

- `ci-environment-parity`: comparable, auditable deterministic gates across local, GitHub-hosted, and registered self-hosted environments.
- `mcp-image-codex-client-acceptance`: real built-image protocol and client acceptance with reproducible, redacted evidence.

### Modified Capabilities

None.

## Impact

This proposal covers `.github/workflows/`, `scripts/run_local_ci.sh`, `scripts/run_local_event_gates.sh`, the MCP Docker build and client test harness, and `docs/ci/`. Existing protected-branch, fork, image/context, secret, and test-discovery controls remain required. The self-hosted fleet is not a trusted Docker-daemon substitute; synthetic isolation qualification must not mount a host socket or alter runner configuration.
