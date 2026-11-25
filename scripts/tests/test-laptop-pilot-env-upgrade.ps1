$ErrorActionPreference = 'Stop'

function Assert-True([bool]$Condition, [string]$Message) {
    if (-not $Condition) { throw $Message }
}

function Assert-Throws([scriptblock]$Action, [string]$Message) {
    $threw = $false
    try { & $Action } catch { $threw = $true }
    Assert-True $threw $Message
}

function Stop-Pilot([string]$Message) { throw $Message }

$launcherPath = Join-Path $PSScriptRoot '..\start-laptop-pilot.ps1'
$tokens = $null
$parseErrors = $null
$launcherAst = [System.Management.Automation.Language.Parser]::ParseFile(
    (Resolve-Path $launcherPath), [ref]$tokens, [ref]$parseErrors
)
Assert-True ($parseErrors.Count -eq 0) 'Launcher source did not parse.'
$functionNames = @('New-PilotSecret', 'Read-PilotEnvironment', 'Ensure-PilotEnvironment', 'Get-PilotVolumes')
$definitions = $launcherAst.FindAll({
    param($node)
    $node -is [System.Management.Automation.Language.FunctionDefinitionAst] -and $node.Name -in $functionNames
}, $true)
Assert-True ($definitions.Count -eq $functionNames.Count) 'Expected launcher functions were not found.'
foreach ($definition in $definitions) { Invoke-Expression $definition.Extent.Text }

function docker {
    $global:LASTEXITCODE = 0
    if ($args -contains '--filter') { return @() }
    return 'guardrails-laptop-pilot_pg_data'
}

$testRoot = Join-Path ([System.IO.Path]::GetTempPath()) ('guardrails-pilot-env-test-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $testRoot | Out-Null
try {
    $Revision = 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb'
    $PreviousRevision = 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'
    $PilotHome = $testRoot
    $ProjectName = 'guardrails-laptop-pilot'
    $ContextArgs = @('--context', 'desktop-linux')
    $PilotEnv = Join-Path $PilotHome ("pilot-{0}.env" -f $Revision)
    $priorEnv = Join-Path $PilotHome ("pilot-{0}.env" -f $PreviousRevision)
    $testSecret = 'Aa0' + ('A' * 43)
    $priorLines = @(
        'DB_USER=guardrail'
        'DB_NAME=guardrails'
        ("DB_PASSWORD={0}" -f $testSecret)
        ("REDIS_PASSWORD={0}" -f $testSecret)
        ("MCP_API_KEY={0}" -f $testSecret)
        ("IDE_API_KEY={0}" -f $testSecret)
        ("JWT_SECRET={0}" -f $testSecret)
        'DEPLOYMENT_PROFILE=local'
        'DEPLOYMENT_BIND_ATTESTATION=isolated-local-bridge:host-loopback-publish'
        ("GIT_COMMIT={0}" -f $PreviousRevision)
    )
    [System.IO.File]::WriteAllLines($priorEnv, $priorLines, [System.Text.UTF8Encoding]::new($false))
    $priorHash = (Get-FileHash -LiteralPath $priorEnv -Algorithm SHA256).Hash

    Ensure-PilotEnvironment @('guardrails-laptop-pilot_pg_data')
    $newValues = Read-PilotEnvironment $PilotEnv $Revision
    $oldValues = Read-PilotEnvironment $priorEnv $PreviousRevision
    Assert-True ($newValues.DB_PASSWORD -eq $oldValues.DB_PASSWORD -and
        $newValues.REDIS_PASSWORD -eq $oldValues.REDIS_PASSWORD -and
        $newValues.MCP_API_KEY -eq $oldValues.MCP_API_KEY -and
        $newValues.IDE_API_KEY -eq $oldValues.IDE_API_KEY -and
        $newValues.JWT_SECRET -eq $oldValues.JWT_SECRET) 'Upgrade did not preserve all generated credentials.'
    Assert-True ((Get-FileHash -LiteralPath $priorEnv -Algorithm SHA256).Hash -eq $priorHash) 'Prior env file was changed.'
    $newHash = (Get-FileHash -LiteralPath $PilotEnv -Algorithm SHA256).Hash
    Ensure-PilotEnvironment @('guardrails-laptop-pilot_pg_data')
    Assert-True ((Get-FileHash -LiteralPath $PilotEnv -Algorithm SHA256).Hash -eq $newHash) 'Rerun changed the current env file.'

    $blockedHome = Join-Path $testRoot 'blocked'
    New-Item -ItemType Directory -Path $blockedHome | Out-Null
    $PilotHome = $blockedHome
    $PilotEnv = Join-Path $PilotHome ("pilot-{0}.env" -f $Revision)
    Assert-Throws { Ensure-PilotEnvironment @('guardrails-laptop-pilot_pg_data') } 'Missing credentials with existing volumes must stop.'
    Assert-True (-not (Test-Path -LiteralPath $PilotEnv)) 'New credentials were generated for existing volumes.'

    $badHome = Join-Path $testRoot 'malformed'
    New-Item -ItemType Directory -Path $badHome | Out-Null
    $PilotHome = $badHome
    $PilotEnv = Join-Path $PilotHome ("pilot-{0}.env" -f $Revision)
    $badPrior = Join-Path $PilotHome ("pilot-{0}.env" -f $PreviousRevision)
    [System.IO.File]::WriteAllText($badPrior, "DB_PASSWORD=invalid`n", [System.Text.UTF8Encoding]::new($false))
    $badHash = (Get-FileHash -LiteralPath $badPrior -Algorithm SHA256).Hash
    Assert-Throws { Ensure-PilotEnvironment @('guardrails-laptop-pilot_pg_data') } 'Malformed prior credentials must stop.'
    Assert-True ((Get-FileHash -LiteralPath $badPrior -Algorithm SHA256).Hash -eq $badHash) 'Malformed prior env file was changed.'
    Assert-True (-not (Test-Path -LiteralPath $PilotEnv)) 'Malformed prior credentials created a new env file.'

    $caseHome = Join-Path $testRoot 'case-sensitive'
    New-Item -ItemType Directory -Path $caseHome | Out-Null
    $PilotHome = $caseHome
    $PilotEnv = Join-Path $PilotHome ("pilot-{0}.env" -f $Revision)
    $casePrior = Join-Path $PilotHome ("pilot-{0}.env" -f $PreviousRevision)
    $caseLines = @($priorLines | ForEach-Object {
        if ($_.StartsWith('DB_PASSWORD=', [System.StringComparison]::Ordinal)) {
            'db_password=' + $_.Substring($_.IndexOf('=') + 1)
        } else { $_ }
    })
    [System.IO.File]::WriteAllLines($casePrior, $caseLines, [System.Text.UTF8Encoding]::new($false))
    Assert-Throws { [void](Read-PilotEnvironment $casePrior $PreviousRevision) } 'Differently cased env keys must be rejected.'
    Assert-True (-not (Test-Path -LiteralPath $PilotEnv)) 'Differently cased prior credentials created a new env file.'

    $detectedVolumes = Get-PilotVolumes
    Assert-True ($detectedVolumes -contains 'guardrails-laptop-pilot_pg_data') 'Exact unmanaged project volume name was not detected.'
    Assert-Throws { Ensure-PilotEnvironment $detectedVolumes } 'Unmanaged project volume without valid credentials must stop.'
    Assert-True (-not (Test-Path -LiteralPath $PilotEnv)) 'New credentials were generated for an unmanaged project volume.'

    'Launcher env upgrade and fail-closed mock checks passed.'
} finally {
    $resolvedTemp = [System.IO.Path]::GetFullPath([System.IO.Path]::GetTempPath())
    $resolvedTest = [System.IO.Path]::GetFullPath($testRoot)
    $testPathPrefix = Join-Path $resolvedTemp 'guardrails-pilot-env-test-'
    if (-not $resolvedTest.StartsWith($testPathPrefix, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw 'Refusing to remove test data outside the temporary directory.'
    }
    Remove-Item -LiteralPath $resolvedTest -Recurse -Force
}
