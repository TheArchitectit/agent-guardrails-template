# pi-mcp-sdk-development Specification

## Purpose

Defines the Pi package's development-only MCP SDK requirement and its optional runtime behavior.

## Requirements

### Requirement: Development checks have the MCP SDK available
The Pi package SHALL declare and lock a compatible MCP SDK in development dependencies, and SHALL NOT require that SDK as a production dependency for standalone use.

#### Scenario: clean development install
- **GIVEN** the committed Pi package manifest and lockfile
- **WHEN** a developer runs `npm ci` and the package test suite
- **THEN** SDK-present bridge tests can import the declared SDK subpaths

### Requirement: The MCP bridge remains optional at runtime
When the MCP SDK is absent from a production installation, the Pi extension SHALL load in standalone mode and SHALL report the bridge unavailable without throwing during package load.

#### Scenario: production install omits development dependencies
- **GIVEN** a packed production install contains no MCP SDK
- **WHEN** the extension loads its standalone tools and checks bridge availability
- **THEN** standalone loading succeeds and the bridge is reported unavailable
