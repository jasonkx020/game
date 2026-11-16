param(
    [string]$ConfigIni = "",
    [switch]$EmitBat,
    [switch]$PrintSummary,
    [string]$WriteEnvFile,
    [string]$WriteBatFile
)

function Safe-JoinPath {
    param([string]$Parent, [string]$Child)
    if ([string]::IsNullOrWhiteSpace($Parent)) { return $Child }
    if ([string]::IsNullOrWhiteSpace($Child)) { return $Parent }
    return [System.IO.Path]::Combine($Parent.TrimEnd('\', '/'), $Child.TrimStart('\', '/'))
}

if ([string]::IsNullOrWhiteSpace($ConfigIni)) {
    $ConfigIni = Safe-JoinPath $PSScriptRoot "config.ini"
}

function Read-IniValue {
    param([string]$Section, [string]$Key, [string]$Default)
    if (-not (Test-Path -LiteralPath $ConfigIni)) { return $Default }
    $current = ""
    foreach ($line in Get-Content -LiteralPath $ConfigIni -ErrorAction SilentlyContinue) {
        $trim = $line.Trim()
        if ($trim -match '^\s*;') { continue }
        if ($trim -match '^\[(.+)\]$') { $current = $matches[1]; continue }
        if ($current -ne $Section) { continue }
        if ($trim -match "^$([regex]::Escape($Key))\s*=\s*(.*)$") { return $matches[1].Trim() }
    }
    return $Default
}

function Resolve-ConfigPath {
    param([string]$Base, [string]$Path, [string]$Fallback)
    $p = if ([string]::IsNullOrWhiteSpace($Path)) { $Fallback } else { $Path.Trim() }
    if ($p -eq "." -or $p -eq ".\") { return $Base }
    if ($p.StartsWith(".\") -or $p.StartsWith("./")) {
        return (Safe-JoinPath $Base $p.Substring(2))
    }
    if (-not [System.IO.Path]::IsPathRooted($p)) {
        return (Safe-JoinPath $Base $p)
    }
    return $p
}

function Resolve-WindowsServiceName {
    param([string]$Configured, [string]$Version)
    if (-not [string]::IsNullOrWhiteSpace($Configured)) { return $Configured.Trim() }
    $candidates = @(
        "postgresql-x64-$Version",
        "postgresql-$Version",
        "postgresql-x64-$Version-x64"
    )
    foreach ($c in $candidates) {
        $svc = Get-Service -Name $c -ErrorAction SilentlyContinue
        if ($svc) { return $c }
    }
    return "postgresql-x64-$Version"
}

function Resolve-PostgresInstallPath {
    param([string]$Base, [string]$Configured)
    $configured = if ($null -eq $Configured) { "" } else { $Configured.Trim() }
    $resolved = Resolve-ConfigPath $Base $configured $Base
    if ($configured -ne "." -and $configured -ne ".\") {
        return $resolved
    }
    $version = Read-IniValue "PostgreSQL" "version" "16"
    $candidates = @(
        (Safe-JoinPath $env:ProgramFiles "PostgreSQL\$version"),
        (Safe-JoinPath ${env:ProgramFiles(x86)} "PostgreSQL\$version"),
        $Base
    )
    foreach ($c in $candidates) {
        if ($c -and (Test-Path -LiteralPath (Safe-JoinPath $c "bin\psql.exe"))) { return $c }
    }
    return $resolved
}

function Escape-Uri([string]$s) {
    return [uri]::EscapeDataString($s)
}

$scriptRoot = if ([string]::IsNullOrWhiteSpace($PSScriptRoot)) {
    Split-Path -Parent $MyInvocation.MyCommand.Path
} else {
    $PSScriptRoot
}

# deploy/postgresql -> repo root
$repoRoot = [System.IO.Path]::GetFullPath((Safe-JoinPath $scriptRoot "..\.."))

$installPath = Resolve-PostgresInstallPath $scriptRoot (Read-IniValue "PostgreSQL" "install_path" ".")
$dataPath = Resolve-ConfigPath $scriptRoot (Read-IniValue "PostgreSQL" "data_path" "data") (Safe-JoinPath $scriptRoot "data")
$logDir = Resolve-ConfigPath $scriptRoot (Read-IniValue "PostgreSQL" "log_dir" "log") (Safe-JoinPath $scriptRoot "log")
$port = Read-IniValue "PostgreSQL" "port" "5432"
$listenBind = Read-IniValue "PostgreSQL" "listen_bind" "127.0.0.1"
$version = Read-IniValue "PostgreSQL" "version" "16"
$runMode = (Read-IniValue "PostgreSQL" "run_mode" "service").ToLower()
$serviceName = Resolve-WindowsServiceName (Read-IniValue "PostgreSQL" "windows_service_name" "") $version
$addFirewall = (Read-IniValue "PostgreSQL" "add_firewall_rules" "false").ToLower()

$superUser = Read-IniValue "Superuser" "user" "postgres"
$superPassword = Read-IniValue "Superuser" "password" ""

$appUser = Read-IniValue "Application" "user" "game"
$appPassword = Read-IniValue "Application" "password" ""
$database = Read-IniValue "Application" "db" "game"
# Legacy aliases
if ([string]::IsNullOrWhiteSpace($database)) {
    $database = Read-IniValue "Application" "db_edge" "game"
}
$authDatabase = Read-IniValue "Application" "db_auth" ""
$defaultScope = Read-IniValue "Init" "default_scope" "game"
$writeEnv = (Read-IniValue "Init" "write_env" "true").ToLower()

function Get-PgExe([string]$Name) {
    $candidates = @(
        (Safe-JoinPath $installPath "bin\$Name.exe"),
        (Safe-JoinPath $installPath "$Name.exe"),
        (Safe-JoinPath $scriptRoot "bin\$Name.exe"),
        (Safe-JoinPath $scriptRoot "$Name.exe")
    )
    foreach ($c in $candidates) {
        if ($c -and (Test-Path -LiteralPath $c)) { return $c }
    }
    $cmd = Get-Command $Name -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }
    return ""
}

$pgCtlExe = Get-PgExe "pg_ctl"
$psqlExe = Get-PgExe "psql"
$pgIsReadyExe = Get-PgExe "pg_isready"
$initdbExe = Get-PgExe "initdb"

$hostName = if ($listenBind -eq "0.0.0.0") { "127.0.0.1" } else { $listenBind }
$authDbName = if ([string]::IsNullOrWhiteSpace($authDatabase)) { $database } else { $authDatabase }
$userEnc = Escape-Uri $appUser
$passEnc = Escape-Uri $appPassword
$gameDsn = "postgres://${userEnc}:${passEnc}@${hostName}:${port}/${database}?sslmode=disable"
$authDsn = "postgres://${userEnc}:${passEnc}@${hostName}:${port}/${authDbName}?sslmode=disable"
$dbInitScript = Safe-JoinPath $scriptRoot "db-init.ps1"

$config = [ordered]@{
    SCRIPT_DIR                  = $scriptRoot
    REPO_ROOT                   = $repoRoot
    CONFIG_FILE                 = $ConfigIni
    INSTALL_PATH                = $installPath
    DATA_PATH                   = $dataPath
    LOG_DIR                     = $logDir
    PG_PORT                     = $port
    LISTEN_BIND                 = $listenBind
    PG_VERSION                  = $version
    RUN_MODE                    = $runMode
    WINDOWS_SERVICE_NAME        = $serviceName
    ADD_FIREWALL                = $addFirewall
    POSTGRES_SUPERUSER          = $superUser
    POSTGRES_SUPERUSER_PASSWORD = $superPassword
    POSTGRES_HOST               = $hostName
    POSTGRES_HOST_PORT          = $port
    POSTGRES_USER               = $appUser
    POSTGRES_PASSWORD           = $appPassword
    POSTGRES_DB                 = $database
    POSTGRES_DB_AUTH            = $authDatabase
    DEFAULT_SCOPE               = $defaultScope
    WRITE_ENV                   = $writeEnv
    DATABASE_URL                = $gameDsn
    AUTH_DATABASE_URL           = $authDsn
    PG_CTL_EXE                  = $pgCtlExe
    PSQL_EXE                    = $psqlExe
    PG_ISREADY_EXE              = $pgIsReadyExe
    INITDB_EXE                  = $initdbExe
    DB_INIT_SCRIPT              = $dbInitScript
}

if ($WriteEnvFile) {
    $postgresBin = Safe-JoinPath $installPath "bin"
    $lines = @(
        "# Generated by read-postgres-config.ps1 from config.ini",
        "POSTGRES_HOST=$hostName",
        "POSTGRES_HOST_PORT=$port",
        "POSTGRES_SUPERUSER=$superUser",
        "POSTGRES_SUPERUSER_PASSWORD=$superPassword",
        "POSTGRES_USER=$appUser",
        "POSTGRES_PASSWORD=$appPassword",
        "POSTGRES_DB=$database",
        "POSTGRES_DB_AUTH=$authDatabase",
        "DATABASE_URL=$gameDsn",
        "AUTH_DATABASE_URL=$authDsn"
    )
    if (Test-Path -LiteralPath (Safe-JoinPath $postgresBin "psql.exe")) {
        $lines += "POSTGRES_HOME=$installPath"
        $lines += "POSTGRES_BIN=$postgresBin"
    }
    if ($psqlExe) {
        $lines += "POSTGRES_PSQL=$psqlExe"
    }
    $utf8NoBom = New-Object System.Text.UTF8Encoding $false
    [System.IO.File]::WriteAllText($WriteEnvFile, ($lines -join "`r`n") + "`r`n", $utf8NoBom)
    Write-Host "Wrote: $WriteEnvFile"
    return
}

if ($PrintSummary) {
    Write-Host "PostgreSQL config ($ConfigIni)"
    foreach ($kv in $config.GetEnumerator()) {
        if ($kv.Key -match 'PASSWORD|DSN|DATABASE_URL|AUTH_DATABASE_URL') { continue }
        Write-Host ("  {0} = {1}" -f $kv.Key, $kv.Value)
    }
    Write-Host ("  DATABASE_URL = postgres://{0}:****@{1}:{2}/{3}?sslmode=disable" -f $appUser, $hostName, $port, $database)
    return
}

function Get-BatSetLines {
    $lines = New-Object System.Collections.Generic.List[string]
    foreach ($kv in $config.GetEnumerator()) {
        $val = [string]$kv.Value
        if ($null -eq $kv.Value) { $val = "" }
        $val = $val -replace '"', '""'
        $lines.Add(("set `"{0}={1}`"" -f $kv.Key, $kv.Value))
    }
    return $lines
}

if ($WriteBatFile) {
    $utf8NoBom = New-Object System.Text.UTF8Encoding $false
    $batLines = Get-BatSetLines
    [System.IO.File]::WriteAllText($WriteBatFile, ($batLines -join "`r`n") + "`r`n", $utf8NoBom)
    return
}

if ($EmitBat) {
    foreach ($line in Get-BatSetLines) {
        Write-Output $line
    }
    return
}

return [pscustomobject]$config
