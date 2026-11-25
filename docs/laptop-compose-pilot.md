# Windows laptop Compose pilot

This is a local pilot for disposable synthetic data. It is not a production deployment or evidence that production/OPS requirements are closed. The pinned launcher run exposed network, schema migration, and readiness lifecycle gaps. A local-only derivative image and Compose overlay subsequently reached healthy status and HTTP 200, but pinned-source acceptance remains open. See the [TPL-39 evidence](../openspec/changes/repair-laptop-docker-compose/evidence.md).

## Prerequisites

- Windows 10/11 within Docker Desktop's supported lifecycle, with WSL 2 enabled and virtualization available. Use the WSL 2 backend for Linux containers.
- Install Docker Desktop only from [Docker's official Windows installation guide](https://docs.docker.com/desktop/setup/install/windows-install/). Start Docker Desktop and wait for its engine to be ready.
- Windows PowerShell and standard Docker Compose v2 (`docker compose`).
- A free local TCP port 8080 and 8081.

From a fresh repository checkout, open PowerShell at the repository root. Confirm the CLI and engine respond:

```powershell
docker version
docker compose version
```

For a one-command run without installing Git or cloning a repository, run [`scripts/start-laptop-pilot.ps1`](../scripts/start-laptop-pilot.ps1) from the checkout root, or save that single file to the Windows laptop and pass its path to PowerShell:

```powershell
& .\scripts\start-laptop-pilot.ps1
```

The launcher downloads only the Compose file and `mcp-server` tree from the pinned public revision `ba9ff630b24821507d6c93c6c6117ac7be4e9b1e`, checks the archive SHA-256, and installs it under `%LOCALAPPDATA%\GuardrailsLaptopPilot`. It reuses that versioned source and its dedicated synthetic env file on rerun. It preserves the pilot env file and data volumes for restart. If Docker Desktop is installed but its local Linux engine is stopped, the launcher starts the standard Docker Desktop app and waits up to five minutes; it does not switch container modes or select another daemon. It verifies the rendered policy, health-aware startup, active loopback bindings, and HTTP readiness before reporting success. The current running recovery uses the documented local derivative overlay; pinned-source acceptance remains open. See the [pilot evidence](../openspec/changes/repair-laptop-docker-compose/evidence.md).

`docker version` must show both Client and Server sections. If it does not, start or repair Docker Desktop before continuing. Before building, ensure `mcp-server` contains no real credentials or other private files: that directory is the Docker build context. The separate TPL-33 build-context hygiene item remains open, and this pilot does not add a context allowlist.

## Create a private synthetic env file

The generated values below are disposable local credentials. The env file is under the current user's Local AppData, outside the repository and Docker build context. Do not copy them into a tracked file or use this pilot configuration for a real deployment.

```powershell
$PilotDir = Join-Path $env:LOCALAPPDATA 'GuardrailsLaptopPilot'
New-Item -ItemType Directory -Force -Path $PilotDir | Out-Null
$PilotEnv = Join-Path $PilotDir 'pilot.env'
if (Test-Path -LiteralPath $PilotEnv) {
    throw 'Pilot env file already exists. Remove or rename it deliberately before creating a new one.'
}

function New-PilotSecret {
    $bytes = New-Object byte[] 32
    $rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
    try { $rng.GetBytes($bytes) } finally { $rng.Dispose() }
    'Aa0' + [Convert]::ToBase64String($bytes).TrimEnd('=').Replace('+', '-').Replace('/', '_')
}

$envLines = @(
    'DB_USER=guardrail'
    'DB_NAME=guardrails'
    "DB_PASSWORD=$(New-PilotSecret)"
    "REDIS_PASSWORD=$(New-PilotSecret)"
    "MCP_API_KEY=$(New-PilotSecret)"
    "IDE_API_KEY=$(New-PilotSecret)"
    "JWT_SECRET=$(New-PilotSecret)"
    'DEPLOYMENT_PROFILE=local'
    'DEPLOYMENT_BIND_ATTESTATION=isolated-local-bridge:host-loopback-publish'
)
# CreateNew fails safely if a file appeared after the check above.
$stream = [System.IO.File]::Open(
    $PilotEnv,
    [System.IO.FileMode]::CreateNew,
    [System.IO.FileAccess]::Write,
    [System.IO.FileShare]::None
)
$writer = [System.IO.StreamWriter]::new($stream, [System.Text.UTF8Encoding]::new($false))
try {
    foreach ($line in $envLines) { $writer.WriteLine($line) }
} finally {
    $writer.Dispose()
}
```

The script prints no generated values, refuses to overwrite an existing file, and writes UTF-8 without a BOM with Windows line endings. Keep `$PilotEnv` in this PowerShell session for the commands below. Confirm it is outside the checkout if needed with:

```powershell
$PilotEnv
```

## Validate, start, and check readiness

First validate Compose configuration without starting containers. `--quiet` avoids printing interpolated environment values:

```powershell
docker compose --env-file $PilotEnv config --quiet
```

Start the stack from the repository root:

```powershell
docker compose --env-file $PilotEnv up -d --build
```

Wait until Postgres, Redis, and the MCP service each report a healthy container status. The following PowerShell loop prints service names and health states only:

```powershell
$services = @('postgres', 'redis', 'mcp-server')
$deadline = (Get-Date).AddMinutes(5)
do {
    $states = @{}
    foreach ($service in $services) {
        $id = docker compose --env-file $PilotEnv ps -q $service
        if ($id) {
            $states[$service] = docker inspect --format '{{.State.Health.Status}}' $id
        } else {
            $states[$service] = 'missing'
        }
    }
    $states.GetEnumerator() | Sort-Object Name | ForEach-Object { '{0}: {1}' -f $_.Name, $_.Value }
    if (@($states.Values | Where-Object { $_ -ne 'healthy' }).Count -eq 0) { break }
    Start-Sleep -Seconds 5
} while ((Get-Date) -lt $deadline)
if (@($states.Values | Where-Object { $_ -ne 'healthy' }).Count -gt 0) {
    throw 'Compose services did not all become healthy before timeout.'
}
```

Verify that both published ports are loopback-only, then check application readiness:

```powershell
$mcpBind = docker compose --env-file $PilotEnv port mcp-server 8080
$webBind = docker compose --env-file $PilotEnv port mcp-server 8081
if ($mcpBind -notmatch '^127\.0\.0\.1:') { throw 'MCP port is not published on IPv4 loopback.' }
if ($webBind -notmatch '^127\.0\.0\.1:') { throw 'Web port is not published on IPv4 loopback.' }
$response = Invoke-WebRequest -UseBasicParsing http://127.0.0.1:8081/health/ready
if ($response.StatusCode -ne 200) { throw 'Application readiness endpoint did not return HTTP 200.' }
```

The Web readiness endpoint checks the application dependencies. Visit `http://127.0.0.1:8081` for the local UI or use `http://127.0.0.1:8080/mcp` for the local MCP endpoint. Do not expose these ports beyond loopback for this pilot.

## Stop the pilot

Stop and remove the containers and network while retaining disposable local volumes:

```powershell
docker compose --env-file $PilotEnv down
```

This intentionally does not remove volumes. Keep the env file if you plan to restart the same pilot. Remove the file later with `Remove-Item -LiteralPath $PilotEnv` after the stack is stopped; this does not delete the Compose data volumes. Do not use `down --volumes` unless you intentionally decide to erase the pilot database and cache data.
