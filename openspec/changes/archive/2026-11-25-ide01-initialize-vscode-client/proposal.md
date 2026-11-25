# Proposal

## Why

`activate()` constructs `GuardrailClient` and calls `testConnection()` before `updateConfiguration(context)`. The client initializes configuration only in that method, so its first fetch reads an unset URL and reports the extension disconnected.

## What Changes

- Initialize the client from workspace settings and extension SecretStorage before the first connection or validation request.
- Preserve command/event registration and later configuration refreshes.
- Add activation tests for a successful configured request and an unreachable-server control.

## Capabilities

### New Capabilities

- `ide-client-initialization`: Initialize VS Code client configuration and secrets before network-dependent behavior.

### Modified Capabilities

None.

## Impact

Affects VS Code extension activation and tests. Existing settings, commands, and extension lifecycle remain compatible. No implementation is included in this planning package.
