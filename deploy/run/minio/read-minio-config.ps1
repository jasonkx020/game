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
$installPath = Resolve-ConfigPath $scriptRoot (Read-IniValue "MinIO" "install_path" ".") $scriptRoot
$dataPath = Resolve-ConfigPath $scriptRoot (Read-IniValue "MinIO" "data_path" "data") (Join-Path $scriptRoot "data")
$apiPort = Read-IniValue "MinIO" "api_port" "9000"
$consolePort = Read-IniValue "MinIO" "console_port" "9001"
$listenBind = Read-IniValue "MinIO" "listen_bind" "127.0.0.1"
$bucket = Read-IniValue "MinIO" "bucket" "freetv-assets"
$addFirewall = (Read-IniValue "MinIO" "add_firewall_rules" "true").ToLower()
$browserRedirect = Read-IniValue "MinIO" "browser_redirect_url" ""

$rootUser = Read-IniValue "Credentials" "root_user" "ycwlminio"
$rootPassword = Read-IniValue "Credentials" "root_password" "YcwlMinioDev123"

$mcAlias = Read-IniValue "Client" "alias" "ycwl-minio"
$endpointRaw = Read-IniValue "Client" "endpoint" ""
if ([string]::IsNullOrWhiteSpace($endpointRaw)) {
    $endpoint = "http://${listenBind}:${apiPort}"
} else {
    $endpoint = $endpointRaw.TrimEnd('/')
}

$minioExe = Resolve-ExePath @(
    (Join-Path $installPath "bin\minio.exe"),
    (Join-Path $installPath "minio.exe"),
    (Join-Path $scriptRoot "bin\minio.exe"),
    (Join-Path $scriptRoot "minio.exe")
)
$mcExe = Resolve-ExePath @(
    (Join-Path $installPath "bin\mc.exe"),
    (Join-Path $installPath "mc.exe"),
    (Join-Path $scriptRoot "bin\mc.exe"),
    (Join-Path $scriptRoot "mc.exe")
)

$config = [ordered]@{
    SCRIPT_DIR       = $scriptRoot
    CONFIG_FILE      = $ConfigIni
    INSTALL_PATH     = $installPath
    DATA_PATH        = $dataPath
    API_PORT         = $apiPort
    CONSOLE_PORT     = $consolePort
    LISTEN_BIND      = $listenBind
    BUCKET           = $bucket
    ADD_FIREWALL     = $addFirewall
    BROWSER_REDIRECT = $browserRedirect
    MINIO_ROOT_USER  = $rootUser
    MINIO_ROOT_PASSWORD = $rootPassword
    MINIO_ALIAS      = $mcAlias
    MINIO_ENDPOINT   = $endpoint
    MINIO_EXE        = $minioExe
    MC_EXE           = $mcExe
}

if ($PrintSummary) {
    Write-Host "MinIO config ($ConfigIni)"
    foreach ($kv in $config.GetEnumerator()) {
        if ($kv.Key -match 'PASSWORD') { continue }
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
