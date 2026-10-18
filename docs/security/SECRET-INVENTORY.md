# Secret inventory (Spec 17 R17-10)

**Purpose.** R17-10: *"Maintain a secret inventory (owner, consumer, source,
expiry, last rotation, revocation procedure) without values."* This file is that
inventory. It records **what** each secret is, **who** owns and consumes it,
**where** it comes from, **when** it expires, **how** it is rotated and **how**
it is revoked — and **no values**.

**No values.** Every cell below is a policy or a mechanism, never credential
material. Do not paste a secret, a digest, a token or a private key into this
file. The coverage gate (`scripts/gates/check_secret_inventory.py`) fails if a
secret-bearing environment name discovered in the tree is missing here, and
fails if a cell contains an assignment of the form `NAME=<value>`.

**Source column.** `<NAME>_FILE` means a read-only mounted secret file resolved
by `internal/secretfile` (regular file, no group/world bits, fail-closed; see
the Spec 17 Sec. 10 note). `<NAME>` (no suffix) means the inline environment source.
Inline wins when both are supplied.

**Owner column.** `deployment owner` is the operator who holds the deployment
(Spec 17 Sec. 2: no separate operations team assumed). `security owner` is the
reviewer accountable for the secret's posture.

| Secret | Owner | Consumer | Source | Expiry | Last rotation | Revocation |
|---|---|---|---|---|---|---|
| `CREDENTIAL_VERIFIER_KEY` | security owner | `internal/auth` registry loader — HMAC key behind every stored credential verifier | `CREDENTIAL_VERIFIER_KEY_FILE` (0600) or `CREDENTIAL_VERIFIER_KEY` | No fixed expiry (long-lived); rotate within 24h of suspected disclosure | operator-recorded in the deployment secret store; never a repo value | Generate a new >=256-bit key out of band, redeploy, re-issue every stored verifier under the new key |
| `CREDENTIAL_REGISTRY` | security owner | `internal/auth` — scoped credential registry (verifiers, never raw credentials) | `CREDENTIAL_REGISTRY_FILE` (0600) or `CREDENTIAL_REGISTRY` | <=90 days per registered credential (Spec 16); <=24h planned overlap | operator-recorded per entry | Remove or expire the entry; the in-process reload denies it within `auth.RevocationPropagationBound` (60s) on both transports |
| `JWT_SECRET` | security owner | `internal/auth` — JWT signing material | `JWT_SECRET_FILE` (0600) or `JWT_SECRET` | `JWT_ROTATION_HOURS` (default 168h / 7 days) | operator-recorded | Stage the replacement, keep both valid for the bounded overlap, then retire the old key so it no longer verifies |
| `MCP_API_KEY` | security owner | `internal/web` + `internal/mcp` — legacy shared transport key (migration surface) | `MCP_API_KEY_FILE` (0600) or `MCP_API_KEY` | Bounded by `LEGACY_ACCEPTANCE_UNTIL` (unset = never retires) | operator-recorded | Set or pass `LEGACY_ACCEPTANCE_UNTIL`; both transports refuse the legacy key after the deadline with no rebuild |
| `IDE_API_KEY` | security owner | `internal/web` — legacy IDE transport key (migration surface) | `IDE_API_KEY_FILE` (0600) or `IDE_API_KEY` | Bounded by `LEGACY_ACCEPTANCE_UNTIL` (unset = never retires) | operator-recorded | Same removal switch as `MCP_API_KEY`; refused on both transports once the deadline passes |
| `DB_PASSWORD` | deployment owner | `internal/database` — PostgreSQL app-role credential | `DB_PASSWORD_FILE` (0600) or `DB_PASSWORD` | operator-set | operator-recorded | Rotate in the database, then redeploy; nonlocal profiles additionally require `DB_SSLMODE=verify-full` (R17-04) |
| `REDIS_PASSWORD` | deployment owner | `internal/cache` — Redis credential | `REDIS_PASSWORD_FILE` (0600) or `REDIS_PASSWORD` | operator-set | operator-recorded | Rotate in Redis, then redeploy; nonlocal profiles additionally require `REDIS_USE_TLS=true` (R17-04) |
| `TLS_KEY_PATH` | deployment owner | `cmd/server` — private key for the deployment TLS material | mounted file (0600) | Certificate/key validity period | operator-recorded at each certificate renewal | Replace certificate and key together; validate the trust chain before cutover (R17-03) |
| `TLS_CERT_PATH` | deployment owner | `cmd/server` — server certificate + chain for the deployment TLS material | mounted file | Certificate validity period | operator-recorded at each certificate renewal | Reissue and redeploy the chain; never a substitute for TLS termination proof (R17-03) |
| `TEAM_ENCRYPTION_KEY` | security owner | `internal/team` — AES-GCM key that seals team assignments at rest (`.teams/*.json`, SEC-007 / R17-06) | `TEAM_ENCRYPTION_KEY_FILE` (0600) or `TEAM_ENCRYPTION_KEY` | No fixed expiry; rotate on suspicion | operator-recorded | Re-encrypt affected team secrets under the new key before retiring the old one |
| `FALLBACK_API_KEY` | security owner | `internal/mcp` vision tools — fallback vision provider key | `FALLBACK_API_KEY_FILE` (0600) or `FALLBACK_API_KEY` | Provider-defined | operator-recorded | Revoke provider-side, then redeploy with the replacement |
| `OPENAI_API_KEY` | security owner | `internal/guardrails` content filter — moderation backend key | `OPENAI_API_KEY_FILE` (0600) or `OPENAI_API_KEY` | Provider-defined | operator-recorded | Revoke provider-side, then redeploy with the replacement |
| `MIGRATION_DSN` | deployment owner | `mcp-server/scripts/run_migrations.go` — one-shot migration job database URL | `MIGRATION_DSN_FILE` (0600, preferred) | operator-set | operator-recorded | Rotate the DB password embedded in the URL; argv is refused by construction |
| `MIGRATION_DATABASE_URL` | deployment owner | `mcp-server/scripts/run_migrations.go` — documented migration fallback when no file is mounted | `MIGRATION_DATABASE_URL` (environment fallback) | operator-set | operator-recorded | Rotate the DB password embedded in the URL; the argv interface is removed |
| `WEBHOOK_ENCRYPTION_KEY` | security owner | `internal/database` webhook store — AES-256-GCM key that seals per-webhook signing material at rest (Spec 17 R17-06) | `WEBHOOK_ENCRYPTION_KEY_FILE` (0600) or `WEBHOOK_ENCRYPTION_KEY` | No fixed expiry; rotate on suspicion | operator-recorded; required for nonlocal/production, optional for `local` | Generate a new >=256-bit key, re-seal affected rows under the new key, then retire the old one |
| `WEBHOOK_HMAC_SECRET` | security owner | `internal/notifications` webhook dispatcher — per-webhook signing material (`secret_hmac` column) | database column, **sealed with `WEBHOOK_ENCRYPTION_KEY`** (AES-256-GCM, `v1:` form) when that key is configured | operator-set per webhook | operator-recorded per webhook | Rotate via `rotate_webhook_key` (staged: the outgoing key stays valid for the 24h bounded overlap, identified to receivers by fingerprint) or cut over immediately |

## Non-env secrets covered

`WEBHOOK_HMAC_SECRET` has no environment source; it persists as the
`secret_hmac` column of `webhook_configs` (with the rotation state
`secret_hmac_previous` / `secret_hmac_rotated_at`, migration 019). It is listed
here because R17-10 names webhook material explicitly. Rotation is implemented
(Spec 17 R17-10 webhook half, 2026-10-17): a staged replacement keeps the
outgoing key valid only inside the 24h bounded overlap and identifies the active
key to receivers by a non-reversible fingerprint. Encrypted-at-rest storage is
implemented (Spec 17 R17-06, 2026-10-17): the store seals both signing columns
with `WEBHOOK_ENCRYPTION_KEY` (AES-256-GCM, versioned `v1:` form) on write and
decrypts on read; a nonlocal/production deployment must set that key, and a
sealed value read with no key — or a tampered one — is refused rather than used
as a signing key. The coverage gate discovers *environment* secrets from the
tree; this row is documented but not tree-discoverable, which is the named limit
of the coverage check below.

## Named limits

- The coverage gate proves *no secret-bearing environment name in the tree is
  absent from this inventory*. It cannot discover a secret that is hard-coded in
  a Go literal (none are, per the R17-11/R17-08 gates and the secret scanner),
  nor a non-environment secret such as `WEBHOOK_HMAC_SECRET`.
- No secret is environment-only. `TEAM_ENCRYPTION_KEY` accepted a file
  source on 2026-10-18 (`TEAM_ENCRYPTION_KEY_FILE`, 0600), which closed the
  last gap R17-06 named; `FALLBACK_API_KEY` and `OPENAI_API_KEY` carry
  `FALLBACK_API_KEY_FILE` / `OPENAI_API_KEY_FILE`.
- **The team key now actually protects team data.** Until 2026-10-18 the key
  was documented here while nothing constructed the manager it configures:
  `NewEncryptionManager` had zero non-test callers, so `.teams/*.json` was
  written as plaintext at mode `0644`. The wiring now exists (built once at
  startup, handed to every team tool), the file is `0600`, and reading sealed
  data with no key, a wrong key or a tampered value is a hard error rather than
  a silent fall back to the ciphertext. The key remains **optional** at the
  load boundary: a nonlocal profile without it logs a WARN naming what is
  unprotected instead of refusing to start, because making it mandatory is a
  deliberate breaking change pending an owner decision.
- Webhook signing material (`WEBHOOK_HMAC_SECRET`) is encrypted at rest when
  `WEBHOOK_ENCRYPTION_KEY` is configured (required for nonlocal/production). A
  legacy plaintext row remains readable during migration, and the store reports
  a sealed-vs-plaintext distinction rather than assuming the stronger form.
