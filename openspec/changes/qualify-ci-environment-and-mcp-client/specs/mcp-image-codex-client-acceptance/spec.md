# Spec Delta

## Purpose

The Docker CLI context transport fixture verifies which files the client sends. It does not prove that the production MCP Dockerfile builds a runnable image or that Codex can speak MCP to that image.

## ADDED Requirements

### Requirement: Production MCP image is built and inspected as an image

Qualification SHALL build the production `mcp-server/deploy/Dockerfile` from a pinned source commit with a unique disposable tag and record the resulting image digest, source revision, and engine/tool versions. Context-transport success alone SHALL NOT satisfy this requirement.

#### Scenario: Reproducible image build

- **WHEN** the production Dockerfile is built in an approved disposable engine
- **THEN** the build succeeds from the reviewed source revision and the receipt records the immutable image digest and build inputs

#### Scenario: Build failure or unavailable engine

- **WHEN** a supported environment cannot build the image or lacks an explicitly required engine
- **THEN** the result is reported as failed or unavailable with the concrete prerequisite, never as an image-build pass

### Requirement: Actual Codex client validates MCP protocol behavior against the built image

The built MCP image SHALL be exercised by an actual Codex MCP client over the supported transport. Acceptance SHALL verify successful MCP initialization, tool enumeration, and one benign tool invocation, and SHALL retain a redacted receipt tied to the exact image digest and client version.

#### Scenario: Initialize, list, and call

- **WHEN** Codex connects to the built image using the supported MCP configuration
- **THEN** initialization negotiates a supported protocol version, `tools/list` returns the expected tool inventory, and one non-mutating `tools/call` returns its expected bounded result

#### Scenario: Client or protocol incompatibility

- **WHEN** initialization, tool listing, or the selected call fails or returns an incompatible result
- **THEN** acceptance fails with the failing stage and client/image identities recorded; a transport-only test does not override the failure

#### Scenario: Redacted evidence

- **WHEN** an acceptance receipt is stored
- **THEN** it includes source commit, image digest, Codex version, protocol version, tool name, result summary, environment, and exit status without credentials or secret-bearing transcript content
