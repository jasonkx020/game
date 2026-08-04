param(
    [string]$ConfigIni = (Join-Path $PSScriptRoot "config.ini"),
    [switch]$EmitBat,
    [switch]$PrintSummary,
    [string]$WriteEnvFile
)

function Read-IniValue {
    param([string]$Section, [string]$Key, [string]$Default)
    if (-not (Test-Path $ConfigIni)) { return $Default }
    $current = ""
    foreach ($line in Get-Content $ConfigIni -ErrorAction SilentlyContinue) {
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
        return (Join-Path $Base $p.Substring(2))
    }
    if (-not [System.IO.Path]::IsPathRooted($p)) {
        return (Join-Path $Base $p)
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
    $any = Get-Service -Name "postgresql*" -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($any) { return $any.Name }
    return "postgresql-x64-$Version"
}

function Resolve-PostgresInstallPath {
    param([string]$Base, [string]$Configured)
    $resolved = Resolve-ConfigPath $Base $Configured $Base
    if ($Configured.Trim() -ne "." -and $Configured.Trim() -ne ".\") {
        return $resolved
    }
    $version = Read-IniValue "PostgreSQL" "version" "16"
    $candidates = @(
        (Join-Path ${env:ProgramFiles} "PostgreSQL\$version"),
        (Join-Path ${env:ProgramFiles(x86)} "PostgreSQL\$version"),
        $Base
    )
    foreach ($c in $candidates) {
        if ($c -and (Test-Path (Join-Path $c "bin\psql.exe"))) { return $c }
    }
    return $resolved
}

$scriptRoot = $PSScriptRoot
$repoRoot = (Resolve-Path (Join-Path $scriptRoot "..\..\..")).Path

$installPath = Resolve-PostgresInstallPath $scriptRoot (Read-IniValue "PostgreSQL" "install_path" ".")
$dataPath = Resolve-ConfigPath $scriptRoot (Read-IniValue "PostgreSQL" "data_path" "data") (Join-Path $scriptRoot "data")
$logDir = Resolve-ConfigPath $scriptRoot (Read-IniValue "PostgreSQL" "log_dir" "log") (Join-Path $scriptRoot "log")
$port = Read-IniValue "PostgreSQL" "port" "5432"
$listenBind = Read-IniValue "PostgreSQL" "listen_bind" "127.0.0.1"
$version = Read-IniValue "PostgreSQL" "version" "16"
$runMode = (Read-IniValue "PostgreSQL" "run_mode" "service").ToLower()
$serviceName = Resolve-WindowsServiceName (Read-IniValue "PostgreSQL" "windows_service_name" "") $version
$addFirewall = (Read-IniValue "PostgreSQL" "add_firewall_rules" "true").ToLower()

$superUser = Read-IniValue "Superuser" "user" "postgres"
$superPassword = Read-IniValue "Superuser" "password" ""

$appUser = Read-IniValue "Application" "user" "freetv"
$appPassword = Read-IniValue "Application" "password" "changeme"
$dbEdge = Read-IniValue "Application" "db_edge" "freetv_edge"
$dbCloud = Read-IniValue "Application" "db_cloud" "freetv_cloud"
$defaultScope = Read-IniValue "Init" "default_scope" "tenant"

function Get-PgExe([string]$Name) {
    $candidates = @(
        (Join-Path $installPath "bin\$Name.exe"),
        (Join-Path $installPath "$Name.exe"),
        (Join-Path $scriptRoot "bin\$Name.exe"),
        (Join-Path $scriptRoot "$Name.exe")
    )
    foreach ($c in $candidates) {
        if ($c -and (Test-Path $c)) { return $c }
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
$tenantDsn = "postgres://${appUser}:$appPassword@${hostName}:${port}/${dbEdge}?sslmode=disable"
$platformDsn = "postgres://${appUser}:$appPassword@${hostName}:${port}/${dbCloud}?sslmode=disable"
$dbInitScript = Join-Path $repoRoot "scripts\db-init.ps1"
$dbMigrateScript = Join-Path $repoRoot "scripts\db-migrate.ps1"

$config = [ordered]@{
    SCRIPT_DIR              = $scriptRoot
    REPO_ROOT               = $repoRoot
    CONFIG_FILE             = $ConfigIni
    INSTALL_PATH            = $installPath
    DATA_PATH               = $dataPath
    LOG_DIR                 = $logDir
    PG_PORT                 = $port
    LISTEN_BIND             = $listenBind
    PG_VERSION              = $version
    RUN_MODE                = $runMode
    WINDOWS_SERVICE_NAME    = $serviceName
    ADD_FIREWALL            = $addFirewall
    POSTGRES_SUPERUSER      = $superUser
    POSTGRES_SUPERUSER_PASSWORD = $superPassword
    POSTGRES_HOST           = $hostName
    POSTGRES_HOST_PORT      = $port
    POSTGRES_USER           = $appUser
    POSTGRES_PASSWORD       = $appPassword
    POSTGRES_DB_EDGE        = $dbEdge
    POSTGRES_DB_CLOUD       = $dbCloud
    POSTGRES_DB             = $dbEdge
    DEFAULT_SCOPE           = $defaultScope
    TENANT_DSN              = $tenantDsn
    PLATFORM_DSN            = $platformDsn
    PG_CTL_EXE              = $pgCtlExe
    PSQL_EXE                = $psqlExe
    PG_ISREADY_EXE          = $pgIsReadyExe
    INITDB_EXE              = $initdbExe
    DB_INIT_SCRIPT          = $dbInitScript
    DB_MIGRATE_SCRIPT       = $dbMigrateScript
}

if ($WriteEnvFile) {
    $postgresBin = Join-Path $installPath "bin"
    $lines = @(
        "# Generated by read-postgres-config.ps1 from config.ini",
        "POSTGRES_HOST=$hostName",
        "POSTGRES_HOST_PORT=$port",
        "POSTGRES_SUPERUSER=$superUser",
        "POSTGRES_SUPERUSER_PASSWORD=$superPassword",
        "POSTGRES_USER=$appUser",
        "POSTGRES_PASSWORD=$appPassword",
        "POSTGRES_DB_EDGE=$dbEdge",
        "POSTGRES_DB_CLOUD=$dbCloud",
        "POSTGRES_DB=$dbEdge"
    )
    if (Test-Path (Join-Path $postgresBin "psql.exe")) {
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
        if ($kv.Key -match 'PASSWORD|DSN') { continue }
        Write-Host ("  {0} = {1}" -f $kv.Key, $kv.Value)
    }
    Write-Host ("  TENANT_DSN = postgres://{0}:****@{1}:{2}/{3}?sslmode=disable" -f $appUser, $hostName, $port, $dbEdge)
    return
}

if ($EmitBat) {
    foreach ($kv in $config.GetEnumerator()) {
        $val = $kv.Value -replace '"', '""'
        Write-Output ("set `"{0}={1}`"" -f $kv.Key, $val)
    }
    return
}

return [pscustomobject]$config
