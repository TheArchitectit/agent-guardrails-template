# vscode-extension-linting Specification

## Purpose

Defines project-scoped lint behavior for the VS Code extension package.

## Requirements

### Requirement: Lint the VS Code TypeScript source in the package lifecycle

The VS Code extension SHALL provide a package-scoped ESLint configuration recognized by its existing lint script, using the locked ESLint and TypeScript parser/plugin dependencies without adding or upgrading dependencies. The configuration SHALL report parser errors and substantive TypeScript source violations, including unused bindings and explicit `any` types. The existing `npm test` lifecycle SHALL continue to compile, lint, and then execute the activation test.

#### Scenario: Normal package test lifecycle

- **GIVEN** locked package dependencies are installed
- **WHEN** a maintainer runs `npm test` from the VS Code extension package
- **THEN** compilation and configured linting pass before the activation test runs

#### Scenario: Invalid TypeScript source

- **GIVEN** a temporary TypeScript source file contains invalid syntax
- **WHEN** the package lint command analyzes `src`
- **THEN** ESLint reports a parser error and exits unsuccessfully

#### Scenario: Parseable TypeScript rule violation

- **GIVEN** a temporary TypeScript source file parses but declares an unused binding
- **WHEN** the package lint command analyzes `src`
- **THEN** ESLint reports the unused-binding rule and exits unsuccessfully
