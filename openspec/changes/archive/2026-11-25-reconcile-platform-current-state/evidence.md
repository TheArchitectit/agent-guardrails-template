# INV-02 evidence

## Source inspection

In [server startup](../../../../mcp-server/cmd/server/main.go#L139), the audit logger
is created before the database connection. After PostgreSQL connects and schema
compatibility is checked, lines 139–140 attach `database.NewAuditStore(db)`
with `auditLogger.SetStore(...)` and log `auditLogger.HasDurableStore()`.

The historical platform inventory retains its 2026-10-03 audit date and now
records the attachment in the inspected startup path. Operational persistence
and coverage of individual `slog` events require integration evidence. The
change updates the inventory and its OpenSpec requirement.

## Validation

- The repository Markdown/link check passed with 1,275 internal links across
  824 Markdown files before archival. The independent replay used the staged
  Git tree with LF line endings in Docker Desktop's Linux engine.
- OpenSpec 2.2.0 strict validation passed for the active change.
- `git diff --check` passed for the implementation before archival.
