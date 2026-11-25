# Proposal

## Why

The 2026-11-25 audit snapshot reports 9 vulnerable development package entries in the Pi extension lockfile and 14 in the VS Code extension lockfile. Both production-only audits (`npm audit --omit=dev`) report zero findings; recording and reviewing dated development-tooling findings provides maintainers a reproducible basis for deciding updates without treating registry severity as a release policy.

## What Changes

- Define dated advisory review for the two locked JavaScript toolchains, with package-entry counts and runtime/development scope recorded separately.
- Require review and disposition of dated findings and repeatable clean-install audit evidence.
- Keep severity thresholds out of this contract until maintainers approve a policy.

## Capabilities

### New Capabilities
- `javascript-dependency-advisory-review`: dated advisory audit evidence and disposition for extension dependency trees.

### Modified Capabilities
- None.

## Impact

`pi-extension/package-lock.json`, `ide/vscode-extension/package-lock.json`, package manifests, and release/CI review procedures. The snapshot is registry-time dependent and does not establish reachability or production exploitability.
