# Design

## Context

The inspected server startup constructs the audit logger, connects to the configured database, then attaches `database.NewAuditStore(db)` and logs whether a durable store is configured. This source establishes a wiring fact, not complete retention, querying, or operational durability guarantees.

## Goals / Non-Goals

**Goals:** Correct the single contradicted statement using a direct code reference and keep its scope explicit.

**Non-Goals:** Redesign audit storage, infer coverage for unrelated `slog` records, or state that persistence behavior has passed a database integration test unless such evidence is independently recorded.

## Decisions

- Replace the outdated sentence with a dated, source-linked statement describing the `SetStore` call.
- Separate startup attachment from the persistence behavior and operational guarantees of every audit event.
- Verify the diff is limited to the relevant paragraph and the code reference resolves.

## Risks / Trade-offs

- A source call does not alone prove durable writes under all runtime configurations; retain that limitation and avoid a blanket “all audit events persist” claim.
