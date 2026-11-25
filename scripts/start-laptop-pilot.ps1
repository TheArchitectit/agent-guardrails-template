[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

$Revision = '2d71e714ae06a6a7b639cf16bd388b6736143104'
$PreviousRevision = 'ba9ff630b24821507d6c93c6c6117ac7be4e9b1e'
$ArchiveSha256 = 'cd7756d0e6baa84af622faa20a1f20cc0d180e8b1eb4df97a0dc1b75accc879f'
$PilotHome = Join-Path $env:LOCALAPPDATA 'GuardrailsLaptopPilot'
$SourcePath = Join-Path $PilotHome ("s-{0}" -f $Revision)
$SourceMarker = Join-Path $SourcePath '.pilot-source-revision'
$PilotEnv = Join-Path $PilotHome ("pilot-{0}.env" -f $Revision)
$ProjectName = 'guardrails-laptop-pilot'
$ComposePath = Join-Path $SourcePath 'docker-compose.yml'
$ContextArgs = @('--context', 'desktop-linux')

function Stop-Pilot([string]$Message) {
    throw $Message
}

function Assert-NoAmbientOverrides {
    $names = @(
        'DOCKER_HOST', 'DOCKER_CONTEXT', 'DOCKER_TLS_VERIFY', 'DOCKER_CERT_PATH',
        'COMPOSE_FILE', 'COMPOSE_PROJECT_NAME',
        'COMPOSE_PROFILES', 'COMPOSE_ENV_FILES', 'COMPOSE_PATH_SEPARATOR',
        'DB_USER', 'DB_NAME', 'DB_PASSWORD', 'REDIS_PASSWORD', 'MCP_API_KEY',
        'IDE_API_KEY', 'JWT_SECRET', 'DEPLOYMENT_PROFILE',
        'DEPLOYMENT_BIND_ATTESTATION', 'MCP_PORT', 'WEB_PORT', 'VERSION',
        'BUILD_TIME', 'GIT_COMMIT', 'WEB_ENABLED', 'PRODUCTION_MODE', 'LOG_LEVEL',
        'REQUEST_TIMEOUT', 'SHUTDOWN_TIMEOUT', 'DB_SSLMODE', 'JWT_ISSUER',
        'JWT_EXPIRY', 'RATE_LIMIT_MCP', 'RATE_LIMIT_IDE', 'RATE_LIMIT_SESSION',
        'RATE_LIMIT_WINDOW', 'CACHE_TTL_RULES', 'CACHE_TTL_DOCS', 'CACHE_TTL_SEARCH',
        'ENABLE_VALIDATION', 'ENABLE_METRICS', 'ENABLE_AUDIT_LOGGING', 'ENABLE_CACHE'
    )
    foreach ($name in $names) {
        if (-not [string]::IsNullOrEmpty([Environment]::GetEnvironmentVariable($name))) {
            Stop-Pilot ("Clear the ambient {0} variable before running the local pilot." -f $name)
        }
    }
}

function New-PilotSecret {
    $bytes = New-Object byte[] 32
    $rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
    try { $rng.GetBytes($bytes) } finally { $rng.Dispose() }
    'Aa0' + [Convert]::ToBase64String($bytes).TrimEnd('=').Replace('+', '-').Replace('/', '_')
}

function Read-PilotEnvironment([string]$Path, [string]$ExpectedRevision) {
    $values = @{}
    foreach ($line in [System.IO.File]::ReadAllLines($Path)) {
        if ([string]::IsNullOrWhiteSpace($line)) { continue }
        $separator = $line.IndexOf('=')
        if ($separator -lt 1) { Stop-Pilot 'The existing pilot env file is malformed and was left unchanged.' }
        $key = $line.Substring(0, $separator)
        if ($values.ContainsKey($key)) { Stop-Pilot 'The existing pilot env file has duplicate keys and was left unchanged.' }
        $values[$key] = $line.Substring($separator + 1)
    }
    $expectedKeys = @('DB_USER', 'DB_NAME', 'DB_PASSWORD', 'REDIS_PASSWORD', 'MCP_API_KEY', 'IDE_API_KEY', 'JWT_SECRET', 'DEPLOYMENT_PROFILE', 'DEPLOYMENT_BIND_ATTESTATION', 'GIT_COMMIT')
    if (@($values.Keys | Where-Object { $_ -cnotin $expectedKeys }).Count -ne 0 -or @($expectedKeys | Where-Object { -not $values.ContainsKey($_) -or [string]::IsNullOrWhiteSpace($values[$_]) }).Count -ne 0) {
        Stop-Pilot 'The existing pilot env file is incomplete or has unexpected keys. It was left unchanged.'
    }
    if ($values.DB_USER -ne 'guardrail' -or $values.DB_NAME -ne 'guardrails' -or $values.GIT_COMMIT -ne $ExpectedRevision -or
        $values.DEPLOYMENT_PROFILE -ne 'local' -or $values.DEPLOYMENT_BIND_ATTESTATION -ne 'isolated-local-bridge:host-loopback-publish') {
        Stop-Pilot 'The existing pilot env file does not match its pinned local pilot revision. It was left unchanged.'
    }
    foreach ($key in @('DB_PASSWORD', 'REDIS_PASSWORD', 'MCP_API_KEY', 'IDE_API_KEY', 'JWT_SECRET')) {
        if ($values[$key] -notmatch '^Aa0[A-Za-z0-9_-]{43}$') { Stop-Pilot 'The existing pilot env file does not contain valid generated synthetic credentials. It was left unchanged.' }
    }
    return $values
}

function Ensure-PilotEnvironment([string[]]$KnownPilotVolumes) {
    if (Test-Path -LiteralPath $PilotEnv) {
        [void](Read-PilotEnvironment $PilotEnv $Revision)
        return
    }

    $priorValues = $null
    $priorEnv = Join-Path $PilotHome ("pilot-{0}.env" -f $PreviousRevision)
    if ($PreviousRevision -ne $Revision -and (Test-Path -LiteralPath $priorEnv)) {
        $priorValues = Read-PilotEnvironment $priorEnv $PreviousRevision
    } elseif (@($KnownPilotVolumes).Count -gt 0) {
        Stop-Pilot 'Existing pilot data volumes were found, but no validated matching credentials are available. The volumes were left unchanged.'
    }

    if ($null -ne $priorValues) {
        # Preserve database and service credentials across a source repin. The
        # prior file remains untouched; the new revision gets a create-new copy.
        $lines = @(
            ("DB_USER={0}" -f $priorValues.DB_USER)
            ("DB_NAME={0}" -f $priorValues.DB_NAME)
            ("DB_PASSWORD={0}" -f $priorValues.DB_PASSWORD)
            ("REDIS_PASSWORD={0}" -f $priorValues.REDIS_PASSWORD)
            ("MCP_API_KEY={0}" -f $priorValues.MCP_API_KEY)
            ("IDE_API_KEY={0}" -f $priorValues.IDE_API_KEY)
            ("JWT_SECRET={0}" -f $priorValues.JWT_SECRET)
            ("DEPLOYMENT_PROFILE={0}" -f $priorValues.DEPLOYMENT_PROFILE)
            ("DEPLOYMENT_BIND_ATTESTATION={0}" -f $priorValues.DEPLOYMENT_BIND_ATTESTATION)
            ("GIT_COMMIT={0}" -f $Revision)
        )
    } else {
        $lines = @(
            'DB_USER=guardrail'
            'DB_NAME=guardrails'
            ("DB_PASSWORD={0}" -f (New-PilotSecret))
            ("REDIS_PASSWORD={0}" -f (New-PilotSecret))
            ("MCP_API_KEY={0}" -f (New-PilotSecret))
            ("IDE_API_KEY={0}" -f (New-PilotSecret))
            ("JWT_SECRET={0}" -f (New-PilotSecret))
            'DEPLOYMENT_PROFILE=local'
            'DEPLOYMENT_BIND_ATTESTATION=isolated-local-bridge:host-loopback-publish'
            ("GIT_COMMIT={0}" -f $Revision)
        )
    }
    $stream = [System.IO.File]::Open($PilotEnv, [System.IO.FileMode]::CreateNew, [System.IO.FileAccess]::Write, [System.IO.FileShare]::None)
    $writer = [System.IO.StreamWriter]::new($stream, [System.Text.UTF8Encoding]::new($false))
    try { foreach ($line in $lines) { $writer.WriteLine($line) } } finally { $writer.Dispose() }
}

function Ensure-PinnedSource {
    if (Test-Path -LiteralPath $SourcePath) {
        if (-not (Test-Path -LiteralPath $SourceMarker) -or ([System.IO.File]::ReadAllText($SourceMarker).Trim() -ne $Revision)) {
            Stop-Pilot 'The existing versioned pilot source is unrecognized and was left unchanged.'
        }
        if (-not (Test-Path -LiteralPath $ComposePath)) { Stop-Pilot 'The existing pinned pilot source is incomplete.' }
        return $SourcePath
    }

    $tempRoot = Join-Path $PilotHome ("z{0}" -f [guid]::NewGuid().ToString('N').Substring(0, 6))
    $zipPath = Join-Path $tempRoot 'source.zip'
    $projectPath = Join-Path $tempRoot 'project'
    New-Item -ItemType Directory -Path $tempRoot | Out-Null
    try {
        $archiveUrl = "https://codeload.github.com/TheArchitectit/agent-guardrails-template/zip/{0}" -f $Revision
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        Invoke-WebRequest -UseBasicParsing -Uri $archiveUrl -OutFile $zipPath
        $archiveHash = (Get-FileHash -LiteralPath $zipPath -Algorithm SHA256).Hash.ToLowerInvariant()
        if ($archiveHash -ne $ArchiveSha256) { Stop-Pilot 'The downloaded pinned source archive failed its SHA-256 check.' }
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        $archive = [System.IO.Compression.ZipFile]::OpenRead($zipPath)
        try {
            $prefix = "agent-guardrails-template-{0}/" -f $Revision
            $projectPrefix = $prefix + 'mcp-server/'
            $ComposeArchivePath = $prefix + 'docker-compose.yml'
            $foundCompose = $false
            $destinationRoot = [System.IO.Path]::GetFullPath($projectPath) + [System.IO.Path]::DirectorySeparatorChar
            foreach ($entry in $archive.Entries) {
                if ($entry.FullName -eq $ComposeArchivePath) {
                    $relative = 'docker-compose.yml'
                    $foundCompose = $true
                } elseif ($entry.FullName.StartsWith($projectPrefix, [StringComparison]::Ordinal)) {
                    $relative = $entry.FullName.Substring($prefix.Length).Replace('/', [System.IO.Path]::DirectorySeparatorChar)
                } else { continue }
                if ([string]::IsNullOrEmpty($relative) -or $relative.Contains('..')) { Stop-Pilot 'The pinned archive contained an unsafe project path.' }
                $destination = [System.IO.Path]::GetFullPath((Join-Path $projectPath $relative))
                if (-not $destination.StartsWith($destinationRoot, [StringComparison]::OrdinalIgnoreCase)) { Stop-Pilot 'The pinned archive path escaped the pilot staging directory.' }
                $parent = [System.IO.Path]::GetDirectoryName($destination)
                New-Item -ItemType Directory -Force -Path $parent | Out-Null
                if ($entry.FullName.EndsWith('/')) { continue }
                $inputStream = $entry.Open()
                $outputStream = [System.IO.File]::Create($destination)
                try { $inputStream.CopyTo($outputStream) } finally { $outputStream.Dispose(); $inputStream.Dispose() }
            }
            if (-not $foundCompose -or -not (Test-Path -LiteralPath (Join-Path $projectPath 'mcp-server\deploy\Dockerfile'))) { Stop-Pilot 'The pinned source archive did not contain the expected Compose project.' }
        } finally { $archive.Dispose() }
        Set-Content -LiteralPath (Join-Path $projectPath '.pilot-source-revision') -Value $Revision -NoNewline -Encoding ASCII
        if (Test-Path -LiteralPath $SourcePath) { Stop-Pilot 'The versioned source appeared during installation; the transient download was discarded.' }
        Move-Item -LiteralPath $projectPath -Destination $SourcePath
    } finally {
        if (Test-Path -LiteralPath $tempRoot) { Remove-Item -LiteralPath $tempRoot -Recurse -Force }
    }
    if (-not (Test-Path -LiteralPath $ComposePath)) { Stop-Pilot 'Pinned source installation did not complete.' }
    return $SourcePath
}

function Ensure-LocalDocker {
    if ($env:OS -ne 'Windows_NT' -or [string]::IsNullOrWhiteSpace($env:LOCALAPPDATA)) { Stop-Pilot 'This launcher requires Windows and a user LOCALAPPDATA directory.' }
    if ($null -eq (Get-Command docker -ErrorAction SilentlyContinue)) { Stop-Pilot 'Install Docker Desktop and ensure its Docker CLI is on PATH.' }
    $composeVersion = & docker compose version --short 2>$null
    if ($LASTEXITCODE -ne 0 -or $composeVersion -notmatch '^v?([0-9]+)\.([0-9]+)') { Stop-Pilot 'Docker Compose v2 is required for this pilot.' }
    $versionParts = [regex]::Match($composeVersion, '^v?([0-9]+)\.([0-9]+)')
    if ([int]$versionParts.Groups[1].Value -lt 2 -or ([int]$versionParts.Groups[1].Value -eq 2 -and [int]$versionParts.Groups[2].Value -lt 17)) { Stop-Pilot 'Docker Compose 2.17 or newer is required for health-aware startup.' }
    $contextHost = & docker @ContextArgs context inspect desktop-linux --format '{{.Endpoints.docker.Host}}' 2>$null
    if ($LASTEXITCODE -eq 0 -and $contextHost -notmatch '^npipe:/{2,4}\./pipe/dockerDesktopLinuxEngine$') {
        Stop-Pilot 'Docker context desktop-linux points somewhere other than Docker Desktop local Linux. No daemon was contacted.'
    }
    $info = $null
    if ($LASTEXITCODE -eq 0) { $info = & docker @ContextArgs info --format '{{.OSType}}' 2>$null }
    if ($LASTEXITCODE -ne 0 -or $info -ne 'linux') {
        $desktopExe = Join-Path $env:ProgramFiles 'Docker\Docker\Docker Desktop.exe'
        if (-not (Test-Path -LiteralPath $desktopExe)) { Stop-Pilot 'Docker Desktop was not found at its standard Program Files path.' }
        $running = Get-Process -Name 'Docker Desktop' -ErrorAction SilentlyContinue
        if ($null -eq $running) { Start-Process -FilePath $desktopExe | Out-Null }
        $deadline = (Get-Date).AddMinutes(5)
        do {
            Start-Sleep -Seconds 5
            $contextHost = & docker @ContextArgs context inspect desktop-linux --format '{{.Endpoints.docker.Host}}' 2>$null
            if ($LASTEXITCODE -eq 0 -and $contextHost -notmatch '^npipe:/{2,4}\./pipe/dockerDesktopLinuxEngine$') { Stop-Pilot 'Docker Desktop created a desktop-linux context that does not point to its local Linux named pipe.' }
            if ($LASTEXITCODE -ne 0) { continue }
            $info = & docker @ContextArgs info --format '{{.OSType}}' 2>$null
            if ($LASTEXITCODE -eq 0 -and $info -eq 'linux') { return }
        } while ((Get-Date) -lt $deadline)
        Stop-Pilot 'Docker Desktop did not provide a ready local Linux engine within five minutes.'
    }
}

function Invoke-DockerCompose([string[]]$Arguments) {
    & docker @ContextArgs compose --project-name $ProjectName --env-file $PilotEnv -f $ComposePath @Arguments
    if ($LASTEXITCODE -ne 0) { Stop-Pilot 'Docker Compose reported a failure. Review its output above; generated credentials were not printed.' }
}

function Assert-NoConflictingContainers {
    foreach ($name in @('guardrail-postgres', 'guardrail-redis', 'guardrail-mcp-server')) {
        $labelsJson = & docker @ContextArgs inspect --format '{{json .Config.Labels}}' $name 2>$null
        if ($LASTEXITCODE -eq 0) {
            $labels = $labelsJson | ConvertFrom-Json
            if ($null -eq $labels -or $labels.'com.docker.compose.project' -ne $ProjectName) {
                Stop-Pilot ("A container named {0} is not owned by this pilot; stop it deliberately before running this pilot." -f $name)
            }
        }
    }
}

function Get-PilotVolumes {
    $volumes = & docker @ContextArgs volume ls --filter ("label=com.docker.compose.project={0}" -f $ProjectName) --format '{{.Name}}' 2>$null
    if ($LASTEXITCODE -ne 0) { Stop-Pilot 'Could not verify existing pilot volume ownership.' }
    $allNames = & docker @ContextArgs volume ls --format '{{.Name}}' 2>$null
    if ($LASTEXITCODE -ne 0) { Stop-Pilot 'Could not verify expected pilot volume names.' }
    $expectedNames = @(("{0}_pg_data" -f $ProjectName), ("{0}_redis_data" -f $ProjectName))
    $volumes += @($allNames | Where-Object { $_ -cin $expectedNames })
    return @($volumes | Where-Object { -not [string]::IsNullOrWhiteSpace($_) } | Sort-Object -Unique)
}

try {
    if ($env:OS -ne 'Windows_NT' -or [string]::IsNullOrWhiteSpace($env:LOCALAPPDATA)) { Stop-Pilot 'This launcher requires Windows and a user LOCALAPPDATA directory.' }
    Assert-NoAmbientOverrides
    New-Item -ItemType Directory -Force -Path $PilotHome | Out-Null
    $SourcePath = Ensure-PinnedSource
    $ComposePath = Join-Path $SourcePath 'docker-compose.yml'
    Ensure-LocalDocker
    Assert-NoConflictingContainers
    $knownPilotVolumes = Get-PilotVolumes
    Ensure-PilotEnvironment $knownPilotVolumes

    Push-Location $SourcePath
    try {
        # Quiet config validation and the JSON policy check keep interpolated secrets internal.
        Invoke-DockerCompose @('config', '--quiet')
        $configText = & docker @ContextArgs compose --project-name $ProjectName --env-file $PilotEnv -f $ComposePath config --format json
        if ($LASTEXITCODE -ne 0) { Stop-Pilot 'Compose could not render the pilot configuration.' }
        $config = ($configText -join "`n") | ConvertFrom-Json
        $service = $config.services.'mcp-server'
        if ($service.environment.DEPLOYMENT_PROFILE -ne 'local' -or $service.environment.DEPLOYMENT_BIND_ATTESTATION -ne 'isolated-local-bridge:host-loopback-publish') { Stop-Pilot 'Rendered Compose profile or bind attestation did not match the local pilot.' }
        if ($service.build.args.GIT_COMMIT -ne $Revision) { Stop-Pilot 'Rendered image build did not pin the expected source revision.' }
        $portSpecs = @($service.ports)
        if ($portSpecs.Count -ne 2 -or @($portSpecs | Where-Object { $_.host_ip -ne '127.0.0.1' -or $_.target -notin @(8080, 8081) -or $_.published -notin @('8080', '8081') }).Count -ne 0) { Stop-Pilot 'Rendered Compose ports did not match the expected loopback-only pilot ports.' }
        if ($service.environment.DB_PASSWORD -ne $config.services.postgres.environment.POSTGRES_PASSWORD -or $service.environment.REDIS_PASSWORD -ne $config.services.redis.environment.REDIS_PASSWORD) { Stop-Pilot 'Rendered service credential sources did not match the synthetic pilot environment.' }
        $migration = $config.services.migrate
        if ($null -eq $migration -or $migration.entrypoint -notcontains '/run-migrations' -or
            $migration.environment.MIGRATION_DATABASE_URL -ne 'postgres://postgres:5432' -or
            $migration.networks.PSObject.Properties.Name -notcontains 'guardrails-net' -or
            $service.networks.PSObject.Properties.Name -notcontains 'guardrails-net' -or
            $migration.depends_on.postgres.condition -ne 'service_healthy' -or
            $service.depends_on.migrate.condition -ne 'service_completed_successfully' -or
            $migration.environment.PGUSER -ne $config.services.postgres.environment.POSTGRES_USER -or
            $migration.environment.PGPASSWORD -ne $config.services.postgres.environment.POSTGRES_PASSWORD -or
            $migration.environment.PGDATABASE -ne $config.services.postgres.environment.POSTGRES_DB -or
            $migration.environment.PGSSLMODE -ne $service.environment.DB_SSLMODE -or
            $migration.image -ne $service.image) {
            Stop-Pilot 'Rendered Compose migration ordering, credential source, or shared network did not match the pilot contract.'
        }
        foreach ($dependency in @('postgres', 'redis', 'mcp-server')) {
            if ($config.services.$dependency.networks.PSObject.Properties.Name -notcontains 'guardrails-net') { Stop-Pilot 'All pilot services must share the isolated guardrails bridge.' }
        }

        Invoke-DockerCompose @('up', '--wait', '--wait-timeout', '300', '-d', '--build')
        $mcpBind = & docker @ContextArgs compose --project-name $ProjectName --env-file $PilotEnv -f $ComposePath port mcp-server 8080
        $webBind = & docker @ContextArgs compose --project-name $ProjectName --env-file $PilotEnv -f $ComposePath port mcp-server 8081
        if ($LASTEXITCODE -ne 0 -or $mcpBind.Trim() -ne '127.0.0.1:8080' -or $webBind.Trim() -ne '127.0.0.1:8081') { Stop-Pilot 'Active port publications are not the expected IPv4 loopback bindings.' }
        foreach ($port in @(8080, 8081)) {
            $probe = [System.Net.Sockets.TcpClient]::new()
            try { $probe.Connect('127.0.0.1', $port) } catch { Stop-Pilot ("Loopback port {0} did not accept connections." -f $port) } finally { $probe.Dispose() }
        }
        $response = Invoke-WebRequest -UseBasicParsing -Uri 'http://127.0.0.1:8081/health/ready' -TimeoutSec 15
        if ($response.StatusCode -ne 200) { Stop-Pilot 'Application readiness endpoint did not return HTTP 200.' }
        $status = & docker @ContextArgs compose --project-name $ProjectName --env-file $PilotEnv -f $ComposePath ps --format '{{.Service}} {{.Status}}'
        if ($LASTEXITCODE -ne 0) { Stop-Pilot 'Could not read pilot service status.' }
        Write-Output ("Pilot source revision: {0}" -f $Revision)
        Write-Output 'Pilot status:'
        $status | ForEach-Object { Write-Output $_ }
        Write-Output 'Ready endpoints: http://127.0.0.1:8081 and http://127.0.0.1:8080/mcp'
        Write-Output 'Restart with this launcher. Stop with: docker --context desktop-linux compose --project-name guardrails-laptop-pilot --env-file <pilot-env> -f <pinned-project> down'
        Write-Output 'The pilot env and data volumes are retained for restart.'
    } finally {
        Pop-Location
    }
} catch {
    Write-Error $_.Exception.Message
    exit 1
}
