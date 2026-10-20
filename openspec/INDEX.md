# OpenSpec index

> **Generated from the tree, not typed.** This repo has no `openspec/specs/`
> capability directory — every spec artefact lives inside a change package
> under `openspec/changes/<change>/`. So this index is a census of change
> packages, and status is derived from what each package actually contains.

**Change packages:** 4 open · 0 archived

| Change | Spec deltas | Package contents | Status |
|---|---|---|---|
| [`aiggp-03-prompt-injection-defense`](changes/aiggp-03-prompt-injection-defense/proposal.md) | 2 | acceptance, adrs, design, proposal, specs, tasks | **PROPOSED** |
| [`aiggp-04-semantic-content-filtering`](changes/aiggp-04-semantic-content-filtering/proposal.md) | 2 | acceptance, adrs, design, proposal, specs, tasks | **PROPOSED** |
| [`aiggp-05-runtime-sandbox-isolation`](changes/aiggp-05-runtime-sandbox-isolation/proposal.md) | 2 | acceptance, adrs, design, proposal, specs, tasks | **PROPOSED** |
| [`aiggp-06-indirect-injection-provenance`](changes/aiggp-06-indirect-injection-provenance/proposal.md) | 2 | acceptance, adrs, design, proposal, specs, tasks | **PROPOSED** |

## Reading the status

- **PROPOSED** — the package is complete (proposal, design, ADRs, spec deltas,
  tasks, acceptance) but carries no closure evidence: `tasks.md` records phase
  outlines rather than checked-off work, so there is nothing to derive a
  CLOSED/PARTIAL from. That is a documentation gap, not a claim the work is
  unimplemented.

## What "open" means here

All four packages are **open** in the OpenSpec sense: they are changes, not
capabilities. This repo has not archived any of them, so none has been declared
shipped. Promoting one means moving it under `openspec/changes/archive/` with
its tasks checked and its spec deltas merged into a capability spec.

## Honest limit

An index generated from a repo that keeps specs only inside change packages
cannot report per-requirement CLOSED/PARTIAL the way a capability-spec repo can.
Adding `openspec/specs/<capability>/spec.md` files would make that possible; it
has not been done here, and this index does not pretend otherwise.
