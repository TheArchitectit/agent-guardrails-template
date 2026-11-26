# Proposal

## Why

The manifest and registry identify the published package as `@architectit/pi-guardrails`, while README and installer help use `@thearchitectit/pi-guardrails`. Users following those commands request a different package identity.

## What Changes

- Use the verified published name `@architectit/pi-guardrails` consistently in install examples and help.
- Keep the internal extension directory `pi-guardrails` unchanged and add package/help smoke tests.

## Capabilities

### New Capabilities

- `pi-package-identity`: Keep Pi package metadata and user-facing install commands bound to one identity.

### Modified Capabilities

None.

## Impact

Affects package README, `install.mjs`, and packaging tests. Registry identity and local install directory remain unchanged. No implementation is included in this planning package.
