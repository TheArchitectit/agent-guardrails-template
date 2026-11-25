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
