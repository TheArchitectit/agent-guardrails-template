# OpenSpec: Versioned Policy-Pack Governance

**Status:** PARTIAL — the lock-record half (R07-4.1 identity/determinism/dependencies, R07-4.3.2
schema-as-data, R07-5.8 CI verification) is implemented and **enforced in CI**, and as of
2026-10-18 there is a **runtime consumer**: the server resolves the configured pack set at
startup, exposes its effective digest through `guardrail://policy-packs`, and can attribute
an enforcement decision to the exact pack rule that governs a tool (R07-4.4.1–R07-4.4.4).
**R07-4.2.1–R07-4.2.3 (precedence, deterministic composition, exception ledger) are
implemented as of 2026-10-18** — declared tiers, a refusal to weaken a mandatory rule without
a reviewed and unexpired exception, and composition that can only strengthen. **R07-4.3.4
(signature verification + out-of-band key set) and R07-4.3.5 (verified / pinned /
locally-modified, never conflated) are implemented as of the same date**, reachable through
`guardrail://policy-packs`. What stays open is R07-4.3.3 — remote acquisition, which is moot
until an authoritative remote source is chosen — plus the named limits recorded below. None
of the remainder is a missing code path.
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

**Evidence.** `scripts/test_check_policy_pack_lock.py` **39/39**, including the two vacuity
cases that the old verifier failed (zero packs; unrecognised legacy lock shape), byte-drift
and version-drift without re-lock, extra/stale lock entries, malformed `resolved_at`, unknown
critical field, bad semver, path/name mismatch, blocking rule without evidence, unknown tool,
duplicate rule id, non-`stop` `on_unknown`, empty non-seed pack, determinism across two
resolutions, and — separately — that a workflow actually invokes the gate. Live on the tree:
`POLICY PACK LOCK: OK` (1 pack, 4 rules, `effective_digest sha256:5d3674c2…` — re-locked when the
packs gained a declared `tier`, which is a content change and therefore a deliberate re-lock). Neighbours
green: action pins 83, tool count 39/22/61, secret inventory 20/16, doc links 965/455, doc
length 455/10 declared, `regression_check --all --pre-commit` exit 0.

**Lock format change.** `policy-packs/LOCK.json` moved from the flat
`{ path: { version, sha256 } }` map to the R07-4.1.3 shape (`resolver_version`,
`effective_digest`, `packs.<path>.{pack, version, digest, resolver_version, overlay_digest,
resolved_at}`). No other consumer of the old shape existed in the tree (grep-verified), so
this is a deliberate one-time re-lock, not a silent break.

**Still absent, and why this spec stays PARTIAL rather than CLOSED:**

- **R07-4.1.4 is now IMPLEMENTED (2026-10-18).** Packs declare `depends_on` entries
(`{pack, range}`); resolution fails when a declared dependency is absent, when the resolved version is outside the
declared range, or when the graph is cyclic — and every error names the dependency **path**
(`core/domain -> core/base`, or `core/a -> core/b -> core/a`), because "resolution failed" is not actionable.
Ranges are deliberately few: an exact `MAJOR.MINOR.PATCH`, or `>=`/`^`/`~` before one; empty, `*`, `latest` and
anything else is refused rather than read as "any", since a constraint that silently widens accepts a breaking change
with no review. Edges are **declared, never inferred** from rule or tool overlap — an inferred edge would change
meaning whenever a rule moved. The walk runs in two places over the same rules: `resolveDependencies` in the runtime
loader and `check_dependency_graph` in `scripts/check_policy_pack_lock.py`, whose accepted field sets are now pinned
against the Go structs by `SchemaParityTest` — the Go source claimed a "pack schema test" kept the two in step and no
such test existed, so the claim was made true rather than deleted. The checked-in tree remains a legal **one-node**
graph (zero edges reported, `guardrails.PackEdge` list empty) and the same walk runs on it, so the check is not
skipped because the tree is small. **Evidence:** 9 new Go tests (satisfied edge, absent dependency naming the path,
cycle naming the closing path, incompatible version, six unusable range forms, self-dependency, zero-edge graph,
`SatisfiesVersionRange` table) + the request-path test
`TestPolicyPackStatus_ReportsResolvedDependencyGraph` reading `guardrail://policy-packs`; 10 new Python cases plus 4
parity assertions, red-on-drift proven (disabling `check_dependency_graph` flips 6 cases; reverting `PACK_FIELDS`
flips the parity test). A dependency graph that must include every pack would make a one-pack tree untestable, so the
fixtures carry the edges the shipped tree does not.
- **R07-4.2.1–R07-4.2.3 are now IMPLEMENTED (2026-10-18).** Before this pass a pack carried
an `enforcement` class and nothing else: there was no declared tier, so "core rules are
mandatory by default" described a tree that could not express core-ness. Every pack was
interchangeable, and a second pack declaring a core rule id could lower its severity or
retarget its tool with nothing in the resolver positioned to notice — the exception ledger
did not exist either, so a weakening had no reviewed form. The only outcomes were "not
expressible" and "silent", and the first was false.
  - **Tier is declared, then checked against the name.** Packs carry `tier` (`core` |
    `domain` | `overlay`) and the value must equal the pack name's prefix, so a domain policy
    cannot be filed as core (or the reverse) by moving a file; the precedence claim appears in
    the diff instead of being inferred from a directory. An absent or unknown tier is
    **refused** rather than defaulted — defaulting to core would grant override authority,
    defaulting to overlay would accept an unchecked weakening.
  - **Precedence follows the tier, never discovery order.** The effective declaration of a
    rule id is the one from the highest tier that declares it. Two packs of **equal** tier
    declaring one id are refused as ambiguous: with no deterministic winner, choosing by file
    name would make the outcome depend on spelling. Strengthening a core rule needs no
    exception — the requirement only restricts weakening, and a control that cannot be
    strengthened is unusable in the only safe direction.
  - **A weakening is a severity drop or a retargeted tool.** Both are refused without an
    allowlisted exception; both name the rule, the two severities (or the two tools) and the
    overriding pack, because "policy invalid" is not actionable. Retargeting counts at equal
    severity: the core rule stops governing the tool it was written for.
  - **Composition takes the stronger action, and there is no weaker branch.** Where more than
    one effective rule governs a tool, the maximum enforcement wins and is written into the
    attribution map; every collision is reported with winner and loser. Equal severities are
    ordered by rule id so the *audit identity* is deterministic — not the action, which is
    identical.
  - **Every exception carries the six R07-4.2.3 facts plus a binding.** Rule, overriding pack,
    scope, reason, approver, `expires` (RFC3339) and `resulting_rule_set_digest`. Validation is
    three ordered passes so the message names the right failure: completeness/expiry, then the
    digest binding, then **coverage in both directions** — a weakening with no exception (a
    control silently shed) and an exception with no weakening (a permanent hole recorded as
    diligence). An expired, stale, duplicated, incomplete or unused exception is refused. A
    malformed ledger is an error, not an empty ledger: a typo must not silently re-permit a
    weakening, nor silently drop a review that exists.
  - **Granting an exception is necessarily two steps, and the tooling says so.**
    `PolicyRuleSetDigest` computes the digest of a set whose ledger does not yet authorise
    it, so an approver binds to a value produced by the resolver rather than a hand-computed
    one that can never match. Whether that set is *activateable* stays a separate question
    answered by `LoadPolicySet`. `LoadPolicySetAt` injects the clock so expiry is a proven
    fact rather than a wait.
  - **Reachable where an operator looks.** `guardrail://policy-packs` reports
    `rule_set_digest`, `applications` (one row per effective rule id with `base_enforcement`,
    `effective_enforcement`, `weakened`, `exception_scope`, `declared_by`), `conflicts` and
    `exceptions` — read from the resolved snapshot, never re-derived in the handler, because a
    surface that recomputes precedence can disagree with the set the server enforces.
  - **Evidence.** `policy_precedence_test.go` 13 tests (thirteen cases incl. seven
    missing-field subtests) driving the REAL loader over REAL packs and a REAL lock: weakening
    refused, retargeting refused, strengthening allowed, equal-tier ambiguity refused,
    tier/path disagreement refused, undeclared tier refused, stronger rule wins composition,
    a complete exception allows and is recorded, expired refused, stale binding refused, unused
    refused, incomplete refused for each of the seven fields, malformed ledger refused, and the
    shipped tree resolving as a single core pack. Plus the request-path
    `TestPolicyPackStatus_ReportsPrecedenceAndExceptions` reading the registered handler.
    **Red-on-drift proven four ways** (weakening undetected; weaker rule winning a conflict;
    expiry unchecked; digest binding unchecked), and the first is now a permanent
    `mutationkill` case confirmed killing.
  - **The shipped tree declares `tier: core`** and is re-locked; the gate's schema gained
    `tier` with the same name-prefix rule, so the Go loader and the CI gate still accept
    exactly the same packs. `SchemaParityTest` pins `packFields`/`ruleFields` on both sides.
  - **Named limits, recorded rather than implied.** The exception approver is a recorded
    identity, not an authenticated one — nothing verifies that `approver: roger` was written by
    Roger, so the ledger's authority is repository review (a signed ledger is R07-4.3.4 work).
    The `resulting_rule_set_digest` binds to the effective rule set, not to the pre-precedence
    input set, so re-signing a pack without changing the resulting decisions does not
    invalidate an exception. Nothing yet refuses a tool call for lack of a covering rule: the
    four-laws rules stay advisory, which remains an owner decision.
- **R07-4.3.4 is now IMPLEMENTED (2026-10-18).** `mcp-server/internal/guardrails/policy_signature.go`
verifies detached Ed25519 signatures over each pack's exact bytes. The trusted-key set is read
**out of band** from `POLICY_PACK_TRUSTED_KEYS_FILE` — never from the pack, the signature file or
the lock — because a key that travels with the artifact it verifies proves only that one party
could write both. Configuring the file **enables** verification: with it set, a pack with no
signature, an unknown key id, a malformed signature or one that does not match the bytes is a
**resolution error**, and a failed candidate leaves the prior active set in force. The key set
itself fails closed — empty set, duplicate key, a declared `key_id` that disagrees with its own
public key, a past `expires`, and a group/world-readable file are all refused. Signing is a
separate operator tool (`SignPolicyPack`), deliberately **not** used at startup: a process that
can sign its own policy can re-author it. **A valid signature is not approval** — a signed pack
that weakens a core rule still requires a reviewed exception, and a test asserts exactly that,
because wiring trust into the approval path would have made signature verification an
authorization grant (gr-xp-03).
- **R07-4.3.5 is now IMPLEMENTED (2026-10-18).** Every resolved pack reports one of three
mutually exclusive states: `verified` (signature checked against the out-of-band set), `pinned`
(bytes match the reviewed lock, no signature verified) and `locally_modified` (bytes are not the
bytes attestation covers). The state is decided at resolution time and carried on the snapshot;
`guardrail://policy-packs` reports it per pack plus a `signature_verification` flag, so `pinned`
cannot be misread as "this deployment has no such feature". Two conflation traps are closed
deliberately: a pack carrying a signature that was **never checked** reports `pinned` and names
**no key id** (an unverified key id is an identity claim), and a missing trust row is not
possible — every pack gets a row, so absence cannot read as "nothing to report". Provenance is
also why `overlay_digest: none` was the wrong place for this fact: one field was carrying both
"no overlay" and "not verified".
- **R07-4.3.3** — acquisition is still local-only: the server reads exactly the configured
directory and follows no URL, redirect or path named inside a policy document. Remote acquisition
remains unimplemented and is moot until an authoritative remote source is chosen (owner).
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