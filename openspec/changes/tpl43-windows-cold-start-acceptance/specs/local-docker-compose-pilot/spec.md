# Spec Delta

## MODIFIED Requirements

### Requirement: Windows launcher is repeatable without replacing pilot data
<!-- id: tpl-43-windows-cold-start -->
The launcher SHALL create its synthetic env file with create-new semantics, validate and reuse its own matching env file on restart, and preserve all pilot volumes. It SHALL reject unexpected ambient Compose overrides and containers with conflicting ownership labels. Cold-start and restart acceptance SHALL run only on a separate authorized Windows host or VM using the launcher's fixed project name, with no credential values recorded.

#### Scenario: Launcher reruns with its own pilot state
- **GIVEN** the versioned source and valid synthetic env file created by this launcher already exist
- **WHEN** the user runs the launcher again
- **THEN** it reuses the existing source and credentials, does not print secret values, and leaves named volumes available

#### Scenario: Existing credentials or containers are not owned by the launcher
- **GIVEN** the dedicated env file is malformed or a fixed-name container belongs to a different Compose project
- **WHEN** the launcher checks local pilot state
- **THEN** it stops without replacing credentials or taking over the container

#### Scenario: cold isolated first-run creates pilot state
- **GIVEN** the separate authorized host has the standard Docker Desktop process stopped, its configured local `desktop-linux` engine unavailable, and no launcher's fixed-name project containers, env file, or named volumes
- **WHEN** the user invokes the parameterless pinned launcher
- **THEN** it starts Docker Desktop, confirms the local Linux engine, creates synthetic credentials with create-new semantics without printing them, starts only the fixed-name isolated project, and records readiness and source revision without storing credential values

#### Scenario: isolated restart reuses credentials and data
- **GIVEN** the isolated cold first-run completed and its launcher-owned env file and named volumes exist
- **WHEN** the documented volume-preserving Compose teardown runs and the user invokes the same pinned launcher again without arguments
- **THEN** the launcher reuses the validated credentials, preserves named volumes, and records a redacted continuity result

#### Scenario: isolation or secret handling cannot be verified
- **GIVEN** the separate host is not authorized, fixed-name state already exists, or secret-safe comparison cannot be performed
- **WHEN** acceptance is prepared or executed
- **THEN** the acceptance run stops before modifying any existing pilot state and remains unverified

### Requirement: Windows launcher targets only the local Docker Desktop Linux engine
<!-- id: tpl-43-windows-engine-startup -->
The launcher SHALL use an explicit `desktop-linux` context whose endpoint is Docker Desktop's local Windows Linux-engine named pipe, verify that the engine reports Linux, and start the installed Docker Desktop app when it is stopped. It SHALL NOT change Docker context or container mode. TPL-43 acceptance SHALL verify this behavior only on a separate authorized Windows host or VM.

#### Scenario: Docker Desktop is stopped
- **GIVEN** the `desktop-linux` context resolves to Docker Desktop's local Linux named pipe but the engine is not ready
- **WHEN** the launcher starts the pilot
- **THEN** it starts the standard Docker Desktop application and waits up to five minutes for the local Linux engine before Compose actions

#### Scenario: stopped Desktop process and unavailable local engine on acceptance host
- **GIVEN** a separate authorized Windows host or VM has the configured local `desktop-linux` context but Docker Desktop is stopped and the engine is unavailable
- **WHEN** the parameterless pinned launcher runs
- **THEN** it starts the standard Docker Desktop application, waits no more than five minutes, confirms the named-pipe endpoint resolves to a responsive Linux engine, and proceeds only against the launcher's fixed project name

#### Scenario: the selected engine is not the authorized local engine
- **GIVEN** the context endpoint or engine identity differs from the separate authorized host's local Docker Desktop Linux engine
- **WHEN** the launcher checks its target
- **THEN** it stops before running any Compose operation and the acceptance remains unverified
