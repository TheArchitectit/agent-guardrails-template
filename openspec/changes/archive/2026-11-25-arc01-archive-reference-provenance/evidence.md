# ARC-01 archive provenance evidence

## Historical AIGGP outputs

The four dated AIGGP packages contain eight delta files that publish seven
unique short-slug spec paths. `openspec/changes/README-aiggp-routed-packages.md`
now enumerates every mapping with local links. `evidence-and-conformance` is
one shared published path produced by both aiggp-03 and aiggp-04; the two
archive delta files are separately linked. `isolation-levels` is included.
Archived proposals, deltas, task records, and acceptance files were not edited.

The package headers reference
`openspec/changes/AIGGP-RETIREMENT-2026-10-02.md`, which is absent from this
checkout. Repository search found no verified replacement retirement record.
The manifest now identifies the reference as unavailable and preserves its
date and filename rather than inventing replacement provenance. The historical
package header references remain byte-preserved.

## Validation

The eight archived delta files map to seven unique output paths. Every mapped
archive and published-spec link resolves locally, including both links to the
shared `evidence-and-conformance` output. The missing retirement target was
checked locally and was not present.

- `openspec validate --archived --strict` with OpenSpec 2.2.0: 19 passed,
  0 failed.
- `openspec validate --all --strict` with OpenSpec 2.2.0: 64 passed,
  0 failed.
- Internal Markdown links in the archive manifest and both evidence files:
  passed.
- `git diff --check`: passed.

This is historical archive traceability only; it does not assert that archived
task checkboxes prove current runtime behavior.
