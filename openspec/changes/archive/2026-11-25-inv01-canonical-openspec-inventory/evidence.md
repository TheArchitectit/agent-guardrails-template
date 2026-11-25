# INV-01 inventory evidence

## Scope and source-of-truth

At the prepared baseline `1ff334b04a7e5b70eb3ae1fc2519ac06d2e26e1a`,
`openspec/INDEX.md` had eight rows while the checkout had 30 files matching
`openspec/specs/**/spec.md`. The earlier review snapshot `943bcf65055128e338b696dc40d74e79ac33449e`
was independently enumerated from its Git tree and had 15 files: eight verbose
specs and seven short-slug specs. The difference reflects specifications
published after that review snapshot; the current count is not retroactively
substituted for the historical count.

Repository searches found no checked-in generator or state ledger for
`openspec/INDEX.md`. The index now documents a manual reproducible update
procedure: enumerate exact spec paths, sort them, derive each status only from
the corresponding file's explicit status or closure evidence, and run strict
spec validation. The full paths remain separate even where names or
requirements overlap.

## Current inventory reconciliation

All 30 current paths are listed in [`openspec/INDEX.md`](../../../INDEX.md).
Statuses were read only from each file's own explicit `Status:` line or actual
requirement closure table: 16 CLOSED, 0 PARTIAL, 0 OPEN, and 14 UNKNOWN.
UNKNOWN means the spec has no status line or closure table; it is not a claim
of incomplete implementation. In particular, ordinary prose containing the
word “absent” in the Gitleaks and Pi SDK specs is not closure evidence. No spec
is marked CLOSED based on an archived task list.

The historical seven short-slug specs remain distinct from their verbose
counterparts. Requirement-level comparison confirms non-identical text and
scenario contracts; no owner-approved consolidation or supersession decision
was found in this checkout, so this change preserves all requirement IDs and
publishes both rows:

| Short-slug path | Related verbose path(s) | Disposition |
|---|---|---|
| [`categories-and-actions`](../../../specs/categories-and-actions/spec.md) | [`semantic-filtering-categories-and-actions`](../../../specs/semantic-filtering-categories-and-actions/spec.md) | Distinct category/action and evidence conditions retained. |
| [`classification-and-mediation`](../../../specs/classification-and-mediation/spec.md) | [`prompt-injection-classification-and-mediation`](../../../specs/prompt-injection-classification-and-mediation/spec.md) | Distinct scenario text retained. |
| [`evidence-and-conformance`](../../../specs/evidence-and-conformance/spec.md) | [`prompt-injection-evidence-and-conformance`](../../../specs/prompt-injection-evidence-and-conformance/spec.md); [`semantic-filtering-evidence-and-conformance`](../../../specs/semantic-filtering-evidence-and-conformance/spec.md) | Shared archive output path remains distinct from both verbose requirements. |
| [`isolation-levels`](../../../specs/isolation-levels/spec.md) | [`sandbox-isolation-levels`](../../../specs/sandbox-isolation-levels/spec.md) | Runtime-boundary and evidence requirements retained. |
| [`conformance-and-evidence`](../../../specs/conformance-and-evidence/spec.md) | [`sandbox-conformance-and-evidence`](../../../specs/sandbox-conformance-and-evidence/spec.md) | Distinct conformance requirement text retained. |
| [`policy-and-conformance`](../../../specs/policy-and-conformance/spec.md) | [`provenance-policy-and-conformance`](../../../specs/provenance-policy-and-conformance/spec.md) | Distinct policy and evidence requirement text retained. |
| [`provenance-records`](../../../specs/provenance-records/spec.md) | [`indirect-injection-provenance-records`](../../../specs/indirect-injection-provenance-records/spec.md) | Distinct records and provenance requirement text retained. |

The [requirement-to-code audit](../../../audits/2026-11-25-template-requirement-trace.md)
provides additional context on the distinct requirement sets and coverage
limits.

## Integrated index

The current index includes later canonical specs created after the audited
baseline. Its count and status totals are computed from the full integrated
source tree. The 30-path snapshot remains the basis for the separate
comparison against the original eight-row index.

## Verification performed on the prepared baseline

- `openspec validate --specs --strict` with OpenSpec 2.2.0: 30 passed, 0 failed.
- `openspec validate --all --strict` with OpenSpec 2.2.0: 64 passed, 0 failed.
- Internal Markdown links in the index, routed-package manifest, and both
  evidence files: passed; 30 spec rows matched the validator's 30 discovered
  spec paths.
- `python scripts/regression_check.py --all` in the available Python/Git
  container: no potential regressions detected.
- `git diff --check`: passed.
