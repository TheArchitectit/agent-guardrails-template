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
