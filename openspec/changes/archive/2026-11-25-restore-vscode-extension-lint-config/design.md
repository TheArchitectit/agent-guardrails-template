# Design

## Context

`ide/vscode-extension/package.json` runs `eslint src --ext ts` and its `pretest` lifecycle runs compile then lint before the activation harness. The extension already locks ESLint 8, `@typescript-eslint/parser`, and `@typescript-eslint/eslint-plugin`, but no configuration exists in the package or its ancestors. As a result, lint exits before source analysis and `npm test` never reaches the activation suite.

## Goals / Non-Goals

**Goals:**

- Make the tracked lint command recognize a package-local configuration.
- Apply substantive recommended core and TypeScript diagnostics, including unused-variable and explicit-`any` checks.
- Keep the extension's existing compile → lint → activation-test lifecycle and locked dependencies.
- Demonstrate both parser and rule failures using disposable source fixtures, then prove the normal package workflow passes.

**Non-Goals:**

- Upgrade or add dependencies, change the package scripts, or modify other IDE clients.
- Enable type-aware lint rules that require a separate TypeScript project configuration.
- Perform unrelated refactoring.

## Decisions

- Add `.eslintrc.json` at the VS Code package root with `root: true`, the existing TypeScript parser/plugin, and the `eslint:recommended` plus `plugin:@typescript-eslint/recommended` sets. Scope is the package's existing `src` target.
- Explicitly keep unused-variable and explicit-`any` diagnostics at error severity, while allowing underscore-prefixed intentionally unused bindings; reject debugger statements.
- Permit empty class methods for lifecycle hooks such as the client's intentional no-op `dispose()`; retain the other recommended diagnostics.
- Leave `pretest`, `compile`, `lint`, and `test` scripts intact so `npm test` remains the complete package entry point.
- Use a temporary syntactically invalid TypeScript file to prove parser errors and a separate parseable unused-variable fixture to prove rule execution; remove both fixtures before final verification.

## Risks / Trade-offs

- [Recommended rules surface existing source defects] → address only small defects necessary for the extension to pass and record any lint suppression or scope choice explicitly.
- [The client has an intentional empty `dispose()` hook] → use the TypeScript rule's method exception, since no resources exist to release, without suppressing other recommended rules.
- [The rule set is mistaken for type checking] → state that this is parser-based linting; TypeScript compilation remains a separate pretest step.
- [Validation fixtures remain in the package] → create them only for controlled negative runs and confirm the fixture paths are absent before the final locked workflow.

## Migration Plan

No dependency or configuration migration is needed. Run `npm ci`, `npm run compile`, `npm run lint`, and `npm test` from `ide/vscode-extension/`, plus strict OpenSpec validation.
