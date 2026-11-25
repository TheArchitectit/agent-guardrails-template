# TPL-39 verification

Daemonless Compose v2 configuration and image pin checks passed. The build
context resolves to the MCP module, both published ports use IPv4 loopback,
and startup waits for healthy PostgreSQL and Redis. The runtime uses the
local deployment profile.

The launcher downloads the approved source archive at
`ba9ff630b24821507d6c93c6c6117ac7be4e9b1e` and verifies SHA-256
`4f09f073c0e0c3c2706365c4444a75282040efb7c44178b777b1d4a1c81f2ffa`.
PowerShell 7.4 parsed the launcher. Mock startup and restart used the actual
archive and daemonless Compose rendering. Restart preserved the synthetic
env file with Windows line endings, and the launcher kept generated values
out of its output. Mock checks rejected an altered archive, a nonlocal
Docker context, an ambient credential override, and a conflicting container
before startup.

## Target Windows laptop pilot — 2026-11-25 (America/Chicago)

The launcher at repository revision `21f55f558d90decf6d95b2fa36238a7c15a78894`
was run with PowerShell 7.6.9 on Windows against Docker Desktop's local
`desktop-linux` context (Windows named pipe), WSL 2 Linux Engine 29.9.3, and
Compose 5.7.1. It staged verified source revision
`ba9ff630b24821507d6c93c6c6117ac7be4e9b1e`, generated its synthetic
credentials, pulled the pinned dependencies, and built the MCP image.

The first launcher run exited 1. PostgreSQL and Redis became healthy, but MCP
restarted repeatedly after a PostgreSQL DNS lookup failure. The Windows
readiness request ended prematurely. Inspection found MCP attached to Compose's
`_default` network while PostgreSQL and Redis were attached to
`_guardrails-net`. Both active host bindings were verified on loopback at
`127.0.0.1:8080` and `127.0.0.1:8081`. Clean launcher acceptance remains open.

During recovery, a manual network connect reported `network sandbox ... not
found` while MCP was restarting. Docker retained the association, and MCP
connected to PostgreSQL on its next start. MCP then stopped on a fresh-database
schema error caused by missing `schema_migrations` (SQLSTATE 42P01), with schema version
19 required. The pinned launcher workflow currently has no one-shot migration
job before MCP startup, although the source tree contains
`mcp-server/scripts/run_migrations.go`. The runtime operator applied the
reviewed migration job to the synthetic pilot database.

After applying migrations on the synthetic database, all 19 clean schema
versions were present, but the pinned-source readiness endpoint returned
HTTP 503. Its report showed config, database, cache, and schema as PASS, with
`STARTING` as the sole required non-PASS. Inspection of pinned
`cmd/server/main.go` found `EvaluateReadiness()` requires lifecycle state
`READY` before that state can be set, creating a circular startup gate. A
published-code fix is still required. A local-only derivative image was used
to continue operating the pilot, and its result is recorded separately from
the pinned launcher/Compose result.

The local derivative image `guardrail-mcp:tpl39-local-readiness` was built
from the staged pinned MCP module with exactly two Go edits to evaluate
candidate READY using the full checks before publishing the READY state. The
local recovery overlay is
`%LOCALAPPDATA%\GuardrailsLaptopPilot\local-readiness-tpl39-ba9ff630b24821507d6c93c6c6117ac7be4e9b1e.compose.yml`.
It assigned that image to MCP and attached it to `guardrails-net`. PostgreSQL and Redis
stayed running while Compose recreated MCP. Final independent
verification at 09:55:15 Central on 2026-11-25 reported body `READY`,
`ready: true`, and all five required checks PASS. Docker inspection reported
all three services `running healthy`. Docker port inspection and Windows
listener checks confirmed `127.0.0.1:8080` and `127.0.0.1:8081`; both TCP
connections were accepted, and an independent GET to `/health/ready` returned
HTTP 200.

The migration ledger contained 19 clean rows (001–019) with 19 distinct
checksums. Named volumes `guardrails-laptop-pilot_pg_data` and
`guardrails-laptop-pilot_redis_data`, created at 09:45:35 Central, remain
mounted and preserved. The recovered pilot stack is running. Pinned-launcher
acceptance and teardown remain open.

## Local implementation and isolated verification plan — 2026-11-25

The local source now assigns PostgreSQL, Redis, the one-shot `migrate` service,
and MCP to the same explicit `guardrails-net`. The migration job invokes the
checked-in `scripts/run_migrations.go` binary built into the MCP image; it
receives its DSN from the runtime environment, waits for healthy PostgreSQL,
and MCP waits for successful job completion as well as healthy PostgreSQL and
Redis. The root service keeps its established default container names.

Startup now evaluates the full readiness dependency report using candidate
READY lifecycle semantics before publishing `READY`. The public readiness
handler still evaluates the actual lifecycle, so it remains non-2xx while the
server is STARTING and returns HTTP 200 only after READY is published.

Daemonless Compose rendering and policy inspection passed with the verification
overlay: all four services share `guardrails-net`, migration ordering and DSN
wiring are present, ports use `127.0.0.1:18080` and `127.0.0.1:18081`, and
container names are distinct. OpenSpec strict validation and PowerShell parser
validation passed. The Windows host has no Go toolchain, so focused Go tests
were run in a pinned Go 1.26.11 container with the checkout mounted read-only;
`go test ./internal/config ./internal/web` passed.

The isolated Windows acceptance procedure is in `docs/laptop-compose-pilot.md`
and uses project `guardrails-tpl39-verify`, overlay
`docker-compose.tpl39-verification.yml`, and a fresh synthetic env file. The
standalone launcher remains pinned to its previously published validated
revision. Its source pin and archive SHA-256 must be updated only in a
follow-up after the implementation commit is actually public and the archive
hash is computed from that immutable revision.

## Isolated Windows acceptance — 2026-11-25

Before the successful run, the first isolated build caught an ambiguous
`/app/migrations` binary path: the source tree already contains a directory of
that name, so the final image entrypoint resolved to a directory. The binary
now builds to `/app/run-migrations`, the build verifies it is executable, the
final image copies it to `/run-migrations`, and Compose overrides the image
entrypoint. A network-isolated `--help` invocation confirmed the copied
executable runs.

The clean `up --build` run used explicit `desktop-linux`, project
`guardrails-tpl39-verify`, distinct container names, ports 18080/18081, and
fresh project-scoped volumes. Its synthetic PostgreSQL password included URI
reserved punctuation. The migration service exited 0; its ledger had 19 clean
rows through version 019, 19 distinct checksums, and no dirty rows. MCP and
PostgreSQL/Redis health checks were healthy. The readiness endpoint returned
HTTP 200 with lifecycle `READY`, `ready: true`, and seven checks. Published
ports were confirmed on IPv4 loopback only. This reserved-character run also
revealed that MCP's separate connection URL builder needed percent-encoded
userinfo; that fix and its focused evidence are in the separate
`encode-postgres-connection-credentials` OpenSpec change (TPL-42).

Preflight and post-run inspection showed the recovered original project
containers retained their baseline IDs and healthy states, and its two named
volumes retained their original creation times and project labels. The original
project was not changed. The successful isolated candidate and its volumes are
left running for independent review. The parent independently confirmed
healthy status, migration completion, schema ledger, loopback publications,
HTTP 200 readiness, and unchanged original containers/volumes at 10:44:56 CT.

After that review, `docker compose down` was run only for
`guardrails-tpl39-verify`, without `--volumes`. Its Postgres and Redis volume
names, creation timestamps, and Compose labels were unchanged. The exact same
project/env inputs were then restarted with `up --build`; the migration job
reported no pending migrations, verified its six required tables, and exited
0. The ledger remained 19 clean rows through 019 with 19 distinct checksums.
The restarted MCP and dependencies were healthy, and readiness again returned
HTTP 200 with lifecycle `READY`. The original project remained unchanged. The
standalone launcher now pins published source revision
`2d71e714ae06a6a7b639cf16bd388b6736143104` with archive SHA-256
`cd7756d0e6baa84af622faa20a1f20cc0d180e8b1eb4df97a0dc1b75accc879f`.
The prior env revision is explicitly `ba9ff630b24821507d6c93c6c6117ac7be4e9b1e`;
mock upgrade and fail-closed controls pass. The parent verified the exact public
codeload archive hash (3,732,332 bytes). The exact repinned launcher run remains
open until the launcher change is published and a coordinated rollback window
is approved. The recovered original project and volumes remain untouched.
