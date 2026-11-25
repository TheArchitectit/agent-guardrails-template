# Tasks

## 1. Compose contract

- [x] 1.1 Point the MCP service build context at `./mcp-server` and its Dockerfile at `deploy/Dockerfile`; verify both paths exist relative to the build context and preserve pinned dependencies, local profile/attestation, loopback publications, and healthy database/cache ordering.
- [x] 1.2 Add the `local-docker-compose-pilot` capability spec with normative root-relative build, Windows preflight, credential boundary, local-only runtime, readiness, and safe teardown requirements; verify strict OpenSpec validation accepts its GIVEN/WHEN/THEN scenarios.

## 2. Windows laptop pilot runbook

- [x] 2.1 Add the Windows PowerShell runbook with Docker Desktop WSL 2 preflight, synthetic credentials written as UTF-8 without BOM to a user-local file outside build context, create-new/no-overwrite semantics, daemonless config validation, health/readiness checks, loopback verification, and `down` without volume deletion; verify the documented safety properties are present.
- [x] 2.2 Link the runbook from the repository index map and verify the link target exists.
- [x] 2.3 Add the standalone Windows PowerShell launcher that can download and verify the exact source archive, stage the pinned Compose inputs, reuse its own safe synthetic env file, bind explicitly to the local Docker Desktop Linux engine, validate effective configuration, and start the pilot with health-aware readiness checks.
- [x] 2.4 Document standalone use, pinned source behavior, safe restart, and target-laptop evidence capture.

## 3. Validation and runtime acceptance

- [x] 3.1 Run OpenSpec strict validation, daemonless Compose configuration, and PowerShell parser and mock launcher checks. Record mock results separately from target-laptop runtime.
- [x] 3.2 Run the pinned launcher on the target Windows laptop, verify local Docker Desktop Linux-engine selection, source archive verification, synthetic credential creation, dependency health, MCP image build, and loopback port bindings; record the initial startup failure and diagnosis in `evidence.md`.
- [ ] 3.3 Attach PostgreSQL, Redis, and MCP to the same intended pilot bridge in the pinned Compose inputs and add a one-shot migration job that completes the required schema version before MCP startup.
- [ ] 3.4 Fix readiness evaluation so startup dependency checks do not require the published lifecycle state to already be `READY`, while preserving HTTP 200 health/readiness checks; verify a clean launcher run has all three services healthy and readiness returns HTTP 200.
- [ ] 3.5 Stop and remove the pilot containers and network while preserving named data volumes, then record teardown verification. Leave this task open while the requested pilot stack remains running.
