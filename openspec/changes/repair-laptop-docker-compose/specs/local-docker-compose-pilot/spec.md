# Spec Delta

## Purpose

Defines the root Compose build contract and a safe, isolated Windows laptop workflow for piloting the local MCP stack with disposable credentials and data.

## ADDED Requirements

### Requirement: Root Compose resolves the MCP build from the module directory
The root Compose file SHALL resolve the MCP image build context from `mcp-server` relative to the repository root and resolve its Dockerfile relative to that context.

#### Scenario: Compose configuration is evaluated from the repository root
- **GIVEN** the user is in the repository root
- **WHEN** the root Compose configuration is resolved
- **THEN** the MCP build context is `./mcp-server` and the Dockerfile path is `deploy/Dockerfile` within that context

### Requirement: Laptop pilot requires a ready local Docker environment
The Windows laptop pilot instructions SHALL require Docker Desktop with the WSL 2 Linux container backend and a responsive Docker Engine before stack startup.

#### Scenario: Preflight runs before startup
- **GIVEN** a user is preparing the laptop pilot on Windows
- **WHEN** Docker CLI and Compose v2 preflight checks run
- **THEN** both client and server availability and a Compose v2 version are confirmed before any service is started

### Requirement: Pilot credentials stay synthetic and outside the build context
The pilot SHALL generate disposable credentials without printing them, store them in a user-local env file outside the repository build context, and refuse to overwrite an existing env file.

#### Scenario: New private env file is created
- **GIVEN** the selected user-local env file does not exist
- **WHEN** the pilot credential generation step runs
- **THEN** it creates a UTF-8 file without a byte order mark outside the repository and emits no credential values

#### Scenario: Existing env file is preserved
- **GIVEN** the selected user-local env file already exists
- **WHEN** the credential generation step runs
- **THEN** it stops without changing that file or printing credentials

### Requirement: Local pilot network publications remain loopback-only
The laptop pilot SHALL set the local deployment profile and isolated local bridge attestation and publish MCP and Web ports only on host loopback.

#### Scenario: Pilot stack starts with local-only host bindings
- **GIVEN** Compose starts the pilot with the generated env file
- **WHEN** the MCP service is created
- **THEN** its deployment profile is `local`, its local bridge attestation is explicit, and both host port publications bind to `127.0.0.1`

### Requirement: Application startup waits for healthy dependencies
The laptop pilot SHALL start the MCP service only after PostgreSQL and Redis report healthy status and SHALL provide an application readiness check.

#### Scenario: Dependencies become ready before the application
- **GIVEN** PostgreSQL or Redis is still unhealthy
- **WHEN** Compose evaluates the MCP service startup dependency
- **THEN** the MCP service is not started until both dependencies are healthy

#### Scenario: Application readiness is checked
- **GIVEN** the services have started
- **WHEN** the pilot readiness check requests `http://127.0.0.1:8081/health/ready`
- **THEN** readiness succeeds only when the endpoint returns HTTP 200

### Requirement: Pilot teardown preserves named data volumes
The default laptop pilot teardown SHALL stop and remove its containers and network without deleting named Postgres or Redis volumes.

#### Scenario: Default teardown retains disposable data
- **GIVEN** the laptop pilot is running with named data volumes
- **WHEN** the documented default teardown command runs
- **THEN** containers and network stop while both volumes remain available for a later restart

### Requirement: Standalone Windows launcher uses a pinned source archive
The Windows pilot SHALL be runnable from a single PowerShell script without requiring Git or an existing repository checkout. It SHALL download the exact approved source revision, verify the expected archive SHA-256 before extraction, and install only the Compose project inputs under a user-local versioned directory.

#### Scenario: Standalone source bootstrap
- **GIVEN** the user has PowerShell, Docker Desktop, and network access to the official GitHub archive host but no repository checkout or Git installation
- **WHEN** the standalone launcher runs
- **THEN** it downloads the archive for revision `ba9ff630b24821507d6c93c6c6117ac7be4e9b1e`, verifies SHA-256 `4f09f073c0e0c3c2706365c4444a752820efb7c44178b777b1d4a1c81f2ffa` before extraction, and installs the Compose file and MCP build context under `%LOCALAPPDATA%\GuardrailsLaptopPilot`

### Requirement: Windows launcher is repeatable without replacing pilot data
The launcher SHALL create its synthetic env file with create-new semantics, validate and reuse its own matching env file on restart, and preserve all pilot volumes. It SHALL reject unexpected ambient Compose overrides and containers with conflicting ownership labels.

#### Scenario: Launcher reruns with its own pilot state
- **GIVEN** the versioned source and valid synthetic env file created by this launcher already exist
- **WHEN** the user runs the launcher again
- **THEN** it reuses the existing source and credentials, does not print secret values, and leaves named volumes available

#### Scenario: Existing credentials or containers are not owned by the launcher
- **GIVEN** the dedicated env file is malformed or a fixed-name container belongs to a different Compose project
- **WHEN** the launcher checks local pilot state
- **THEN** it stops without replacing credentials or taking over the container

### Requirement: Windows launcher targets only the local Docker Desktop Linux engine
The launcher SHALL use an explicit `desktop-linux` context whose endpoint is Docker Desktop's local Windows Linux-engine named pipe, verify that the engine reports Linux, and start the installed Docker Desktop app when it is stopped. It SHALL not change Docker context or container mode.

#### Scenario: Docker Desktop is stopped
- **GIVEN** the `desktop-linux` context resolves to Docker Desktop's local Linux named pipe but the engine is not ready
- **WHEN** the launcher starts the pilot
- **THEN** it starts the standard Docker Desktop application and waits up to five minutes for the local Linux engine before Compose actions

### Requirement: Launcher checks effective local policy before startup
Before container startup the launcher SHALL render and inspect Compose configuration without revealing credentials, then verify source revision, local profile and attestation, credential wiring, and IPv4 loopback-only port publications. After startup it SHALL check the active port bindings and application readiness.

#### Scenario: Rendered configuration matches pilot policy
- **GIVEN** ambient variables cannot override the selected synthetic env file
- **WHEN** the launcher validates Compose configuration
- **THEN** it confirms the expected build revision, synthetic credential sources, local deployment profile, local bridge attestation, and ports 8080 and 8081 on `127.0.0.1` before `up`

#### Scenario: Started pilot is ready on loopback
- **GIVEN** Compose health-aware startup succeeds
- **WHEN** the launcher checks the active bindings and readiness endpoint
- **THEN** both services ports map to IPv4 loopback and `http://127.0.0.1:8081/health/ready` returns HTTP 200
