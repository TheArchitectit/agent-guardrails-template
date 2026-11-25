# Proposal

## Why

Four direct auth, config, and secret-file test fixtures request mode `0644` and
are created as `0600` under umask `0077`; the same fixtures pass under scoped
umask `0022`. One downstream mutation baseline also depends on a fixture whose
mode varies with the test process. Tests need deterministic fixture modes
without weakening production secret-file permissions.

## What Changes

- Set and verify expected test fixture modes explicitly, independent of the
  caller's umask.
- Keep separate assertions that production credential and secret files remain
  mode `0600`.
- Verify permission-sensitive tests under restrictive and ordinary umasks.

## Capabilities

### New Capabilities

- `test-fixture-permissions`: deterministic permission-sensitive fixtures and
  explicit production secret-file mode assertions.

### Modified Capabilities

None. Existing OpenSpec capabilities do not specify test fixture mode
independence.

## Impact

Affected tests are in `mcp-server/internal/auth`, `mcp-server/internal/config`,
`mcp-server/internal/secretfile`, and a dependent mutation baseline. No
production file mode is to be relaxed. Audit finding `QA-02` is recorded in
`openspec/audits/2026-11-25-qa-and-gap-review.md`.
