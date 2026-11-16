#Requires -Version 5.1
<#
.SYNOPSIS
  GameS first-time PostgreSQL bootstrap: create app role + database(s).

.EXAMPLE
  .\db-init.ps1
  .\db-init.ps1 -Scope all -SplitAuth -WriteEnv
#>
param(
    [ValidateSet('game', 'auth', 'all')]
    [string]$Scope = '',
    [switch]$SplitAuth,
    [switch]$WriteEnv,
    [switch]$NoWriteEnv,
    [string]$ConfigIni = '',
    [string]$AppPassword = ''
)

$ErrorActionPreference = 'Stop'
try {
    [Console]::OutputEncoding = [System.Text.UTF8Encoding]::new()
} catch { }

$ScriptDir = $PSScriptRoot
if ([string]::IsNullOrWhiteSpace($ConfigIni)) {
    $ConfigIni = Join-Path $ScriptDir 'config.ini'
}

$cfg = & (Join-Path $ScriptDir 'read-postgres-config.ps1') -ConfigIni $ConfigIni
if (-not $cfg) { throw 'failed to load config.ini' }

$RepoRoot = [string]$cfg.REPO_ROOT
$Psql = [string]$cfg.PSQL_EXE
$AdminUser = [string]$cfg.POSTGRES_SUPERUSER
$AdminPassword = [string]$cfg.POSTGRES_SUPERUSER_PASSWORD
if (-not $AdminPassword -and $env:PGPASSWORD) {
    $AdminPassword = [string]$env:PGPASSWORD
}
if ($env:GAMES_PG_ADMIN_USER) {
    $AdminUser = [string]$env:GAMES_PG_ADMIN_USER
}
$AppUser = [string]$cfg.POSTGRES_USER
$Database = [string]$cfg.POSTGRES_DB
$AuthDatabase = [string]$cfg.POSTGRES_DB_AUTH
$HostName = [string]$cfg.POSTGRES_HOST
$Port = [string]$cfg.PG_PORT
$DefaultScope = [string]$cfg.DEFAULT_SCOPE
$WriteEnvCfg = ([string]$cfg.WRITE_ENV).ToLower() -in @('true', '1', 'yes')

if (-not $Scope) { $Scope = if ($DefaultScope) { $DefaultScope } else { 'game' } }
if ($SplitAuth -and [string]::IsNullOrWhiteSpace($AuthDatabase)) {
    $AuthDatabase = 'auth'
}
if ($AppPassword) {
    $cfgPassword = $AppPassword
} else {
    $cfgPassword = [string]$cfg.POSTGRES_PASSWORD
}

function New-AppPassword {
    $alphabet = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789'
    $bytes = New-Object byte[] 24
    [System.Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
    -join ($bytes | ForEach-Object { $alphabet[$_ % $alphabet.Length] })
}

function Escape-SqlLiteral([string]$s) {
    return $s.Replace("'", "''")
}

function Escape-Uri([string]$s) {
    return [uri]::EscapeDataString($s)
}

function Upsert-EnvFile {
    param([string]$Path, [hashtable]$Pairs)
    $dir = Split-Path -Parent $Path
    if (-not (Test-Path $dir)) { New-Item -ItemType Directory -Path $dir | Out-Null }
    $lines = @()
    if (Test-Path $Path) {
        $lines = @(Get-Content -LiteralPath $Path)
    } else {
        $example = Join-Path $dir '.env.example'
        if (Test-Path $example) { $lines = @(Get-Content -LiteralPath $example) }
    }
    $pending = @{}
    foreach ($k in $Pairs.Keys) { $pending[$k] = $true }
    $out = New-Object System.Collections.Generic.List[string]
    foreach ($line in $lines) {
        if ($line -match '^\s*#' -or $line.Trim() -eq '') {
            $out.Add($line)
            continue
        }
        $idx = $line.IndexOf('=')
        if ($idx -lt 1) {
            $out.Add($line)
            continue
        }
        $k = $line.Substring(0, $idx).Trim()
        if ($pending.ContainsKey($k)) {
            $out.Add("$k=$($Pairs[$k])")
            $pending.Remove($k)
        } else {
            $out.Add($line)
        }
    }
    foreach ($k in @($pending.Keys)) {
        $out.Add("$k=$($Pairs[$k])")
    }
    Set-Content -LiteralPath $Path -Value $out -Encoding UTF8
}

if (-not $Psql -or -not (Test-Path -LiteralPath $Psql)) {
    throw "psql not found. Set install_path in config.ini or add psql to PATH."
}

if (-not $cfgPassword) {
    $cfgPassword = New-AppPassword
    Write-Host "Generated Application password (saved to postgres.env)."
}

$doGame = ($Scope -eq 'game' -or $Scope -eq 'all')
$doAuth = ($Scope -eq 'auth' -or $Scope -eq 'all') -and -not [string]::IsNullOrWhiteSpace($AuthDatabase)
if ($Scope -eq 'auth' -and [string]::IsNullOrWhiteSpace($AuthDatabase)) {
    throw "Scope=auth requires Application.db_auth (or -SplitAuth)."
}
if ($Scope -eq 'all' -and [string]::IsNullOrWhiteSpace($AuthDatabase)) {
    # all without db_auth: just ensure game
    $doAuth = $false
}

$roleSql = Escape-SqlLiteral $AppUser
$passSql = Escape-SqlLiteral $cfgPassword

$sql = @"
DO `$`$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '$roleSql') THEN
    CREATE ROLE $AppUser LOGIN PASSWORD '$passSql';
  ELSE
    ALTER ROLE $AppUser WITH LOGIN PASSWORD '$passSql';
  END IF;
END
`$`$;
"@

if ($doGame) {
    $sql += @"

SELECT format('CREATE DATABASE %I OWNER %I', '$Database', '$AppUser')
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = '$Database')\gexec
GRANT ALL PRIVILEGES ON DATABASE $Database TO $AppUser;
ALTER DATABASE $Database OWNER TO $AppUser;
"@
}
if ($doAuth) {
    $sql += @"

SELECT format('CREATE DATABASE %I OWNER %I', '$AuthDatabase', '$AppUser')
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = '$AuthDatabase')\gexec
GRANT ALL PRIVILEGES ON DATABASE $AuthDatabase TO $AppUser;
ALTER DATABASE $AuthDatabase OWNER TO $AppUser;
"@
}

$schemaSql = @"
GRANT ALL ON SCHEMA public TO $AppUser;
ALTER SCHEMA public OWNER TO $AppUser;
"@

$tmpSql = Join-Path $env:TEMP ("games-db-init-{0}.sql" -f [guid]::NewGuid().ToString('N'))
$tmpSchema = Join-Path $env:TEMP ("games-db-schema-{0}.sql" -f [guid]::NewGuid().ToString('N'))
Set-Content -LiteralPath $tmpSql -Value $sql -Encoding UTF8
Set-Content -LiteralPath $tmpSchema -Value $schemaSql -Encoding UTF8

function Invoke-AdminPsql {
    param([string]$Db, [string]$File)
    $args = @('-v', 'ON_ERROR_STOP=1', '-U', $AdminUser, '-d', $Db, '-f', $File)
    if ($AdminPassword) {
        $env:PGPASSWORD = $AdminPassword
        $args = @('-h', $HostName, '-p', $Port) + $args
        Write-Host "Connecting $AdminUser@$HostName`:$Port db=$Db ..."
    } else {
        if ($env:PGPASSWORD) { Remove-Item Env:PGPASSWORD -ErrorAction SilentlyContinue }
        Write-Host "Connecting $AdminUser via local socket (no password) db=$Db ..."
    }
    & $Psql @args
    if ($LASTEXITCODE -ne 0) { throw "psql failed on $Db (exit $LASTEXITCODE)" }
}

try {
    Invoke-AdminPsql -Db 'postgres' -File $tmpSql
    if ($doGame) { Invoke-AdminPsql -Db $Database -File $tmpSchema }
    if ($doAuth) { Invoke-AdminPsql -Db $AuthDatabase -File $tmpSchema }
} finally {
    Remove-Item -LiteralPath $tmpSql, $tmpSchema -Force -ErrorAction SilentlyContinue
}

$authDbName = if ($AuthDatabase) { $AuthDatabase } else { $Database }
$userEnc = Escape-Uri $AppUser
$passEnc = Escape-Uri $cfgPassword
$gameDsn = "postgres://${userEnc}:${passEnc}@${HostName}:${Port}/${Database}?sslmode=disable"
$authDsn = "postgres://${userEnc}:${passEnc}@${HostName}:${Port}/${authDbName}?sslmode=disable"

$outEnv = Join-Path $ScriptDir 'postgres.env'
$localLines = @(
    "# Generated by deploy/postgresql/db-init.ps1 — do not commit"
    "POSTGRES_HOST=$HostName"
    "POSTGRES_HOST_PORT=$Port"
    "POSTGRES_SUPERUSER=$AdminUser"
    "POSTGRES_SUPERUSER_PASSWORD=$AdminPassword"
    "POSTGRES_USER=$AppUser"
    "POSTGRES_PASSWORD=$cfgPassword"
    "POSTGRES_DB=$Database"
    "POSTGRES_DB_AUTH=$AuthDatabase"
    "DATABASE_URL=$gameDsn"
    "AUTH_DATABASE_URL=$authDsn"
)
Set-Content -LiteralPath $outEnv -Value $localLines -Encoding UTF8
Write-Host "Wrote $outEnv"

$shouldWriteEnv = $WriteEnv -or ($WriteEnvCfg -and -not $NoWriteEnv)
if ($shouldWriteEnv) {
    Upsert-EnvFile -Path (Join-Path $RepoRoot 'server\.env') -Pairs @{ DATABASE_URL = $gameDsn }
    Upsert-EnvFile -Path (Join-Path $RepoRoot 'auth-service\.env') -Pairs @{ DATABASE_URL = $authDsn }
    Write-Host "Updated server/.env and auth-service/.env DATABASE_URL"
}

Write-Host ""
Write-Host "Done. Next (from repo root):"
Write-Host "  .\scripts\dev.ps1 seed-dev"
Write-Host "  .\scripts\dev.ps1 run-auth"
Write-Host "DSN (game): $gameDsn"
if ($AuthDatabase) {
    Write-Host "DSN (auth): $authDsn"
}
