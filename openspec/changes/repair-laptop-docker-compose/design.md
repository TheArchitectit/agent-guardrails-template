# Design

## Context

See proposal.md for the motivation. The root Compose file currently resolves `mcp-server/deploy/Dockerfile` from repository-root context, while that Dockerfile expects its context root to contain `go.mod`, `cmd/server`, `web`, and `.guardrails`. The runtime contract already defaults to `DEPLOYMENT_PROFILE=local`, uses the isolated bridge attestation, publishes only MCP and Web to loopback, and gates the MCP service on healthy Postgres and Redis.

The pilot is for a Windows laptop with Docker Desktop using its WSL 2 backend. This executor is Linux without an accessible container daemon, so only daemonless configuration and static checks can be claimed here; Windows startup remains unrun.

## Goals / Non-Goals

**Goals:**
- Make the standard root `docker compose` build resolve the existing Dockerfile from its expected module root.
- Provide safe and repeatable laptop setup steps that keep synthetic disposable credentials outside the build context and avoid volume deletion by default.
- Preserve existing image pins, local deployment profile and host loopback publications.

**Non-Goals:**
- Fix the separate TPL-33 build-context allowlist / `.dockerignore` finding.
- Change the production/remote deployment profile or claim OPS closure.
- Install Docker Desktop, start containers, or exercise runtime health in this cloud executor.

## Decisions

- Set the root service build context to `./mcp-server` and Dockerfile to `deploy/Dockerfile`. This is the smallest path correction that satisfies the Dockerfile's `COPY go.mod`, `COPY . .`, and subsequent build inputs; changing Dockerfile paths would couple its internal copy logic to repository layout.
- Keep the root compose entry point and pinned Redis/Postgres images, local profile/attestation, loopback host ports, service network, and healthy dependency ordering. This retains the existing profile contract and avoids creating a second pilot compose variant.
- Add a Windows PowerShell runbook that stores generated credentials under `%LOCALAPPDATA%`, passes the file explicitly with `--env-file`, validates with `docker compose config --quiet`, checks service health before probing `http://127.0.0.1:8081/health/ready`, and tears down with `docker compose down` without `-v`. This avoids OpenSSL requirements, terminal output of generated credentials, committing a `.env`, and accidental deletion of disposable data. Use .NET `UTF8Encoding(false)` with a create-new file stream and `StreamWriter.WriteLine` so Windows PowerShell 5.1 writes UTF-8 without a BOM and Windows newlines, and an existing env file cannot be overwritten or appended to.
- Add a single-file PowerShell launcher that can run without Git or a checkout. It fetches the exact validated GitHub archive revision, checks its SHA-256 before extracting only `docker-compose.yml` and `mcp-server/` into a short transient child under `%LOCALAPPDATA%\GuardrailsLaptopPilot`, then moves those files to a revision-named immutable source directory. The generated env file has a separate revision-specific name and uses create-new semantics; reruns validate and reuse it without printing values. The launcher rejects ambient Compose overrides, validates `desktop-linux` against Docker Desktop's exact local Linux named pipe, starts the standard Desktop app if needed, and refuses container-name collisions owned by another Compose project.
- Keep all generated credentials and rendered Compose configuration in process memory or the Local AppData directory. Before startup, require quiet config validation and internally verify the local profile, attestation, exact source build argument, expected credential wiring, and both IPv4 loopback publications. Start only with `docker --context desktop-linux compose up --wait --wait-timeout 300 -d --build`, check the active port mappings and HTTP readiness, and report only the pinned revision, service statuses, and loopback endpoints. Do not change container modes, Docker contexts, global configuration, or remove volumes.
- Link the runbook from the existing repository index map. Keep the production deployment guide unchanged because its scope and steps are distinct from a laptop pilot.

## Risks / Trade-offs

- [Existing defaults in the Compose file are development placeholders] → The runbook always requires an external env file with fresh synthetic credentials and explicitly rejects using defaults for any non-disposable environment.
- [A downloaded archive could be altered or point at moving content] → The launcher pins a full commit URL and checks the previously validated archive SHA-256 before extraction; no Git checkout or branch resolution is used.
- [A named Docker context could target a remote daemon] → The launcher requires the named `desktop-linux` context to use Docker Desktop's local Windows Linux-engine named pipe and checks the server OS before Compose calls.
- [A rerun could overwrite credentials or attach to another project's containers] → The env file is create-new and strictly validated on reuse; container-name collisions are rejected unless their Compose project label matches this launcher's dedicated project name.
- [The `mcp-server` build context can include unrelated local files] → TPL-33 remains separately tracked and open; this change makes no claim that local build inputs are allowlisted.
- [Compose config validation does not build or start containers] → Windows Docker Desktop startup and readiness must be verified on the target laptop; this work reports it as NOT_RUN.

## Migration Plan

No data migration is needed. Use the root `docker-compose.yml` with a generated local env file. If startup fails, capture non-secret status/errors and run `docker compose down` without volume removal; reverting the root context/Dockerfile path change restores the former build behavior.
