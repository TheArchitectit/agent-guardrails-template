# Spec Delta

## Purpose

Defines the MCP Docker build-context boundary and the source assets that must remain available to the builder.

## ADDED Requirements

### Requirement: MCP build context excludes local-only files
The Docker build SHALL exclude VCS metadata, local credentials/configuration, caches, and test/build artifacts from the `mcp-server` context while retaining Go module files, Go source, web assets, and prevention rules required by the builder.

#### Scenario: local secret and cache are present
- **GIVEN** the build context contains a benign `.env` marker, `.git` data, and a local cache fixture
- **WHEN** the Dockerfile builder executes `COPY . .`
- **THEN** those local-only paths are unavailable to the builder and no marker content is printed

#### Scenario: required build inputs are present
- **GIVEN** the production MCP build uses the declared context
- **WHEN** the builder copies source inputs
- **THEN** `go.mod`, `go.sum`, Go source, `web/`, and `.guardrails/prevention-rules` remain available and the representative image build succeeds
