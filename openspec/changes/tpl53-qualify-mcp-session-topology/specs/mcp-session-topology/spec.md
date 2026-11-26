# Spec Delta

## Purpose

Session state is process-local and the existing Kubernetes example requests two replicas. Adopters require a route and restart contract for each supported profile.

## ADDED Requirements

### Requirement: Deployment supports only validated MCP session topologies
Deployment preflight SHALL verify that configured replicas and routing match the supported session mode. Session renewal or resumption after replacement SHALL retain principal and resource isolation using the documented contract.

#### Scenario: Unsupported configuration
- **WHEN** a profile declares incompatible replica count, session state or routing
- **THEN** preflight fails with the named conflict before deployment

#### Scenario: Supported session restart
- **WHEN** a client reconnects after a process or serving instance is replaced
- **THEN** a process-local profile requires a fresh initialization and a supported resumable profile verifies current authority and session identity before resuming

#### Scenario: Expired revoked or mismatched session
- **WHEN** a session is expired, the current credential has been revoked, or the authenticated principal differs from the session owner
- **THEN** no serving instance accepts the session or exposes its data to that caller
