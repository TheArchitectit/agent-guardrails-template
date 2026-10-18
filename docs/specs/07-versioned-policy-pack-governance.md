# OpenSpec: Versioned Policy-Pack Governance

**Status:** PARTIAL — the lock-record half (R07-4.1 identity/determinism, R07-4.3.2
schema-as-data, R07-5.8 CI verification) is implemented and **enforced in CI**, and as of
2026-10-18 there is a **runtime consumer**: the server resolves the configured pack set at
startup, exposes its effective digest through `guardrail://policy-packs`, and can attribute
an enforcement decision to the exact pack rule that governs a tool (R07-4.4.1–R07-4.4.4).
The rows that stay open are named below and none of them is reachability.
**Priority:** Important; prerequisite to distributing policy changes safely.
**Related:** `docs/specs/guardrail-gaps-2026/02-semantic-content-filtering.md`,
`04-multi-agent-safety-policies.md`, and `06-regulatory-compliance-mapping.md`.
**Prior art reviewed:** private `guardrail-policy-packs` repository. Its core
policies, pin-by-version model, overlay approach, and trust-boundary document
inform this proposal; its domain packs are explicitly empty v0 seeds, not
production policies.

## 1. Problem

The six guardrail-gap specs assume shared configuration and policy files, but
there is no shipped `guardrails.yaml` loader and policy consumers are not
consistently wired. Separately, policy text can contain instructions or
configuration that agents may mistake for authority merely because it was
loaded. Policy distribution therefore needs both reproducible versioning and
an explicit trust boundary.

This spec proposes a governance and resolution contract. It does not claim
that the private policy-pack repository is currently consumed by this
project.

## 2. Goals and non-goals

### Goals

1. Make the effective policy set reproducible from immutable, reviewable
   inputs.
2. Allow domain and project overlays without silently weakening mandatory
   core controls.
3. Reject malformed, unpinned, incompatible, or unauthorized policy inputs
   before they affect a running guardrail engine.
4. Make every effective policy decision traceable to pack identity,
   revision, overlay, and rule identifier.
5. Treat policy-pack contents as data. Text in a pack must never acquire
   instruction authority merely because an agent reads it.

### Non-goals

- Defining the semantic meaning or thresholds of every security rule; the
  individual capability specs own those.
- Automatically installing or executing a pack from an arbitrary URL.
- Treating signatures as proof that policy is safe or correct.
- Replacing human review of changes to mandatory core policy.
- Shipping game, infrastructure, or web domain rules as part of this spec.

## 3. Terminology

- **Core pack:** reviewed mandatory baseline policy set.
- **Domain pack:** reviewed extension for a domain; cannot weaken core rules
  by default.
- **Project overlay:** local additions or explicitly reviewed exceptions.
- **Lock record:** immutable resolved identities and digests for every input
  used to produce the effective policy set.
- **Effective policy set:** deterministic result of resolving core, domain,
  and project inputs under this spec.

## 4. Requirements

### 4.1 Source identity and resolution

1. A consumer MUST identify each pack by repository identity and immutable
   commit or content digest. A floating branch or tag alone MUST NOT be
   accepted for a production lock record.
2. Resolution MUST be deterministic: identical pack bytes, resolver version,
   and overlay MUST produce the same effective policy digest.
3. A lock record MUST include pack name, version label (if present), immutable
   revision, digest, resolver version, overlay digest, and resolution time.
4. Resolution MUST fail if a declared dependency is absent, incompatible, or
   cyclic. The error MUST identify the dependency path.
5. The consumer MUST be able to inspect the resolved policy set without
   executing policy text.

### 4.2 Precedence and mandatory controls

1. Core rules are mandatory by default. A domain pack or project overlay MUST
   NOT remove, disable, or lower the severity of a core rule unless a named
   exception is explicitly allowlisted and reviewed by an authorized human.
2. Additive rules MUST compose deterministically. Conflicting rules MUST
   resolve using a documented deterministic rule; ambiguity MUST fail
   closed rather than silently choosing the weaker action.
3. Every exception MUST record the affected rule, scope, reason, approving
   identity, expiry/review date, and resulting effective-policy digest.
4. An empty domain pack is valid only when explicitly marked as a seed; it
   MUST NOT be represented as having domain coverage.

### 4.3 Trust and parsing boundary

1. Pack text and metadata MUST be parsed as data only. No prompt, shell
   command, script, template expression, or network reference embedded in a
   pack may execute during resolution.
2. Pack schemas MUST reject unknown critical fields and invalid types. Errors
   MUST identify the file and field path without dumping secrets.
3. Pack acquisition MUST use an explicit configured source. Arbitrary paths,
   URLs, redirects, or transitive dependencies supplied by a policy document
   MUST NOT be fetched implicitly.
4. If signature verification is enabled, the trusted-key set MUST be supplied
   out of band. A signature failure MUST block resolution. A valid signature
   MUST NOT be treated as semantic approval.
5. The consumer MUST record whether a resolved pack is verified, merely
   pinned, or locally modified; these states MUST not be conflated.

### 4.4 Runtime activation and auditability

1. The engine MUST expose the effective policy digest and lock-record
   identity at startup and through a read-only diagnostic surface.
2. A policy update MUST build and validate a candidate policy set before
   activation. Failed resolution or validation MUST leave the prior active
   set unchanged.
3. Every enforcement decision MUST be attributable to the active digest and
   rule ID. Secret-bearing configuration values MUST not appear in the audit
   record.
4. Policy resolution and runtime enforcement MUST be separate operations;
   resolving a pack MUST not itself imply that its controls are active.

## 5. Acceptance criteria

The implementation is acceptable when all of the following are reproducible:

1. A fixture with core + domain + overlay resolves to a deterministic lock
   record; resolving it twice produces the same effective digest.
2. Changing a pinned commit or one byte of pack content changes the digest.
3. Floating references, missing dependencies, cycles, malformed schema, and
   conflicting weakening overlays are rejected with actionable errors.
4. A project overlay cannot disable a mandatory core rule without a matching
   reviewed exception; an expired exception is rejected.
5. A pack containing shell-looking or prompt-injection-looking text is
   returned as inert data and causes no process execution or instruction
   override.
6. A failed candidate update leaves the currently active digest unchanged.
7. An enforcement test identifies the active digest and exact rule ID that
   produced its decision.
8. CI verifies the checked-in lock record matches the declared pack
   revisions and content digests.

The test harness, fixtures, lock schema, resolver, and runtime integration do
not yet exist. This section specifies acceptance; it is not evidence that any
criterion currently passes.

## 6. Dependencies and sequencing

- Requires a decision on the core pack repository identity, versioning policy,
  trusted maintainers, and exception approvers.
- Integrates with content-filter, injection, multi-agent, and compliance
  policies only after those engines load and enforce their effective rules.
- Should be implemented before automatic policy-pack consumption is enabled
  in production.
- Does not block fixing or documenting existing guardrail code that does not
  consume packs.

## 7. Open decisions

1. Which immutable source and release process are authoritative for the core
   pack: the private `guardrail-policy-packs` repository or a new canonical
   location?
2. Which signature system and key-rotation process, if any, are required?
3. Are project overlays permitted in production, and who approves exceptions?
4. Is runtime hot reload required, or is restart-on-policy-change the initial
   supported lifecycle?
5. Which policy formats are supported initially? The initial implementation
   should choose one declarative format and reject executable/template forms.

## 8. Traceability

| Related spec | Relationship |
|--------------|--------------|
| 01 Prompt Injection Defense | Supplies policy/rule configuration after resolver is implemented |
| 02 Semantic Content Filtering | Supplies taxonomy policies and per-category actions |
| 04 Multi-Agent Safety Policies | Supplies validator-chain and conflict policies |
| 06 Regulatory Compliance Mapping | Supplies mappings only; pack presence is not compliance evidence |

**Implementation status (original proposal):** no resolver, lock format, pack
signature verification, or production consumer existed when this spec was written.
Superseded in part by the dated section below.

## Implementation status (2026-10-18) — lock record enforced; runtime unchanged

**What landed.** `scripts/check_policy_pack_lock.py` is the single decision for in-repo
policy-pack resolution and lock verification, wired into `regression-guard.yml` as
"Policy-pack lock record (Spec 07 R07-4.1 / R07-5.8)". It replaces
`scripts/validate-policy-packs.mjs`, which **no workflow ever invoked** (the repository's own
audit recorded that) and which **passed vacuously** — with zero packs and `{}` in `LOCK.json`
it printed `lock ok` and exited 0, because its expected-lock loop iterated over an empty set.
That is the same failure class as the F001 tool-count gate and the R19-11 docs gate in this
repository. The `.mjs` file is retained as a delegating shim so the two cannot diverge.

**Rows now enforced:**

- **R07-4.1.2 / R07-4.1.5 / R07-4.1.3** — resolution is deterministic (the effective policy
digest is recomputed and re-derived in the same run; `resolved_at` is metadata and is
excluded from it), inspectable without executing policy text, and the lock record carries
pack name, version, content digest, resolver version, overlay digest and resolution time.
- **R07-4.3.2** — packs are parsed as inert data and schema-checked: unknown top-level or rule
fields fail (an explicit `x-` annotation prefix is allowed), bad types fail, non-semver
versions fail, a `pack` name that disagrees with its own path fails, duplicate rule ids fail.
- **R07-4.2.4** — an empty rule set is valid only with `seed: true`.
- **R07-5.8** — CI verifies the checked-in lock against the bytes on disk, in both
directions (a changed pack with no re-lock fails; a lock entry with no pack fails), and a
missing/unreadable/unrecognised lock record fails rather than skipping.
- **Rule→tool references** — every rule's `tool` must be a tool the server actually
registers; a rule naming a nonexistent tool can never fire.
- **Non-vacuity** — zero discovered packs, or zero known tools, is a FAILURE.

**Evidence.** `scripts/test_check_policy_pack_lock.py` **26/26**, including the two vacuity
cases that the old verifier failed (zero packs; unrecognised legacy lock shape), byte-drift
and version-drift without re-lock, extra/stale lock entries, malformed `resolved_at`, unknown
critical field, bad semver, path/name mismatch, blocking rule without evidence, unknown tool,
duplicate rule id, non-`stop` `on_unknown`, empty non-seed pack, determinism across two
resolutions, and — separately — that a workflow actually invokes the gate. Live on the tree:
`POLICY PACK LOCK: OK` (1 pack, 4 rules, `effective_digest sha256:980f2d48…`). Neighbours
green: action pins 83, tool count 39/22/61, secret inventory 20/16, doc links 965/455, doc
length 455/10 declared, `regression_check --all --pre-commit` exit 0.

**Lock format change.** `policy-packs/LOCK.json` moved from the flat
`{ path: { version, sha256 } }` map to the R07-4.1.3 shape (`resolver_version`,
`effective_digest`, `packs.<path>.{pack, version, digest, resolver_version, overlay_digest,
resolved_at}`). No other consumer of the old shape existed in the tree (grep-verified), so
this is a deliberate one-time re-lock, not a silent break.

**Still absent, and why this spec stays PARTIAL rather than CLOSED:**

- **R07-4.1.4** — no inter-pack dependency graph or cycle detection: only one core pack
ships and it declares no dependencies.
- **R07-4.2.1–R07-4.2.3** — no precedence engine and no exception ledger; there is nothing
for a domain pack or overlay to weaken yet.
- **R07-4.3.4** — no signature verification and no trusted-key set.
- **R07-4.3.5 / R07-4.3.3** — the lock records pinned-vs-verified state as `overlay_digest:
none` and does not acquire remote packs; acquisition posture is moot until a remote source is
chosen.
- **R07-4.4 is now IMPLEMENTED (2026-10-18).** `mcp-server/internal/guardrails/policy_pack.go`
is the consumer: `LoadPolicySet` resolves a configured pack directory, re-derives the
effective digest with the **same canonical form** the CI gate uses (so the runtime and the
gate cannot disagree about which set they are looking at), and refuses: a missing lock
record, a content digest that changed without a re-lock, a lock entry with no pack, an empty
pack set, an empty non-seed pack, a non-`stop` `on_unknown`, an unknown critical field, and a
rule naming a tool the server does not register. `PolicyPackManager.ActivateFrom` is
candidate-first and atomic — a failed candidate leaves the previously active set in force
(R07-4.4.2), never "no policy". `guardrail://policy-packs` reports `not_configured` /
`active` / `failed` with the digest, packs, rules and per-tool attribution, so an absence
never reads as an empty-but-active policy (R07-4.4.1). Resolution and enforcement stay
separate operations: nothing in the loader enforces anything (R07-4.4.4), and attribution is
a lookup the enforcement path performs (R07-4.4.3). Pack text is decoded as inert data and
never executed (R07-4.3.1 / acceptance 5).
- **R07-4.4 named limit, stated rather than implied:** the server *reports* attribution; it
  does not yet *refuse* a tool call for lack of a covering rule. Making the four-laws rules
  blocking for their named tools is a behavioural change to the tool surface and stays an
  owner decision, not a silent narrowing of what the software does.

**Owner decisions still open (Sec. 7):** the authoritative core-pack source and release
process, whether signature verification is required, overlay permissions and exception
approvers, hot-reload vs restart-on-change, and whether the pack format is extended beyond
the current declarative JSON.