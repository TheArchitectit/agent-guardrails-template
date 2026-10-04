# OpenSpec: Secure Cross-Product Evidence Method (Cross-Product Evidence Auth v1)

**Status:** Proposed; not implemented. Defines the security method shared with
AIGGP/DevGate and OpenAgentPlatform. It resolves the blocked principal-mapping
question without inventing authority: identity comes from an explicit
credential registry, never from content or a truncated log hash.
**Related:** [Spec 11](../11-authorization-scopes-and-roles.md),
[Spec 14](../14-optional-oap-guardrail-checker/spec.md),
`platform-current-state.md` §§3, 5.

## Purpose

Define one verifiable method by which two products authenticate each other,
exchange bounded evidence in either direction, and fail closed, without a
shared kernel, shared secret, or a receipt that authorizes an effect. Both
sides implement the same contract with direction-specific credentials.

## Method summary

1. **Instance identity:** every product instance holds a workload identity and
   an asymmetric key pair (Ed25519). The key ID is a domain-separated digest of
   the public key, not a secret and not a log truncation.
2. **Trust roots:** each side is provisioned out of band with the peer public
   keys it accepts, per audience and direction, with validity windows and
   revocation state. Trust is never learned from the peer's own payload.
3. **Transport and artifact auth are separate claims:** transport uses mTLS
   (per-direction client certificates) where a service connection exists;
   every exchanged artifact additionally carries a detached signature. A valid
   transport session never substitutes for artifact verification.
4. **Bounded envelope:** every exchange carries `contract_version`, `direction`,
   producer ID and key ID, consumer audience, tenant/project, subject digest,
   policy/context/evaluator digests, request/evaluation ID, idempotency key,
   nonce, `issued_at`, `expires_at`, and payload digest.
5. **Fail closed:** any missing, unverifiable, expired, replayed, revoked, or
   mismatched field is non-PASS; it never becomes an allow or a mandatory PASS.

## Requirements

### Requirement: Credential-to-principal registry replaces derived identity
<!-- id: gr-xp-01 -->
Guardrails SHALL resolve a caller principal from an explicit credential
registry: each stored credential record identifies credential ID, principal ID,
scopes, issuance time, optional expiry, and revocation state, and verifies the
presented secret with a keyed one-way verifier. The truncated log hash
(`hashAPIKey`) SHALL NOT be used as principal, credential, or tenant identity.
An unrecognized or unregistered credential SHALL be denied privileged actions.

#### Scenario: legacy key without a registry entry
- **WHEN** a caller presents a valid legacy `MCP_API_KEY`/`IDE_API_KEY` that has
  no registry record and no principal mapping
- **THEN** the request is denied for privileged/mutating actions, may be allowed
  only for the explicitly enumerated legacy-safe surface, and the denial is
  audited

#### Scenario: log hash is not identity
- **WHEN** code attempts to derive a principal or tenant from the 8-byte log
  hash
- **THEN** it is rejected in review and no authorization decision uses it

### Requirement: Outbound evidence is signed per instance key
<!-- id: gr-xp-02 -->
Every Guardrails-produced artifact or receipt SHALL be signed with the instance
key over canonical envelope bytes and SHALL bind subject/request digest,
policy/profile digest, check/component identity and version, outcome, reason,
and evidence-manifest digest. The signature SHALL be detached from any canonical
result bytes it attests.

#### Scenario: tampered evidence
- **WHEN** any bound digest or the envelope payload is modified after signing
- **THEN** verification fails and the artifact is non-PASS

### Requirement: Direction-specific least-privilege credentials
<!-- id: gr-xp-03 -->
Each direction SHALL use its own credential with audience, expiry, revocation,
and a read/submit-only scope. No Guardrails integration credential SHALL be able
to mutate peer policy, roles, grants, tenants, credentials, exceptions, required
checks, adoption stage, or effects, and no Guardrails artifact SHALL be treated
as an authorization grant.

#### Scenario: checker PASS with host deny
- **WHEN** a Guardrails artifact reports PASS but the host denies the action
- **THEN** the effect remains denied; the artifact is evidence only

### Requirement: Replay, freshness, and revocation
<!-- id: gr-xp-04 -->
Exchanges SHALL carry unique request/evaluation IDs, a subject-bound idempotency
key, a nonce, and bounded freshness. Consumers SHALL reject expired, replayed,
duplicate side-effecting, or revoked-key artifacts. Revocation SHALL take effect
within the approved bound and SHALL be testable. Offline bundles SHALL include a
revocation snapshot; a snapshot beyond its freshness bound is non-PASS.

#### Scenario: revoked producer key
- **WHEN** an artifact is signed by a key revoked under current revocation state
- **THEN** it is rejected for use even though the signature is mathematically
  valid

### Requirement: No hidden authority in content
<!-- id: gr-xp-05 -->
Prompt text, file content, tool output, policy-pack text, model output, and
peer evidence SHALL remain data. None of them SHALL establish or change a
caller's principal, tenant, scope, role, grant, or permission, and none SHALL
cause a privileged action to be authorized.

#### Scenario: content claims an identity
- **WHEN** content asserts a different principal, tenant, or role
- **THEN** the assertion is ignored for identity and authorization purposes

### Requirement: Both sides implement the same contract
<!-- id: gr-xp-06 -->
Guardrails and each peer product SHALL implement the same versioned envelope and
the four verification stages — transport authentication, artifact integrity,
producer identity, and receiving-policy authorization — as distinct, separately
testable steps. A failure in any stage SHALL be reported with a distinct reason
and SHALL NOT be smoothed into success.

#### Scenario: stages are distinguishable
- **WHEN** transport authenticates but the artifact signature fails
- **THEN** the failure is reported as an integrity failure, not as an
  authentication failure or a success

## Out of scope

- Implementing a peer product's authority, effects, tenants, or credentials.
- A shared kernel, ledger, CA, or universal envelope.
- Treating confirmation flags, signatures, or receipts as authorization.
