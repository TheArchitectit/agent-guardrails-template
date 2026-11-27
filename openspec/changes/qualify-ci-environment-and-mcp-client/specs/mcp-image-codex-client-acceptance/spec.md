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

### Requirement: Candidate builds run deterministic MCP acceptance without model usage

The build workflow SHALL associate deterministic real-image MCP protocol acceptance with the exact candidate source and image digest. The workflow SHALL show whether acceptance passed, failed, was skipped, or could not run, and SHALL NOT substitute a source-only or context-transport result for it. Unattended builds SHALL NOT invoke a model-backed Codex session. Actual Codex client acceptance SHALL be available on demand and reported separately. Access to a private host SHALL be scoped to trusted execution; the workflow SHALL NOT expose a host engine socket or private-host credential to untrusted pull-request code.

#### Scenario: Candidate image is built

- **WHEN** a candidate image is built for a reviewed source revision
- **THEN** its workflow records deterministic MCP initialization, tool enumeration and benign tool-call results tied to that exact revision and image digest, without a model invocation

#### Scenario: Codex acceptance is requested

- **WHEN** an operator explicitly runs an on-demand Codex client test against the isolated image
- **THEN** its result is recorded separately with the same source and image identities; unattended builds identify the model-backed test as intentionally not run

#### Scenario: Acceptance prerequisites are unavailable

- **WHEN** the image engine or isolated test host is unavailable
- **THEN** the workflow visibly reports the missing prerequisite and does not report real-image protocol acceptance as passed
