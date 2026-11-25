# Tasks

## 1. Compose contract

- [x] 1.1 Point the MCP service build context at `./mcp-server` and its Dockerfile at `deploy/Dockerfile`; verify both paths exist relative to the build context and preserve pinned dependencies, local profile/attestation, loopback publications, and healthy database/cache ordering.
- [x] 1.2 Add the `local-docker-compose-pilot` capability spec with normative root-relative build, Windows preflight, credential boundary, local-only runtime, readiness, and safe teardown requirements; verify strict OpenSpec validation accepts its GIVEN/WHEN/THEN scenarios.

## 2. Windows laptop pilot runbook

- [x] 2.1 Add the Windows PowerShell runbook with Docker Desktop WSL 2 preflight, synthetic credentials written as UTF-8 without BOM to a user-local file outside build context, create-new/no-overwrite semantics, daemonless config validation, health/readiness checks, loopback verification, and `down` without volume deletion; verify the documented safety properties are present.
- [x] 2.2 Link the runbook from the repository index map and verify the link target exists.

## 3. Static validation

- [x] 3.1 Run `docker compose --env-file <temporary-outside-context-env> config --quiet` without starting services and static path/policy checks; record that Windows container startup and runtime health remain NOT_RUN.
- [ ] 3.2 Install Docker Desktop with the WSL 2 backend on the target Windows laptop, build and start the stack with synthetic data, verify all three service health checks and loopback-only readiness, then stop the pilot and record the result.
