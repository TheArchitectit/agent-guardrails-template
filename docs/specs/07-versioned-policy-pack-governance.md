# OpenSpec: Versioned Policy-Pack Governance

**Status:** Proposed — no policy-pack loader, resolver, or trust verifier is
implemented in this repository.
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

**Implementation status:** proposal only. No resolver, lock format, pack
signature verification, or production consumer is implemented in this repo.