# ide-client-initialization Specification

## Purpose

Defines how the VS Code extension loads configuration and SecretStorage before making its initial network request.

## Requirements

### Requirement: Initialize VS Code configuration before requests
The VS Code extension SHALL initialize its client from workspace settings and extension SecretStorage before the first connection check or validation request, and SHALL use those values for the request.

#### Scenario: activation uses configured URL and secret
- **GIVEN** workspace configuration contains a non-default server URL and SecretStorage contains an API key
- **WHEN** activation performs its initial readiness check
- **THEN** the request uses the configured URL and bearer key, and a successful response marks the extension connected

#### Scenario: initial connection fails
- **GIVEN** the activation request has initialized configuration but the stubbed server is unreachable
- **WHEN** the connection check completes
- **THEN** the extension reports disconnected and remains activated without an uninitialized-config error
