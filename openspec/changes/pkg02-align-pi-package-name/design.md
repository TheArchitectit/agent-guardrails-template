# Design

## Context

The manifest and registry identify the published package as `@architectit/pi-guardrails` v0.2.0. README and `install.mjs` currently show a different scope. The internal install target `~/.pi/agent/extensions/pi-guardrails` is independent of the npm package name.

## Goals / Non-Goals

**Goals:**

- Align all user-facing install references with the verified published identity.
- Preserve the installed extension directory and existing package contents.

**Non-Goals:**

- Rename or move the local extension directory.
- Publish a new package or alter registry metadata as part of this change.

## Decisions

- Use `@architectit/pi-guardrails`, which matches both the current manifest and verified registry listing.
- Update README, installer usage/help, and package validation to that exact identity.
- Test the package identity across sources and run installer help with a temporary home to ensure it performs no installation/removal.

## Risks / Trade-offs

- [A stale command remains] → compare all install examples and the generated help against the manifest in a test.
- [A smoke test mutates a user installation] → set `HOME` to a temporary directory and invoke only `--help`.

## Migration Plan

No package publication is needed; the existing registry identity is retained. Update the references and run the package tests and dry-run pack checks in `tasks.md`.
