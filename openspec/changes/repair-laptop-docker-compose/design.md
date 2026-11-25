# Design

## Context

See proposal.md for the motivation. The root Compose file currently resolves `mcp-server/deploy/Dockerfile` from repository-root context, while that Dockerfile expects its context root to contain `go.mod`, `cmd/server`, `web`, and `.guardrails`. The runtime contract already defaults to `DEPLOYMENT_PROFILE=local`, uses the isolated bridge attestation, publishes only MCP and Web to loopback, and gates the MCP service on healthy Postgres and Redis.

The pilot is for a Windows laptop with Docker Desktop using its WSL 2 backend. Runtime verification uses a separate Compose project, alternate fixed container names, distinct loopback ports, and fresh project-scoped volumes; the recovered `guardrails-laptop-pilot` project is never changed as part of isolated acceptance.

## Goals / Non-Goals

**Goals:**
- Make the standard root `docker compose` build resolve the existing Dockerfile from its expected module root.
- Provide safe and repeatable laptop setup steps that keep synthetic disposable credentials outside the build context and avoid volume deletion by default.
- Preserve existing image pins, local deployment profile and host loopback publications.
- Run the checked-in versioned PostgreSQL migration job before MCP and allow a healthy STARTING instance to publish READY without weakening the public readiness endpoint.

**Non-Goals:**
- Fix the separate TPL-33 build-context allowlist / `.dockerignore` finding.
- Change the production/remote deployment profile or claim OPS closure.
- Stop the recovered pilot or remove any of its data volumes.

## Decisions

- Set the root service build context to `./mcp-server` and Dockerfile to `deploy/Dockerfile`. This is the smallest path correction that satisfies the Dockerfile's `COPY go.mod`, `COPY . .`, and subsequent build inputs; changing Dockerfile paths would couple its internal copy logic to repository layout.
- Keep the root compose entry point and pinned Redis/Postgres images, local profile/attestation, loopback host ports, service network, and healthy dependency ordering. This retains the existing profile contract and avoids creating a second pilot compose variant.
- Connect PostgreSQL, Redis, the one-shot migration job, and MCP to `guardrails-net`. Build the supported `run_migrations.go` binary and checked-in migration manifest into the pinned MCP image; pass DSN fields through PG environment settings and use a host-only URL so URI-reserved password characters remain intact. Gate the job on healthy PostgreSQL and MCP on both healthy dependencies plus successful migration completion.
- Evaluate the startup transition using the full readiness checks with candidate READY semantics. The public readiness handler continues to use the actual lifecycle, so STARTING remains HTTP non-2xx until READY is set.
- Add a Windows PowerShell runbook that stores generated credentials under `%LOCALAPPDATA%`, passes the file explicitly with `--env-file`, validates with `docker compose config --quiet`, checks service health before probing `http://127.0.0.1:8081/health/ready`, and tears down with `docker compose down` without `-v`. This avoids OpenSSL requirements, terminal output of generated credentials, committing a `.env`, and accidental deletion of disposable data. Use .NET `UTF8Encoding(false)` with a create-new file stream and `StreamWriter.WriteLine` so Windows PowerShell 5.1 writes UTF-8 without a BOM and Windows newlines, and an existing env file cannot be overwritten or appended to.
- Add a single-file PowerShell launcher that can run without Git or a checkout. It fetches the exact validated GitHub archive revision, checks its SHA-256 before extracting only `docker-compose.yml` and `mcp-server/` into a short transient child under `%LOCALAPPDATA%\GuardrailsLaptopPilot`, then moves those files to a revision-named immutable source directory. Each source pin has a separately named env file with create-new semantics. On a pin update, the launcher validates the immediately prior revision's env file and creates a new file with the same credentials and updated source metadata; if project volumes exist without validated credentials, it stops instead of generating replacements. Prior env files stay untouched. The launcher rejects ambient Compose overrides, validates `desktop-linux` against Docker Desktop's exact local Linux named pipe, starts the standard Desktop app if needed, and refuses container-name collisions owned by another Compose project.
- Keep all generated credentials and rendered Compose configuration in process memory or the Local AppData directory. Before startup, require quiet config validation and internally verify the local profile, attestation, exact source build argument, expected credential wiring, and both IPv4 loopback publications. Start only with `docker --context desktop-linux compose up --wait --wait-timeout 300 -d --build`, check the active port mappings and HTTP readiness, and report only the pinned revision, service statuses, and loopback endpoints. Do not change container modes, Docker contexts, global configuration, or remove volumes.
- Link the runbook from the existing repository index map. Keep the production deployment guide unchanged because its scope and steps are distinct from a laptop pilot.

## Risks / Trade-offs

- [Existing defaults in the Compose file are development placeholders] → The runbook always requires an external env file with fresh synthetic credentials and explicitly rejects using defaults for any non-disposable environment.
- [A downloaded archive could be altered or point at moving content] → The launcher pins a full commit URL and checks the previously validated archive SHA-256 before extraction; no Git checkout or branch resolution is used.
- [A named Docker context could target a remote daemon] → The launcher requires the named `desktop-linux` context to use Docker Desktop's local Windows Linux-engine named pipe and checks the server OS before Compose calls.
- [A rerun could overwrite credentials or attach to another project's containers] → The env file is create-new and strictly validated on reuse; container-name collisions are rejected unless their Compose project label matches this launcher's dedicated project name.
- [The `mcp-server` build context can include unrelated local files] → TPL-33 remains separately tracked and open; this change makes no claim that local build inputs are allowlisted.
- [A source pin changes while a Compose database persists] → The launcher migrates only from the validated immediate prior revision's env file using create-new semantics; without it, existing project volumes make startup fail closed.

## Migration Plan

Use the root `docker-compose.yml` with a generated local env file. For isolated Windows acceptance, use `docker-compose.tpl39-verification.yml`, project `guardrails-tpl39-verify`, and distinct ports. Stop only that verification project with `down` and preserve its volumes for inspection; do not target the recovered project's containers or volumes. The standalone launcher now pins published revision `2d71e714ae06a6a7b639cf16bd388b6736143104` and verified archive SHA-256 `cd7756d0e6baa84af622faa20a1f20cc0d180e8b1eb4df97a0dc1b75accc879f`. Exact launcher acceptance on the recovered pilot remains pending the launcher change being published and a coordinated rollback window.
