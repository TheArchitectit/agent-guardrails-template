# Design

## Context

Docker resolves `.dockerignore` from the build-context root. The documented context is `mcp-server`, and the builder runs `COPY . .`. A repository-root ignore file would not govern that context. Required inputs include Go module files, source, web assets, and `.guardrails/prevention-rules`.

## Goals / Non-Goals

**Goals:**

- Exclude local-only files from context transfer and intermediate builder layers.
- Retain all inputs required by the current Dockerfile and runtime artifact assembly.

**Non-Goals:**

- Claim arbitrary context files currently enter the final runtime image.
- Change the Dockerfile's final-stage selection of artifacts.

## Decisions

- Place ignore rules at `mcp-server/.dockerignore` and exclude `.git`, local environment/credential files, local databases, caches, test outputs, and editor artifacts.
- Explicitly retain `go.mod`, `go.sum`, Go source, `web/`, and `.guardrails/prevention-rules`. CA certificates and zoneinfo are installed in the image builder and need not be copied from the host context.
- Use a temporary context with safe marker contents and a context-inspection target; assert ignored files are unavailable and required assets remain available. Then run the production Dockerfile build.

## Risks / Trade-offs

- [Broad ignore rules omit required source] → assert required paths in the context fixture and run a representative image build.
- [Test leaks a marker value] → use non-secret synthetic content and assert only path presence/absence; never print file contents.

## Migration Plan

No application migration is required. Add the context ignore file, run the context fixture and representative Docker build, and inspect the resulting context path list.
