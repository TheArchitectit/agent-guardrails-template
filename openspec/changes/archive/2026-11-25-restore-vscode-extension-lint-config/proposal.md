# Proposal

## Why

The VS Code package declares a locked ESLint 8 toolchain and a tracked `npm run lint` script, but has no ESLint configuration. ESLint exits before checking source, so the package's normal `pretest` lifecycle stops after compilation and never runs the activation regression.

## What Changes

- Add a VS Code-extension-scoped ESLint configuration using the existing locked TypeScript parser and plugin.
- Enable substantive ESLint and TypeScript-aware source diagnostics without changing dependencies or the existing compile, lint, and test scripts.
- Exercise parse failures and a parseable rule violation with temporary fixtures, and run the complete package workflow.

## Capabilities

### New Capabilities

- `vscode-extension-linting`: Provide runnable, scoped TypeScript linting in the VS Code package lifecycle.

### Modified Capabilities

None.

## Impact

Adds a project-scoped ESLint config and validation evidence for the VS Code extension. The existing lockfile and package scripts remain unchanged. No dependency upgrades or unrelated IDE behavior changes are included.
