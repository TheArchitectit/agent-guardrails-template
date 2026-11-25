# Tasks

## 1. Compose contract

- [x] 1.1 Point the MCP service build context at `./mcp-server` and its Dockerfile at `deploy/Dockerfile`; verify both paths exist relative to the build context and preserve pinned dependencies, local profile/attestation, loopback publications, and healthy database/cache ordering.
- [x] 1.2 Add the `local-docker-compose-pilot` capability spec with normative root-relative build, Windows preflight, credential boundary, local-only runtime, readiness, and safe teardown requirements; verify strict OpenSpec validation accepts its GIVEN/WHEN/THEN scenarios.

## 2. Windows laptop pilot runbook

- [x] 2.1 Add the Windows PowerShell runbook with Docker Desktop WSL 2 preflight, synthetic credentials written as UTF-8 without BOM to a user-local file outside build context, create-new/no-overwrite semantics, daemonless config validation, health/readiness checks, loopback verification, and `down` without volume deletion; verify the documented safety properties are present.
- [x] 2.2 Link the runbook from the repository index map and verify the link target exists.
- [x] 2.3 Add the standalone Windows PowerShell launcher that can download and verify the exact source archive, stage the pinned Compose inputs, reuse its own safe synthetic env file, bind explicitly to the local Docker Desktop Linux engine, validate effective configuration, and start the pilot with health-aware readiness checks.
- [x] 2.4 Document standalone use, pinned source behavior, safe restart, and the pending target-laptop evidence step.

## 3. Static validation

- [x] 3.1 Run OpenSpec strict validation, daemonless Compose configuration, and PowerShell parser and mock launcher checks. Record mock results separately from target-laptop runtime.
- [ ] 3.2 Confirm the installed Docker Desktop uses the local Linux engine on the target Windows laptop, build and start the stack with synthetic data, verify all three service health checks and loopback-only readiness, then stop the pilot and record the result.
