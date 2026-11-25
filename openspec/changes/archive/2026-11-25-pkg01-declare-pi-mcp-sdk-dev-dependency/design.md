# Design

## Context

Bridge source imports SDK subpaths dynamically and has an explicit absent-package fallback. The manifest currently lists only `@sinclair/typebox` under production dependencies and no MCP SDK entry in the lockfile. Making the SDK a production dependency would alter runtime requirements for standalone users.

## Goals / Non-Goals

**Goals:**

- Make SDK-present bridge code available to clean development tests.
- Preserve no-SDK standalone runtime behavior.

**Non-Goals:**

- Make the SDK mandatory for package installation or standalone features.
- Change the bridge endpoint contract.

## Decisions

- Add the compatible SDK to `devDependencies` and the lockfile; do not add it under production `dependencies` or make it a required peer dependency.
- Test the installed-SDK bridge path after a clean locked install.
- Test the absent-SDK path from an isolated packed production install with dev dependencies omitted; assert standalone import succeeds and bridge availability is reported false without throwing.

## Risks / Trade-offs

- [Development tests still do not exercise SDK loading] → add direct SDK-present bridge assertions, not only tests of the missing-package fallback.
- [The SDK accidentally becomes a runtime requirement] → inspect production dependency metadata and run the packed install without development dependencies.

## Migration Plan

No consumer install change is required. Run `npm ci && npm test` from `pi-extension/`, then the isolated production-package smoke test from `tasks.md`.
