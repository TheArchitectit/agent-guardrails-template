# Canonical sprint plan: Runtime Integrations

> Planning baseline only. CLOSED means the canonical spec records exercised closure; this plan schedules regression/evidence review, not reimplementation. UNKNOWN means no status or closure table is recorded: first inspect implementation and evidence, then propose a narrowly scoped change only if a gap is confirmed. Do not infer status from archived task checkboxes.

## Operating model

- **Models:** use Luna (`gpt-6-luna`) for bounded implementation, test, and evidence preparation; use Astra (`gpt-6-astra`) for design decisions, security-sensitive review, and final sign-off. These are recommendations, not execution evidence.
- **Time:** per-spec ranges below are person-hours for the listed scope. Estimates exclude external approvals, credentials, unavailable services, and unrelated repair.
- **Platform applicability:** names identify where a requirement applies. Applicability is not a claim of current runtime compatibility. Source-only, CI, and server requirements are editor-neutral unless named.
- **Acceptance:** retain the canonical SHALL clauses and named scenarios as the acceptance source. Record exact commit, command/tool versions, output hashes where useful, and failures/skips.

## Planned work

<a id="agent-install-path-preservation"></a>
### [agent-install-path-preservation](../../../openspec/specs/agent-install-path-preservation/spec.md) — UNKNOWN

- **Disposition:** baseline status unknown; inspect code, tests, and current workflow evidence first; create a narrowly scoped implementation change only for a reproduced gap.
- **Scope:** Installer preserves hidden destination components; Installer confines writes to configured root.
- **Steps:** map each requirement to current source and focused tests; run or add bounded checks for every scenario below. Keep fixes scoped to a confirmed gap, preserve interfaces, and retain fail-closed behavior.
- **Scenario acceptance:** install a hidden Claude skill — an installation request targets `.claude/skills/guardrails-enforcer.json` under a configured root; the resulting file is placed beneath the root's `.claude/skills` directory with the hidden component preserved; destination contains parent traversal — a requested destination resolves outside the configured root; the installer reports failure and writes no file outside that root.
- **Evidence:** record exact revision, commands, tool versions, scenario outcomes, and logs or hashes in the owning change/evidence file; explicitly mark unavailable environments and failed checks. UNKNOWN remains UNKNOWN until source-backed implementation and test evidence justify a status update.
- **Ownership:** Platform (lead); Security (boundary review); QA (acceptance). **Coding-platform scope:** Claude Code, Copilot, Cursor, OpenCode, Windsurf (installer destination behavior)
- **Models/time:** Execution and evidence: Luna (1–2h); lead decision and sign-off: Astra (0.5–1h). Estimate excludes unavailable approvals or infrastructure.

<a id="ide-client-initialization"></a>
### [ide-client-initialization](../../../openspec/specs/ide-client-initialization/spec.md) — UNKNOWN

- **Disposition:** baseline status unknown; inspect code, tests, and current workflow evidence first; create a narrowly scoped implementation change only for a reproduced gap.
- **Scope:** Initialize VS Code configuration before requests.
- **Steps:** map each requirement to current source and focused tests; run or add bounded checks for every scenario below. Keep fixes scoped to a confirmed gap, preserve interfaces, and retain fail-closed behavior.
- **Scenario acceptance:** activation uses configured URL and secret — workspace configuration contains a non-default server URL and SecretStorage contains an API key; activation performs its initial readiness check; the request uses the configured URL and bearer key, and a successful response marks the extension connected; initial connection fails — the activation request has initialized configuration but the stubbed server is unreachable; the connection check completes; the extension reports disconnected and remains activated without an uninitialized-config error.
- **Evidence:** record exact revision, commands, tool versions, scenario outcomes, and logs or hashes in the owning change/evidence file; explicitly mark unavailable environments and failed checks. UNKNOWN remains UNKNOWN until source-backed implementation and test evidence justify a status update.
- **Ownership:** Frontend (lead); Platform (integration review); QA (acceptance). **Coding-platform scope:** VS Code extension only
- **Models/time:** Execution and evidence: Luna (1–2h); lead decision and sign-off: Astra (0.5–1h). Estimate excludes unavailable approvals or infrastructure.

<a id="ide-rest-endpoint"></a>
### [ide-rest-endpoint](../../../openspec/specs/ide-rest-endpoint/spec.md) — UNKNOWN

- **Disposition:** baseline status unknown; inspect code, tests, and current workflow evidence first; create a narrowly scoped implementation change only for a reproduced gap.
- **Scope:** IDE clients default to the server REST listener.
- **Steps:** map each requirement to current source and focused tests; run or add bounded checks for every scenario below. Keep fixes scoped to a confirmed gap, preserve interfaces, and retain fail-closed behavior.
- **Scenario acceptance:** fresh IDE configuration — an IDE integration has no saved endpoint override; it requests health or IDE validation from a default local server; it sends the REST request to port 8081 and does not use the MCP transport port; explicit endpoint override — a user configures a non-default server URL; the IDE sends a REST request; it uses the configured URL instead of replacing it with the default; VS Code Configure waits for stored credentials — the VS Code Configure command saves a non-default REST URL and SecretStorage resolves asynchronously; Configure probes endpoint readiness; it waits until the updated URL and stored credential are loaded and sends its first probe to the configured endpoint.
- **Evidence:** record exact revision, commands, tool versions, scenario outcomes, and logs or hashes in the owning change/evidence file; explicitly mark unavailable environments and failed checks. UNKNOWN remains UNKNOWN until source-backed implementation and test evidence justify a status update.
- **Ownership:** Frontend (lead); Platform (integration review); QA (acceptance). **Coding-platform scope:** VS Code, JetBrains, Vim, Neovim clients named by the spec
- **Models/time:** Execution and evidence: Luna (2–4h); lead decision and sign-off: Astra (0.5–1h). Estimate excludes unavailable approvals or infrastructure.

<a id="ide-selection-contract"></a>
### [ide-selection-contract](../../../openspec/specs/ide-selection-contract/spec.md) — CLOSED

- **Disposition:** regression and evidence review; keep closed unless new contradictory evidence appears.
- **Scope:** IDE selection payloads use a canonical field.
- **Steps:** map each requirement to current source and focused tests; run or add bounded checks for every scenario below. Keep fixes scoped to a confirmed gap, preserve interfaces, and retain fail-closed behavior.
- **Scenario acceptance:** canonical selection request — a request contains a non-empty `selection` field; the selection-validation handler processes it; it validates that selected text; legacy client migration — a request contains non-empty legacy `code` and no `selection` field; the handler processes it; it validates the legacy selected text during migration; conflicting fields — a request contains different non-empty values for `selection` and `code`; the handler processes it; it returns HTTP 400 without selecting either value; empty selection — neither field contains non-empty selected text; the handler processes the request; it returns HTTP 400.
- **Evidence:** record exact revision, commands, tool versions, scenario outcomes, and logs or hashes in the owning change/evidence file; explicitly mark unavailable environments and failed checks. UNKNOWN remains UNKNOWN until source-backed implementation and test evidence justify a status update.
- **Ownership:** Frontend (lead); Platform (integration review); QA (acceptance). **Coding-platform scope:** VS Code, JetBrains, Vim, and Neovim selection clients plus the MCP REST handler
- **Models/time:** Execution and evidence: Luna (1–2.5h); lead decision and sign-off: Astra (0.5–1h). Estimate excludes unavailable approvals or infrastructure.

<a id="local-docker-compose-pilot"></a>
### [local-docker-compose-pilot](../../../openspec/specs/local-docker-compose-pilot/spec.md) — UNKNOWN

- **Disposition:** baseline status unknown; inspect code, tests, and current workflow evidence first; create a narrowly scoped implementation change only for a reproduced gap.
- **Scope:** Root Compose resolves the MCP build from the module directory; Laptop pilot requires a ready local Docker environment; Pilot credentials stay synthetic and outside the build context; Local pilot network publications remain loopback-only; Application startup waits for healthy dependencies; Pilot teardown preserves named data volumes; Standalone Windows launcher uses a pinned source archive; Windows launcher is repeatable without replacing pilot data; Windows launcher preserves credentials across source updates; Windows launcher does not replace credentials for existing data; Windows launcher targets only the local Docker Desktop Linux engine; Launcher checks effective local policy before startup.
- **Steps:** map each requirement to current source and focused tests; run or add bounded checks for every scenario below. Keep fixes scoped to a confirmed gap, preserve interfaces, and retain fail-closed behavior.
- **Scenario acceptance:** Compose configuration is evaluated from the repository root — the user is in the repository root; the root Compose configuration is resolved; the MCP build context is `./mcp-server` and the Dockerfile path is `deploy/Dockerfile` within that context; Preflight runs before startup — a user is preparing the laptop pilot on Windows; Docker CLI and Compose v2 preflight checks run; both client and server availability and a Compose v2 version are confirmed before any service is started; New private env file is created — the selected user-local env file does not exist; the pilot credential generation step runs; it creates a UTF-8 file without a byte order mark outside the repository and emits no credential values; Existing env file is preserved — the selected user-local env file already exists; the credential generation step runs; it stops without changing that file or printing credentials; Pilot stack starts with local-only host bindings — Compose starts the pilot with the generated env file; the MCP service is created; its deployment profile is `local`, its local bridge attestation is explicit, and both host port publications bind to `127.0.0.1`; Dependencies become ready before the application — PostgreSQL or Redis is still unhealthy; Compose evaluates the MCP service startup dependency; the MCP service is not started until both dependencies are healthy; Database migrations complete before the application starts — PostgreSQL is healthy but the pilot database is not at the schema version required by MCP; Compose starts the pilot application; a one-shot job from the same versioned MCP image applies and verifies the checked-in migrations through the required schema version using a protected runtime environment source, and MCP starts only after successful completion; Pilot services share the isolated application bridge — the root Compose pilot is rendered; Compose creates PostgreSQL, Redis, the migration job, and MCP; all four services attach to the same explicitly declared `guardrails-net` bridge and host publications remain loopback-only; Application readiness is checked — the services have started; the pilot readiness check requests `http://127.0.0.1:8081/health/ready`; readiness succeeds only when the endpoint returns HTTP 200; Startup dependency checks do not depend on published lifecycle readiness — all startup dependencies and required schema checks pass while the application lifecycle is `STARTING`; the application evaluates whether it can transition to `READY`; it evaluates those dependency checks independently of the published readiness state, transitions to `READY`, and continues to require HTTP 200 for the readiness endpoint; Default teardown retains disposable data — the laptop pilot is running with named data volumes; the documented default teardown command runs; containers and network stop while both volumes remain available for a later restart; Standalone source bootstrap — the user has PowerShell, Docker Desktop, and network access to the official GitHub archive host but no repository checkout or Git installation; the standalone launcher runs; it downloads the archive for revision `2d71e714ae06a6a7b639cf16bd388b6736143104`, verifies SHA-256 `cd7756d0e6baa84af622faa20a1f20cc0d180e8b1eb4df97a0dc1b75accc879f` before extraction, and installs the Compose file and MCP build context under `%LOCALAPPDATA%\GuardrailsLaptopPilot`; Launcher reruns with its own pilot state — the versioned source and valid synthetic env file created by this launcher already exist; the user runs the launcher again; it reuses the existing source and credentials, does not print secret values, and leaves named volumes available; Existing credentials or containers are not owned by the launcher — the dedicated env file is malformed or a fixed-name container belongs to a different Compose project; the launcher checks local pilot state; it stops without replacing credentials or taking over the container; Source revision update preserves credentials for existing volumes — the prior pinned revision has a valid launcher-owned env file and its Compose project volumes exist; the launcher starts from a newly pinned revision; it validates the prior file, creates the new revision's env file without overwriting either file, preserves all credential values, and uses the new source revision for the image build; Existing volumes without validated credentials fail closed — project volumes exist but no valid current or prior launcher env file is available; the launcher prepares pilot credentials; it stops without generating new credentials or changing the volumes; Docker Desktop is stopped — the `desktop-linux` context resolves to Docker Desktop's local Linux named pipe but the engine is not ready; the launcher starts the pilot; it starts the standard Docker Desktop application and waits up to five minutes for the local Linux engine before Compose actions; Rendered configuration matches pilot policy — ambient variables cannot override the selected synthetic env file; the launcher validates Compose configuration; it confirms the expected build revision, synthetic credential sources, local deployment profile, local bridge attestation, and ports 8080 and 8081 on `127.0.0.1` before `up`; Started pilot is ready on loopback — Compose health-aware startup succeeds; the launcher checks the active bindings and readiness endpoint; both services ports map to IPv4 loopback and `http://127.0.0.1:8081/health/ready` returns HTTP 200.
- **Evidence:** record exact revision, commands, tool versions, scenario outcomes, and logs or hashes in the owning change/evidence file; explicitly mark unavailable environments and failed checks. UNKNOWN remains UNKNOWN until source-backed implementation and test evidence justify a status update.
- **Ownership:** Platform (lead); Security (boundary review); QA (acceptance). **Coding-platform scope:** Local Docker / MCP service; editor-neutral
- **Models/time:** Execution and evidence: Luna (2–4h); lead decision and sign-off: Astra (0.5–1h). Estimate excludes unavailable approvals or infrastructure.

<a id="mcp-docker-context"></a>
### [mcp-docker-context](../../../openspec/specs/mcp-docker-context/spec.md) — UNKNOWN

- **Disposition:** baseline status unknown; inspect code, tests, and current workflow evidence first; create a narrowly scoped implementation change only for a reproduced gap.
- **Scope:** MCP build context excludes local-only files.
- **Steps:** map each requirement to current source and focused tests; run or add bounded checks for every scenario below. Keep fixes scoped to a confirmed gap, preserve interfaces, and retain fail-closed behavior.
- **Scenario acceptance:** local secret and cache are present — the build context contains a benign `.env` marker, `.git` data, and a local cache fixture; the Dockerfile builder executes `COPY . .`; those local-only paths are unavailable to the builder and no marker content is printed; required build inputs are present — the production MCP build uses the declared context; the builder copies source inputs; `go.mod`, `go.sum`, Go source, `web/`, and `.guardrails/prevention-rules` remain available and the representative image build succeeds.
- **Evidence:** record exact revision, commands, tool versions, scenario outcomes, and logs or hashes in the owning change/evidence file; explicitly mark unavailable environments and failed checks. UNKNOWN remains UNKNOWN until source-backed implementation and test evidence justify a status update.
- **Ownership:** Platform (lead); Security (boundary review); QA (acceptance). **Coding-platform scope:** MCP Docker image builder; editor-neutral
- **Models/time:** Execution and evidence: Luna (1–2h); lead decision and sign-off: Astra (0.5–1h). Estimate excludes unavailable approvals or infrastructure.

<a id="pi-mcp-sdk-development"></a>
### [pi-mcp-sdk-development](../../../openspec/specs/pi-mcp-sdk-development/spec.md) — UNKNOWN

- **Disposition:** baseline status unknown; inspect code, tests, and current workflow evidence first; create a narrowly scoped implementation change only for a reproduced gap.
- **Scope:** Development checks have the MCP SDK available; The MCP bridge remains optional at runtime.
- **Steps:** map each requirement to current source and focused tests; run or add bounded checks for every scenario below. Keep fixes scoped to a confirmed gap, preserve interfaces, and retain fail-closed behavior.
- **Scenario acceptance:** clean development install — the committed Pi package manifest and lockfile; a developer runs `npm ci` and the package test suite; SDK-present bridge tests can import the declared SDK subpaths; production install omits development dependencies — a packed production install contains no MCP SDK; the extension loads its standalone tools and checks bridge availability; standalone loading succeeds and the bridge is reported unavailable.
- **Evidence:** record exact revision, commands, tool versions, scenario outcomes, and logs or hashes in the owning change/evidence file; explicitly mark unavailable environments and failed checks. UNKNOWN remains UNKNOWN until source-backed implementation and test evidence justify a status update.
- **Ownership:** Platform (lead); QA (acceptance). **Coding-platform scope:** Pi extension only
- **Models/time:** Execution and evidence: Luna (1–2h); lead decision and sign-off: Astra (0.5–1h). Estimate excludes unavailable approvals or infrastructure.

<a id="pi-package-identity"></a>
### [pi-package-identity](../../../openspec/specs/pi-package-identity/spec.md) — UNKNOWN

- **Disposition:** baseline status unknown; inspect code, tests, and current workflow evidence first; create a narrowly scoped implementation change only for a reproduced gap.
- **Scope:** Pi install instructions identify the published package.
- **Steps:** map each requirement to current source and focused tests; run or add bounded checks for every scenario below. Keep fixes scoped to a confirmed gap, preserve interfaces, and retain fail-closed behavior.
- **Scenario acceptance:** user follows a clean-install command — a user reads the README or installer help; the user copies the published-package install command; it names `@architectit/pi-guardrails` and resolves to the package whose manifest has that name; local installer help is checked — `install.mjs --help` runs with a temporary home directory; the help text is compared with package metadata; the package identity matches and no files are installed or removed.
- **Evidence:** record exact revision, commands, tool versions, scenario outcomes, and logs or hashes in the owning change/evidence file; explicitly mark unavailable environments and failed checks. UNKNOWN remains UNKNOWN until source-backed implementation and test evidence justify a status update.
- **Ownership:** Platform (lead); QA (acceptance). **Coding-platform scope:** Pi extension only
- **Models/time:** Execution and evidence: Luna (1–2h); lead decision and sign-off: Astra (0.5–1h). Estimate excludes unavailable approvals or infrastructure.

<a id="platform-skill-file-destinations"></a>
### [platform-skill-file-destinations](../../../openspec/specs/platform-skill-file-destinations/spec.md) — UNKNOWN

- **Disposition:** baseline status unknown; inspect code, tests, and current workflow evidence first; create a narrowly scoped implementation change only for a reproduced gap.
- **Scope:** Single-file platform skills install at their declared file path; Directory-based destinations remain directories.
- **Steps:** map each requirement to current source and focused tests; run or add bounded checks for every scenario below. Keep fixes scoped to a confirmed gap, preserve interfaces, and retain fail-closed behavior.
- **Scenario acceptance:** install Windsurf rules individually — the user installs the `windsurf-rules` skill into a target root; the rules are written as a regular `.windsurfrules` file directly beneath that root; install a directory-based skill — a skill declares a platform destination directory; its source file is placed within that directory using the platform's expected relative path.
- **Evidence:** record exact revision, commands, tool versions, scenario outcomes, and logs or hashes in the owning change/evidence file; explicitly mark unavailable environments and failed checks. UNKNOWN remains UNKNOWN until source-backed implementation and test evidence justify a status update.
- **Ownership:** Platform (lead); QA (acceptance). **Coding-platform scope:** Windsurf single-file rules; directory-based skills for the configured platform targets
- **Models/time:** Execution and evidence: Luna (1–2h); lead decision and sign-off: Astra (0.5–1h). Estimate excludes unavailable approvals or infrastructure.

<a id="postgres-connection-credential-encoding"></a>
### [postgres-connection-credential-encoding](../../../openspec/specs/postgres-connection-credential-encoding/spec.md) — UNKNOWN

- **Disposition:** baseline status unknown; inspect code, tests, and current workflow evidence first; create a narrowly scoped implementation change only for a reproduced gap.
- **Scope:** PostgreSQL URL credentials are encoded as URI userinfo.
- **Steps:** map each requirement to current source and focused tests; run or add bounded checks for every scenario below. Keep fixes scoped to a confirmed gap, preserve interfaces, and retain fail-closed behavior.
- **Scenario acceptance:** Ordinary credentials keep the established URL — the PostgreSQL username and password contain only ordinary unreserved characters; the configuration layer builds its connection URL; it returns the existing URL format and query parameter order; Reserved characters round-trip through URL parsing — the PostgreSQL password contains URI-reserved punctuation; the configuration layer builds its connection URL and a PostgreSQL URL parser reads it; parsing succeeds and recovers the original username and password exactly.
- **Evidence:** record exact revision, commands, tool versions, scenario outcomes, and logs or hashes in the owning change/evidence file; explicitly mark unavailable environments and failed checks. UNKNOWN remains UNKNOWN until source-backed implementation and test evidence justify a status update.
- **Ownership:** Backend (lead); QA (acceptance). **Coding-platform scope:** Server configuration; editor-neutral
- **Models/time:** Execution and evidence: Luna (1–2h); lead decision and sign-off: Astra (0.5–1h). Estimate excludes unavailable approvals or infrastructure.

<a id="team-cli-backend-distribution"></a>
### [team-cli-backend-distribution](../../../openspec/specs/team-cli-backend-distribution/spec.md) — CLOSED

- **Disposition:** regression and evidence review; keep closed unless new contradictory evidence appears.
- **Scope:** Installed CLI locates its packaged backend; Explicit backend override is preserved.
- **Steps:** map each requirement to current source and focused tests; run or add bounded checks for every scenario below. Keep fixes scoped to a confirmed gap, preserve interfaces, and retain fail-closed behavior.
- **Scenario acceptance:** invoke an installed command outside the checkout — a user runs an installed team CLI command from a directory outside the source checkout; the CLI invokes the backend installed with the CLI without requiring a working-directory-relative source path; provide a backend override — `TEAM_MANAGER_PATH` names an available backend; the CLI uses that backend before its installation-owned default.
- **Evidence:** record exact revision, commands, tool versions, scenario outcomes, and logs or hashes in the owning change/evidence file; explicitly mark unavailable environments and failed checks. UNKNOWN remains UNKNOWN until source-backed implementation and test evidence justify a status update.
- **Ownership:** Backend (lead); QA (acceptance). **Coding-platform scope:** Team CLI; editor-neutral
- **Models/time:** Execution and evidence: Luna (0.5–1.5h); lead decision and sign-off: Astra (0.5–1h). Estimate excludes unavailable approvals or infrastructure.

<a id="vscode-extension-linting"></a>
### [vscode-extension-linting](../../../openspec/specs/vscode-extension-linting/spec.md) — UNKNOWN

- **Disposition:** baseline status unknown; inspect code, tests, and current workflow evidence first; create a narrowly scoped implementation change only for a reproduced gap.
- **Scope:** Lint the VS Code TypeScript source in the package lifecycle.
- **Steps:** map each requirement to current source and focused tests; run or add bounded checks for every scenario below. Keep fixes scoped to a confirmed gap, preserve interfaces, and retain fail-closed behavior.
- **Scenario acceptance:** Normal package test lifecycle — locked package dependencies are installed; a maintainer runs `npm test` from the VS Code extension package; compilation and configured linting pass before the activation test runs; Invalid TypeScript source — a temporary TypeScript source file contains invalid syntax; the package lint command analyzes `src`; ESLint reports a parser error and exits unsuccessfully; Parseable TypeScript rule violation — a temporary TypeScript source file parses but declares an unused binding; the package lint command analyzes `src`; ESLint reports the unused-binding rule and exits unsuccessfully.
- **Evidence:** record exact revision, commands, tool versions, scenario outcomes, and logs or hashes in the owning change/evidence file; explicitly mark unavailable environments and failed checks. UNKNOWN remains UNKNOWN until source-backed implementation and test evidence justify a status update.
- **Ownership:** Frontend (lead); Platform (integration review); QA (acceptance). **Coding-platform scope:** VS Code extension only
- **Models/time:** Execution and evidence: Luna (2–4h); lead decision and sign-off: Astra (0.5–1h). Estimate excludes unavailable approvals or infrastructure.
