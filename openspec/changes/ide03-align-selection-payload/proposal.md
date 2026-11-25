# Proposal

## Why

The Go selection-validation handler reads `selection` and rejects an empty value. VS Code, JetBrains, Vim, and Neovim send selected text as `code`, so current client requests fail the handler contract.

## What Changes

- Make `selection` the canonical field for new IDE clients.
- Accept legacy `code` only when `selection` is absent; reject different values when both fields are present.
- Add handler tests for canonical, legacy, conflicting, and empty payloads, plus request tests for all supported IDEs.

## Capabilities

### New Capabilities

- `ide-selection-contract`: Maintain a safe request contract across the handler and IDE clients.

### Modified Capabilities

None.

## Impact

Affects the Go selection handler and four IDE client builders. Existing installed clients that send only `code` remain compatible during migration. No implementation is included in this planning package.
