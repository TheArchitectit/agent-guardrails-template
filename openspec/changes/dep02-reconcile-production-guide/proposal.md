# Proposal

## Why

`mcp-server/deployment-guide.md` is dated 2026-08-15 and contains manual deployment commands that predate current profile validation, migration-job, secret-source, and staging controls. Operators need one current procedure that matches supported controls and has been exercised as written.

## What Changes

- Reconcile the production guide with selected profile, render/preflight flow, protected secret handling, staging verification, migration job ordering, readiness, and documented rollback boundaries. Remove or clearly mark superseded commands.

## Capabilities

### New Capabilities

- `production-deployment-procedure`: reconcile the production deployment guide evidence for an enterprise rollout.

### Modified Capabilities

None.

## Impact

`mcp-server/deployment-guide*.md`, `docs/mcp-server/migration-procedures.md`, and supported deployment command paths. Do not place credentials or private endpoint values in docs.
