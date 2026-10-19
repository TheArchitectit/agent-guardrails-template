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

**Update 2026-10-19 — the JWT rotation/expiry bound is now enforced.** Of the
remaining R17-10 rotation halves (DB/JWT/TLS), the JWT *posture* is now bounded:
`Config.ValidateJWTPosture` (`internal/config/jwt.go`) is called from
`Config.Validate` and refuses `JWT_ROTATION_HOURS` above `MaxJWTRotationWindow`
(168h, the audited bound) and refuses a `JWT_EXPIRY` longer than the rotation
window ("a token must not outlive the rotation of the key that signed it"). This
closes the open MEDIUM row 9 of `docs/security/security-audit-config.md`. **The
staged-replacement/cutover half of R17-10 for JWT, DB and TLS remains open** — an
operator still has no staged-rotation path for those, only for webhook signing
material. Honest limit recorded in-repo: no JWT is minted or verified anywhere in
this tree today (no JWT library is a dependency; agent/session tokens are opaque
and database-backed), so this bounds a declared posture, not a live token path.

## 18. Implementation note (2026-10-19) — R17-10 JWT rotation/expiry bound (audit row 9)

`JWT_ROTATION_HOURS` (default 168h) and `JWT_EXPIRY` (default 15m) were declared
environment settings that `ValidateTimeout` never touched: the shutdown, request,
DB and circuit-breaker timers were bounded, the JWT pair was not. An operator
could set any rotation window at all and startup accepted it, and a
hand-built `Config` with both fields at zero was accepted too ("rotate never").
This is the exact finding of `docs/security/security-audit-config.md` row 9
("no mechanism for emergency rotation").

**Mechanism.** `internal/config/jwt.go`:

- `MaxJWTRotationWindow = 168 * time.Hour` (published, so a silent change fails
the build) is the maximum rotation window;
- `EffectiveJWTRotationWindow` / `EffectiveJWTTTL` default a non-positive value
to the documented default, so an unset posture is bounded rather than unbounded;
- `ValidateJWTPosture` refuses a rotation window over the bound and a TTL that
exceeds the rotation window, naming R17-10 in both errors.

It is called from `Config.Validate`, so the posture gates startup.

**Evidence.** `internal/config/jwt_posture_test.go`: unset adopts the default and
passes; defaults pass; the window at the bound (168h) passes; a window over the
bound is refused naming R17-10; a TTL longer than the rotation window is refused;
`TestMaxJWTRotationWindowPublished` pins the bound; and
`TestValidateCouplesJWTPostureToStartup` proves the gate runs inside `Validate`.
**Red-on-drift proven** (disabling the `Validate()` call flips the coupling test
PASS→FAIL; restored).
