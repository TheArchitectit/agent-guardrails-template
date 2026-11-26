# Proposal: Reconcile platform audit-persistence statement

## Why

`docs/platform-current-state.md` is dated 2026-10-03 and says startup constructs an audit logger but does not attach a Postgres audit store. In the reviewed source, `mcp-server/cmd/server/main.go` calls `auditLogger.SetStore(database.NewAuditStore(db))` after connecting to the database. The current-state description should distinguish that attached store from any broader persistence or event coverage not established by this code path.

## What Changes

- Update only the stale audit-store statement to reflect the inspected startup attachment.
- Preserve the date and qualify what the source establishes: this logger is given a durable store in the inspected server path.
- Keep other subsystem persistence descriptions and broader audit coverage claims unchanged unless independently verified.

## Capabilities

### New Capabilities

- `platform-state-evidence`: The platform current-state inventory describes persistence wiring only to the extent supported by inspected source evidence.

### Modified Capabilities

None. This change adds no behavior contract to an existing capability.

## Impact

`docs/platform-current-state.md` and the source reference `mcp-server/cmd/server/main.go:139-140`. It does not assert that every `slog` event is persisted, that all deployment modes use Postgres, or that operational durability has been tested.
