# Design

## Context

The index declares itself generated, but the repository review must identify a reproducible source before treating the Markdown file as derived output. The audit companion records eight indexed and seven unindexed spec paths and documents requirement-level overlap.

## Goals / Non-Goals

**Goals:** Make the inventory exhaustive and status inputs traceable by exact path; preserve distinctions among overlapping requirements.

**Non-Goals:** Merge short and verbose specs by slug, change implementation status without review, or rewrite archived proposals as current behavior.

## Decisions

- Use full repository paths as identities; short names are display labels only.
- Compare each requirement block and closure evidence before proposing consolidation or supersession.
- If a generator is absent, document the source-of-truth and update procedure instead of inventing a generator.
- Keep archive README and stale retirement-reference work in the separate `ARC-01` change.

## Risks / Trade-offs

- A complete index may expose contradictory closure claims; annotate the evidence and leave substantive acceptance to the responsible owner.
