#Requires -Version 5.1
<#
.SYNOPSIS
  Thin wrapper → deploy/postgresql/db-init.ps1 (GameS first-time DB bootstrap).

.EXAMPLE
  .\scripts\init-postgres.ps1 -WriteEnv
  .\scripts\init-postgres.ps1 -SplitAuth -WriteEnv
#>
param(
    [string]$PgHost = '',
    [int]$Port = 5432,
    [string]$AdminUser = '',
    [string]$AdminPassword = '',
    [string]$AppUser = '',
    [string]$AppPassword = '',
    [string]$Database = '',
    [string]$AuthDatabase = '',
    [switch]$SplitAuth,
    [switch]$WriteEnv,
    [switch]$NoWriteEnv,
    [switch]$Docker,
    [ValidateSet('game', 'auth', 'all')]
    [string]$Scope = 'game'
)

$ErrorActionPreference = 'Stop'
$Root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$DeployPg = Join-Path $Root 'deploy\postgresql'
$DbInit = Join-Path $DeployPg 'db-init.ps1'
$ConfigIni = Join-Path $DeployPg 'config.ini'

if (-not (Test-Path -LiteralPath $DbInit)) {
    throw "missing $DbInit"
}

if ($Docker) {
    Write-Host "Note: -Docker is not used by deploy/postgresql; use compose defaults (game:game) or run init against host PG."
    Write-Host "Forwarding to deploy/postgresql/db-init.ps1 without Docker."
}

# Optional one-shot overrides via env for db-init / config consumers
if ($AdminPassword) { $env:PGPASSWORD = $AdminPassword }
if ($AdminUser) {
    # db-init reads superuser from config.ini; allow env hint for advanced use
    $env:GAMES_PG_ADMIN_USER = $AdminUser
}

$args = @('-ConfigIni', $ConfigIni, '-Scope', $Scope)
if ($SplitAuth) { $args += '-SplitAuth' }
if ($WriteEnv) { $args += '-WriteEnv' }
if ($NoWriteEnv) { $args += '-NoWriteEnv' }
if ($AppPassword) { $args += @('-AppPassword', $AppPassword) }

# If caller passed app/db overrides, patch a temp note — prefer editing config.ini.
if ($AppUser -or $Database -or $AuthDatabase -or $PgHost) {
    Write-Host "Tip: set Application/PostgreSQL fields in deploy/postgresql/config.ini for persistent overrides."
}

& $DbInit @args
