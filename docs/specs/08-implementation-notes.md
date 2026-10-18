# Spec 08 implementation notes — R08-4.1.5 profile requirement set

Companion to `08-control-plane-composition-and-evidence.md`. The spec file is at
its 500-line cap, so the detail of this requirement lives here and the spec
carries the status and the pointer.

## What was missing

R08-4.1.5 is one sentence:

> Profiles MUST specify required checks and minimum evidence, not merely a list
> of installed components.

Until this pass the manifest declared components and nothing else. `mandatory`
was a label the evaluator checked against measured state, and the profile field
was recorded and reported — but no manifest could say *what evidence the profile
accepts*. Concretely: nothing could express "the detector must be reachable" or
"this check must have run", so a profile's obligations lived in prose in a spec
and in nobody's code.

The gap was not "a feature is absent". It was that **the manifest could not
fail**: every requirement a reader inferred from the prose was unenforceable, so
the composition could pass while proving less than the profile claimed.

## What landed

`internal/controlplane/manifest.go`:

- `EvidenceFloor` with exactly two legal values, `reachable` and `exercised`.
  `declared`, `installed` and `initialized` are **refused**: each is a statement
  that the code is present, which is precisely the "merely a list of installed
  components" the requirement forbids. `failed` is refused too — a floor is the
  weakest acceptable answer, not a description of a broken one.
- `Requirement` binding components (with floors) and named checks, plus an
  optional `minimum_passing_checks` floor. Absent means "all named checks".
- `resolveRequirements` runs inside `Resolve()`, so no caller can hold a
  manifest with an unvalidated requirement set.

`internal/controlplane/evaluate.go`:

- `RequirementResult` per obligation, with `Met` derived from the measured
  states and check outcomes. Nothing assigns it.
- `evaluateRequirements` measures a component against **the floor the manifest
  declared**, never against the strongest state the process happens to have.
- A named check with no record is `did not run` and fails the requirement
  outright: a passing floor cannot be met by a check that does not exist.
- A check that ran and did not pass is **recorded**, but the declared passing
  floor decides whether the obligation is met. Hard-failing on any non-pass
  would make `minimum_passing_checks` meaningless — a floor exists so a profile
  can tolerate some inconclusive checks.
- Unmet requirements feed `deriveVerdict`, so they prevent `pass`; a `did not
  run` also marks the verdict `failed` rather than the softer `incomplete`,
  because a missing check is a wiring defect, not a refusal.

`internal/controlplane/readiness.go` and `internal/mcp/control_plane.go`:
readiness and the `guardrail://control-plane` resource both report requirement
results, so readiness, the resource and the bundle cannot disagree about one
process.

`internal/controlplane/governance.go`: the requirement set is a **governed
root**. Lowering a floor from `exercised` to `reachable`, or deleting the
requirement that held a mandatory control, changes the strength of every future
evaluation without touching a component. A diff that walked only components
would report "no change" for exactly that edit, so `RequiresReview` now walks
requirements too.

## Refusals, each for a way a requirement set can constrain nothing

| Defect | Why it is refused |
|---|---|
| No requirements at all | The exact shape the requirement forbids |
| Requirement names no component and no check | Nothing could ever fail it |
| Requirement with no description | An obligation with no stated purpose |
| Unstable or duplicate requirement id | Two obligations indistinguishable in a report |
| Floor of `declared`/`installed`/`initialized` | Restates the component list |
| Floor of `failed` | A floor is a minimum, not a defect description |
| Requirement names an undeclared component | Can never be evidenced; silently caps the profile |
| Component listed twice in one requirement | Ambiguous floor |
| Empty or duplicate check id | Same |
| `minimum_passing_checks` without checks | A floor over an empty set |
| `minimum_passing_checks` < 1 | Met by nothing |
| `minimum_passing_checks` > named count | Could never be met |
| Mandatory component held by no requirement | A `mandatory: true` row that constrains nothing |
| One component at two different floors | The weaker decides, so the stronger is decoration |

## Evidence

`requirements_test.go`: 6 refusal-matrix cases via the real `Resolve`, 14
refusal cases incl. all four illegal floors, plus evaluation, digest, governance
and readiness cases — all over the real evaluator, not a stub.

**Red-on-drift proven three ways, each PASS→FAIL then restored:**

1. Empty requirement set allowed → `TestManifestWithoutRequirementsIsRefused`
   fails.
2. Floor validation removed → four `TestRequirementRefusals` subtests fail
   (installed / declared / initialized / failed).
3. `RequirementResult.Met` forced true → four tests fail, including
   `TestUnmetRequirementPreventsPass` and
   `TestReadinessReflectsUnmetRequirements`.

The checked-in `control-plane.manifest.json` now carries two requirements:
`policy-identity` (policy-engine at `reachable`) and `injection-proof`
(injection-detector at `reachable`, naming the `canonical-injection-detected`
check with a floor of 1). Those are the floors this repository can honestly
meet today — the server has no durable audit store by default and the sandbox is
optional, so demanding `exercised` from every component would make the shipped
profile permanently `incomplete`.

## Named limits

- **Floors are per component, not per check.** A check's evidence is the check
  outcome recorded in the bundle; there is no separate "this check must have run
  N times" floor.
- **Requirements are per manifest, not selected by profile name.** A deployment
  runs one manifest, so one requirement set. Multi-profile selection is the
  manifest's job, not the resolver's.
- **Trust-label propagation (R08-4.5.3/4.5.4) remains open.** Component output
  is still not tagged with its origin trust label at evaluation time, and only
  the single policy digest is recorded rather than the digest of every policy
  input. That is the last open row in this spec.
