param(
    [string]$ConfigIni = (Join-Path $PSScriptRoot "config.ini"),
    [switch]$EmitBat,
    [switch]$PrintSummary
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

function Resolve-ExePath {
    param([string[]]$Candidates)
    foreach ($c in $Candidates) {
        if ($c -and (Test-Path $c)) { return $c }
    }
    return $Candidates[-1]
}

$scriptRoot = $PSScriptRoot
$installPath = Resolve-ConfigPath $scriptRoot (Read-IniValue "Redis" "install_path" ".") $scriptRoot
$dataPath = Resolve-ConfigPath $scriptRoot (Read-IniValue "Redis" "data_path" "data") (Join-Path $scriptRoot "data")
$logPath = Resolve-ConfigPath $scriptRoot (Read-IniValue "Redis" "log_path" "logs") (Join-Path $scriptRoot "logs")
$port = Read-IniValue "Redis" "port" "6379"
$listenBind = Read-IniValue "Redis" "listen_bind" "127.0.0.1"
$configTemplate = Resolve-ConfigPath $scriptRoot (Read-IniValue "Redis" "config_template" "redis.windows.conf") (Join-Path $scriptRoot "redis.windows.conf")
$confOutput = Resolve-ConfigPath $scriptRoot (Read-IniValue "Redis" "conf_output" "conf\redis.conf") (Join-Path $scriptRoot "conf\redis.conf")
$appendOnly = (Read-IniValue "Redis" "appendonly" "true").ToLower()
$addFirewall = (Read-IniValue "Redis" "add_firewall_rules" "true").ToLower()
$runMode = (Read-IniValue "Redis" "run_mode" "background").ToLower()

$password = Read-IniValue "Credentials" "password" ""
$cliAlias = Read-IniValue "Client" "alias" "ycwl-redis"

$redisServerExe = Resolve-ExePath @(
    (Join-Path $installPath "bin\redis-server.exe"),
    (Join-Path $installPath "redis-server.exe"),
    (Join-Path $scriptRoot "redis-server.exe")
)
$redisCliExe = Resolve-ExePath @(
    (Join-Path $installPath "bin\redis-cli.exe"),
    (Join-Path $installPath "redis-cli.exe"),
    (Join-Path $scriptRoot "redis-cli.exe")
)

$logFile = Join-Path $logPath "redis.log"
$renderScript = Join-Path $scriptRoot "render-redis-conf.ps1"
$redisAddr = "${listenBind}:${port}"

$config = [ordered]@{
    SCRIPT_DIR         = $scriptRoot
    CONFIG_FILE        = $ConfigIni
    INSTALL_PATH       = $installPath
    DATA_PATH          = $dataPath
    LOG_PATH           = $logPath
    LOG_FILE           = $logFile
    PORT               = $port
    LISTEN_BIND        = $listenBind
    REDIS_ADDR         = $redisAddr
    CONFIG_TEMPLATE    = $configTemplate
    CONF_OUTPUT        = $confOutput
    APPENDONLY         = $appendOnly
    ADD_FIREWALL       = $addFirewall
    RUN_MODE           = $runMode
    REDIS_PASSWORD     = $password
    REDIS_ALIAS        = $cliAlias
    REDIS_SERVER_EXE   = $redisServerExe
    REDIS_CLI_EXE      = $redisCliExe
    RENDER_SCRIPT      = $renderScript
}

if ($PrintSummary) {
    Write-Host "Redis config ($ConfigIni)"
    foreach ($kv in $config.GetEnumerator()) {
        if ($kv.Key -match 'PASSWORD') {
            if ([string]::IsNullOrWhiteSpace($kv.Value)) {
                Write-Host ("  {0} = (empty)" -f $kv.Key)
            } else {
                Write-Host ("  {0} = ********" -f $kv.Key)
            }
            continue
        }
        Write-Host ("  {0} = {1}" -f $kv.Key, $kv.Value)
    }
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
