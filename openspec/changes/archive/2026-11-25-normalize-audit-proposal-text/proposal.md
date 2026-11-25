# Keep audit proposals compatible with the text integrity gate

## Why

The pinned DevGate charset gate rejected three audit proposal files in the
template draft #13. Seven section-sign glyphs prevent the draft's
documentation checks from passing.

## What Changes

- Use plain-text section citations in the three affected audit documents.
- Verify the complete draft against the existing charset policy and pinned
  control tests.

## Capabilities

### New Capabilities

- `audit-proposal-text-integrity` covers published audit proposal text.

### Modified Capabilities

None.

## Impact

Three active OpenSpec audit documents and the evidence for their acceptance.
