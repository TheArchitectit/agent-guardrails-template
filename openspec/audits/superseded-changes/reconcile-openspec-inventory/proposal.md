# Proposal: Reconcile the OpenSpec inventory

## Why

The current OpenSpec index reports eight specifications, all closed. A
repository review counted fifteen specification files: the eight indexed
entries and seven additional files. Several have overlapping short names
while describing different requirement sets, and the four completed aiggp
packages are archived. The index therefore does not provide a complete,
unambiguous inventory or a direct trace from archived
work to the current specifications. This is an inventory and traceability
finding; it does not establish that implementation requirements are unmet.

The archive README counts six unique short-spec outputs although the four
packages produce seven. It also omits `isolation-levels`, and package headers
refer to a retirement record absent from this checkout. These manifest issues
are tracked as `ARC-01`.

## Change

Define a reproducible inventory that distinguishes specifications by their
full repository paths, records closure evidence for every indexed spec, and
links archived changes to the specs they produced. Resolve ambiguous short
IDs without erasing either semantically distinct specification. Keep the
index's generated status claims consistent with its inputs.

## Impact

This is documentation and OpenSpec process work. It improves reviewability of
status and provenance claims without changing runtime behavior or asserting
new implementation gaps. Audit finding `INV-01` in
`openspec/audits/2026-11-25-qa-and-gap-review.md` tracks the observed
inventory/identity ambiguity.

## Scope

Reconcile the inventory and its traceability. Do not modify production code,
retroactively claim closure, or collapse specifications solely because their
short names overlap.
