# OpenSpec: Control-Plane Composition and Evidence

**Status:** Proposed — current repositories contain components and an
architecture direction, but no verified end-to-end control-plane contract.
**Priority:** Important; begin after the capability specs identify stable
interfaces and acceptance tests.
**Related:** `docs/specs/guardrail-gaps-2026/STATUS.md`,
`07-versioned-policy-pack-governance.md`, and all six guardrail-gap specs.
**Prior art reviewed:** private `guardrails-control-plane` architecture and
its pinned DevGate specs. The control-plane repo describes one-way
composition and trust boundaries; DevGate includes game-quality gates. This
proposal addresses guardrail-component composition and evidence only. Game
regression, game-type phase gating, and per-screen scene tracking remain in
DevGate’s domain and are not copied here.

## 1. Problem

The template and related private repositories contain multiple useful
components, but code presence does not establish that a component is called,
configured, or effective in the request path. Current documentation has
repeatedly mistaken a library for a shipped control. There is no common
machine-readable manifest that answers:

- which components are present and which are actually enabled;
- what policy/configuration digest they run with;
- which checks ran for a request or CI evaluation;
- which checks were skipped, unavailable, or inconclusive; and
- what evidence supports the final allow/block/defer decision.

A control plane must compose independently versioned components without
turning missing evidence into a pass or allowing a lower-trust component to
rewrite policy.

## 2. Goals and non-goals

### Goals

1. Define component identity, version, dependencies, enablement, and trust
   level in a machine-readable manifest.
2. Make integration status explicit: declared, installed, initialized,
   reachable, and exercised are distinct states.
3. Produce replayable evidence for evaluations, including missing or skipped
   checks, without claiming more than the evidence proves.
4. Enforce one-way dependencies: enforcement components may consume policy
   and emit evidence, but must not silently rewrite their own trust roots.
5. Support staged adoption and independent component failure handling with
   fail-closed behavior for mandatory controls.

### Non-goals

- Reimplementing the guardrail subsystems in a meta-repository.
- Claiming certification or regulatory compliance from a successful run.
- Automatically importing every component from similarly named private
  repositories.
- Including game-specific CI gates (those remain in the DevGate specs).
- Treating a signed or replayed evaluation as proof that a policy is correct.

## 3. Terminology

- **Component:** a versioned executable or library that supplies one or more
  checks or evidence sources.
- **Control manifest:** declared components, immutable versions, dependencies,
  policy inputs, and mandatory/optional classification.
- **Evaluation:** one bounded run over a subject (request, change, build, or
  release) under a known manifest and policy digest.
- **Evidence bundle:** machine-readable record of checks, inputs by digest,
  results, errors, and provenance, excluding secret values.
- **Integration state:** `declared`, `installed`, `initialized`, `reachable`,
  `exercised`, or `failed`.

## 4. Requirements

### 4.1 Component manifest and dependency graph

1. Every component MUST have a stable ID, immutable version or commit,
   digest, owner, supported interface version, trust tier, and declared
   dependencies.
2. The resolver MUST reject dependency cycles, incompatible interface
   versions, missing mandatory dependencies, and mutable production refs.
3. Dependency direction MUST be acyclic and explicit. A consumer MUST NOT
   acquire policy-writing authority solely because it is a dependency.
4. Each component MUST declare whether it is mandatory or optional for each
   evaluation profile. Missing mandatory components MUST result in
   `incomplete` or `blocked`, never `passed`.
5. Profiles MUST specify required checks and minimum evidence, not merely a
   list of installed components.

### 4.2 Integration state and health

1. The control plane MUST distinguish `installed` from `reachable` and
   `reachable` from `exercised`.
2. A component MUST report initialization outcome, interface version, active
   configuration/policy digest, and health without exposing credentials.
3. A check that times out, errors, is skipped, or returns an unknown result
   MUST be represented explicitly. It MUST NOT be converted to a successful
   result.
4. Readiness MUST report per-mandatory-component state and return not-ready
   when a mandatory component is unavailable. Liveness and readiness MUST be
   separate signals.
5. A status endpoint or report MUST avoid claiming that an unwired library is
   active protection.

### 4.3 Evaluation evidence and replay

1. Every evaluation MUST have an immutable evaluation ID, subject digest,
   manifest digest, policy digest, start/end times, and evaluator version.
2. Each check record MUST include component ID/version, check ID, input
   digest, outcome (`pass`, `block`, `inconclusive`, `error`, `skipped`),
   reason code, and bounded diagnostic detail.
3. Secret values, raw credentials, and unnecessary user content MUST not be
   stored in an evidence bundle. Redaction rules MUST be tested.
4. The overall result MUST be derived deterministically from profile
   requirements and per-check outcomes. A mandatory `error`, `inconclusive`,
   or `skipped` check MUST prevent a `pass` result.
5. A replay MUST identify unavailable external inputs and MUST NOT label a
   non-equivalent replay as identical. Store hashes and version identifiers,
   not a claim of perfect reproducibility.
6. Evidence retention, signing, access control, and deletion policy MUST be
   configurable and auditable.

### 4.4 Staged adoption and change governance

1. A profile MAY begin in observe-only mode, but reports MUST label that mode
   and MUST NOT present observe-only results as enforcement.
2. Promotion from observe-only to blocking MUST require a reviewed change,
   explicit profile version, and passing acceptance suite.
3. Changes to mandatory policy roots, component trust, or exception
   allowlists MUST require an authorized human review and produce an audit
   record.
4. Rollback MUST restore the prior manifest and policy digest atomically; a
   partial rollback MUST report failure and leave the system not-ready.
5. A control plane MUST not claim a check is active unless an end-to-end test
   proves the request/evaluation path reaches it.

### 4.5 Trust boundaries

1. Policy inputs, component output, user content, external content, and
   evidence metadata MUST have distinct trust labels.
2. Lower-trust component output MUST NOT modify the manifest, trusted key set,
   policy lock, or mandatory control list.
3. Component-provided recommendations MUST be data for a decision, not
   executable instructions.
4. The system MUST record the source and digest of every policy input used by
   an evaluation.
5. When a trust label or provenance chain is unknown, the result MUST be
   explicit and follow the selected profile’s fail-closed rule.

## 5. Acceptance criteria

The implementation is acceptable only when a clean test environment can
reproduce all of these checks:

1. A manifest with an incompatible dependency or cycle is rejected before
   any component starts.
2. A component that is installed but not wired reports `installed` but not
   `reachable`/`exercised`; the system-level report does not claim its
   protection is active.
3. A mandatory component that is missing, times out, errors, or returns
   unknown prevents the overall result from being `pass`.
4. Given fixed manifest, policy, subject, and deterministic test doubles, the
   evaluation result and evidence digest are stable across repeated runs.
5. Evidence records identify all checks and explicit skip/error states while
   containing no seeded secret values from the redaction test fixture.
6. A replay with a changed component version or policy digest is marked
   non-equivalent.
7. Observe-only and blocking profiles produce distinguishable results, and
   observe-only cannot satisfy a blocking release gate.
8. Rollback tests prove the previous manifest/policy digest is restored as a
   unit; an interrupted rollback leaves readiness false.
9. An integration test proves each mandatory check is invoked through the
   actual request/evaluation entry point, not merely constructed in a unit
   test.

No control-plane implementation or evidence schema currently exists here;
these criteria are proposals, not passing tests.

## 6. Dependencies and sequencing

1. Complete and stabilize the individual capability contracts first. This
   spec depends on their callable interfaces, not just library packages.
2. Implement policy identity and locking from
   [07-versioned-policy-pack-governance.md](07-versioned-policy-pack-governance.md)
   before composing policies from multiple sources.
3. Start with one bounded evaluation profile and two components; add broader
   orchestration only after evidence semantics are tested.
4. Keep DevGate game-specific checks in DevGate. Integrate them only through
   the component manifest if a future product decision requires a shared
   control plane.

## 7. Open decisions

1. Is the control plane a package in this repository, a separate repository,
   or a thin coordinator over existing services?
2. Which evaluation subject is first: MCP requests, code changes, builds, or
   releases? The initial vertical slice should choose exactly one.
3. Where is evidence stored, for how long, and who may read/delete it?
4. Which components are mandatory for the first profile, and what is the
   explicit failure action for each?
5. What signing/key-management model is required for manifests and evidence?
6. Does staged rollout use profiles, environments, or a separate promotion
   record? Choose one source of truth.

## 8. Traceability and boundaries

| Existing work | Relationship |
|---------------|--------------|
| Six `guardrail-gaps-2026` specs | Define capabilities this spec may compose only after they are reachable and testable |
| Spec 07 — policy-pack governance | Supplies deterministic policy identity and trust rules |
| Private `guardrails-control-plane` repo | Prior art for one-way composition and pinned components; its claims are not assumed implemented here |
| Private DevGate OpenSpecs | Game-quality gates remain out of scope for this guardrail control-plane proposal |

**Implementation status:** proposal only. No manifest resolver, integration-
state model, evidence bundle, replay system, staged promotion flow, or
control-plane readiness evaluator is implemented in this repository.