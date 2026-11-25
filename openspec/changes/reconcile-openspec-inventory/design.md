# Design

## Context

Published specs, archived deltas, and the pre-reboot design ledger use
different identities and dates. The [proposal](proposal.md) tracks inventory
and traceability work across those records.

## Decisions

- Discover the published specs from `openspec/specs/*/spec.md` and identify
  each by its full slug. Record the canonical or superseded relationship for
  each overlapping slug after comparing the requirements and scenarios.
- Record declaration, implementation, and verification separately for each
  surviving requirement. Each verification reference should name the
  relevant test or artifact and the revision it represents. Record missing
  evidence explicitly so the inventory can retain every requirement.
- Generate the reviewed index from those records with a reproducible check.
  Preserve archived deltas and add links to their current destination. Keep
  historical ledger counts attached to the revision or date they describe.

## Rollout

Review divergent requirements before assigning supersession. Check that the
catalog accounts for all spec files and that archived requirements resolve to
a destination or an explicit disposition before enabling the inventory gate.
