# Spec 17 implementation notes (part 3 of 3)

Companion to
[`17-deployment-tls-and-secret-boundary.md`](17-deployment-tls-and-secret-boundary.md).
(Part 1 is `17-implementation-notes.md`; part 2 is
`17-implementation-notes-2.md`. The notes are split so that no Markdown file
exceeds the 500-line limit — Spec 19 R19-11.)

## 17. Implementation note (2026-10-17) — R17-10 / R16-09 registered-credential validity bound

R17-10 requires webhook/JWT/TLS/DB/Redis rotation **and** adopts Spec 16's
registered-credential lifecycle: *"For registered credentials adopt Spec 16's
proposed <=90-day validity, <=24-hour planned overlap and <=60-second tested
denial of revoked keys on both transports."* Spec 16 Sec. 4 states the same bound:
*"new credentials expire within 90 days."*

Two of the three bounds were already enforced and pinned — the <=24h planned
overlap (`MaxRotationOverlap`, `TestRegistryRotationOverlapBounds`) and the
<=60s revocation denial (`RevocationPropagationBound`,
`TestRegistryRevocationPropagationBound`). The **validity bound was enforced
nowhere**: `buildSnapshot` validated scopes, verifier shape and uniqueness but
never inspected `IssuedAt`/`ExpiresAt`. A record with no `expires_at` was an
unbounded grant that never expires, and a record could declare an arbitrarily
long window — the exact posture the 90-day bound exists to forbid. The
`CREDENTIAL_REGISTRY` row of `docs/security/SECRET-INVENTORY.md` already stated
`<=90 days per registered credential (Spec 16)`, so the inventory claimed a
bound the code did not implement.

**Mechanism.** `internal/auth/registry.go` gains
`MaxCredentialValidity = 90 * 24 * time.Hour` (published, so a silent change
fails the build) and enforces it inside `buildSnapshot`, the single choke point
every load path shares (`New`, `LoadFromSources`, `LoadFromSourcesEx`,
`Reload`, `ReloadFromSources`, `RotateVerifierKey`):

- a record with a zero `ExpiresAt` is **refused** — an unbounded credential is
  not a permitted state;
- the declared window is measured from `IssuedAt` when supplied, otherwise from
  the load time, and a window longer than the maximum is refused.

Because the check lives in `buildSnapshot`, a bad registry fails **closed**: the
whole snapshot is rejected and the prior state is left intact on the reload and
rotation paths, so a rejected replacement never installs an over-long
credential and never enlarges access.

**Evidence.** `internal/auth/registry_validity_test.go`:
`TestRegistryRequiresExpiry` (no expiry refused, error names `expires_at` and
R17-10), `TestRegistryRejectsOverLongValidity` (a window past the maximum
refused; exactly at the bound accepted), `TestRegistryValidityWindowMeasuredFromIssuedAt`
(a long-lived record re-issued recently is accepted; a stale issuance whose
window runs past the bound is refused), `TestRegistryValidityBoundAppliesToReloadAndRotate`
(the bound holds on the reload and rotation paths and the prior snapshot
survives a rejected replacement), and `TestMaxCredentialValidityPublished` (the
published bound is pinned). All existing registry fixtures now declare a bounded
window. New CI row `security-matrix-credential-validity`
(`.github/workflows/team-validation.yml`); new mutation-kill case
`credential-validity-bound-removed` — verified killed (harness **32 cases, all
killed**).

**Named limit (documented, not hidden).** The bound is enforced at the load
boundary for a registry that is configured; the unconfigured (absent) registry
keeps the documented bounded legacy mode (Spec 16 Sec. 4) and is unaffected. DB/JWT/
TLS staged rotation remains open under R17-10 (the webhook half landed
2026-10-17; see part 2 Sec. 15).
