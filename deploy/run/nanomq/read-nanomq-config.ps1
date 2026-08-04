param(
    [string]$ConfigIni = (Join-Path $PSScriptRoot "config.ini"),
    [switch]$EmitBat,
    [switch]$PrintSummary,
    [string]$WriteEnvFile
)

function Read-IniFile {
    param([string]$Path)
    $sections = @{}
    if (-not (Test-Path $Path)) { return $sections }
    $current = ""
    foreach ($line in Get-Content $Path -ErrorAction SilentlyContinue) {
        $trim = $line.Trim()
        if ($trim -eq "" -or $trim -match '^\s*[;#]') { continue }
        if ($trim -match '^\[(.+)\]$') {
            $current = $matches[1].Trim()
            if (-not $sections.ContainsKey($current)) {
                $sections[$current] = @{}
            }
            continue
        }
        if ($current -eq "") { continue }
        if ($trim -match '^([^=]+)=(.*)$') {
            $sections[$current][$matches[1].Trim()] = $matches[2].Trim()
        }
    }
    return $sections
}

function Get-IniValue {
    param(
        [hashtable]$Ini,
        [string]$Section,
        [string]$Key,
        [string]$Default = ""
    )
    if ($Ini.ContainsKey($Section) -and $Ini[$Section].ContainsKey($Key)) {
        $value = [string]$Ini[$Section][$Key]
        if ($value -ne "") { return $value }
    }
    return $Default
}

function Get-IniBool {
    param(
        [hashtable]$Ini,
        [string]$Section,
        [string]$Key,
        [bool]$Default = $false
    )
    $raw = Get-IniValue $Ini $Section $Key ""
    if ($raw -eq "") { return $Default }
    switch ($raw.ToLower()) {
        { $_ -in @("1", "true", "yes", "on") } { return $true }
        { $_ -in @("0", "false", "no", "off") } { return $false }
        default { return $Default }
    }
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

function Resolve-NanoMQInstallPath {
    param([string]$Base, [string]$Configured)
    $resolved = Resolve-ConfigPath $Base $Configured $Base
    $trim = if ($Configured) { $Configured.Trim() } else { "." }
    if ($trim -ne "." -and $trim -ne ".\") {
        return $resolved
    }
    if (Test-Path (Join-Path $Base "bin\nanomq.exe")) {
        return $Base
    }
    return $resolved
}

function Resolve-TlsPath {
    param([string]$InstallPath, [string]$Path)
    if ([string]::IsNullOrWhiteSpace($Path)) { return "" }
    return Resolve-ConfigPath $InstallPath $Path $Path
}

function Get-NanoMQExe {
    param([string]$InstallPath, [string]$ScriptRoot)
    $candidates = @(
        (Join-Path $InstallPath "bin\nanomq.exe"),
        (Join-Path $InstallPath "nanomq.exe"),
        (Join-Path $ScriptRoot "bin\nanomq.exe")
    )
    foreach ($c in $candidates) {
        if ($c -and (Test-Path $c)) { return $c }
    }
    $cmd = Get-Command nanomq -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }
    return ""
}

$scriptRoot = $PSScriptRoot
$repoRoot = (Resolve-Path (Join-Path $scriptRoot "..\..\..")).Path
$ini = Read-IniFile $ConfigIni

$installPath = Resolve-NanoMQInstallPath $scriptRoot (Get-IniValue $ini "NanoMQ" "install_path" ".")
$logDir = Resolve-ConfigPath $scriptRoot (Get-IniValue $ini "NanoMQ" "log_dir" "log") (Join-Path $installPath "log")
$mqttPort = Get-IniValue $ini "NanoMQ" "mqtt_port" "1883"
$sslPort = Get-IniValue $ini "NanoMQ" "ssl_port" "8883"
$sslEnabled = Get-IniBool $ini "NanoMQ" "ssl_enabled" $false
$httpPort = Get-IniValue $ini "NanoMQ" "http_port" "8081"
$wsPort = Get-IniValue $ini "NanoMQ" "ws_port" "8083"
$allowAnonymous = Get-IniBool $ini "NanoMQ" "allow_anonymous" $false
$logLevel = Get-IniValue $ini "NanoMQ" "log_level" "info"
$addFirewall = Get-IniBool $ini "NanoMQ" "add_firewall_rules" $true

$tenantApiBase = (Get-IniValue $ini "TenantAPI" "base_url" "http://127.0.0.1:8443").TrimEnd('/')
$healthPath = Get-IniValue $ini "TenantAPI" "health_path" "/api/v1/health"
$authPath = Get-IniValue $ini "TenantAPI" "auth_path" "/api/v1/mqtt/auth"
$superPath = Get-IniValue $ini "TenantAPI" "superuser_path" "/api/v1/mqtt/superuser"
$aclPath = Get-IniValue $ini "TenantAPI" "acl_path" "/api/v1/mqtt/acl"
$checkHealth = Get-IniBool $ini "TenantAPI" "check_health_before_start" $true

$certFile = Resolve-TlsPath $installPath (Get-IniValue $ini "TLS" "cert_file" "")
$keyFile = Resolve-TlsPath $installPath (Get-IniValue $ini "TLS" "key_file" "")
$caFile = Resolve-TlsPath $installPath (Get-IniValue $ini "TLS" "ca_file" "")

$hubUsername = Get-IniValue $ini "Hub" "username" "gouser"
$hubPassword = Get-IniValue $ini "Hub" "password" ""

$nanomqExe = Get-NanoMQExe $installPath $scriptRoot
$confTemplate = Join-Path $scriptRoot "nanomq.conf.template"
$confOutput = Join-Path $installPath "etc\nanomq.conf"
$renderScript = Join-Path $scriptRoot "render-nanomq-conf.ps1"

$healthUrl = "$tenantApiBase$healthPath"
$authUrl = "$tenantApiBase$authPath"
$superUrl = "$tenantApiBase$superPath"
$aclUrl = "$tenantApiBase$aclPath"

$config = [ordered]@{
    SCRIPT_DIR           = $scriptRoot
    REPO_ROOT            = $repoRoot
    CONFIG_FILE          = $ConfigIni
    INSTALL_PATH         = $installPath
    NANOMQ_PATH          = $installPath
    NANOMQ_BIN           = $nanomqExe
    NANOMQ_CONFIG        = $confOutput
    CONF_TEMPLATE        = $confTemplate
    CONF_OUTPUT          = $confOutput
    RENDER_SCRIPT        = $renderScript
    MQTT_PORT            = $mqttPort
    SSL_PORT             = $sslPort
    SSL_ENABLED          = if ($sslEnabled) { "true" } else { "false" }
    HTTP_PORT            = $httpPort
    WS_PORT              = $wsPort
    ALLOW_ANONYMOUS      = if ($allowAnonymous) { "true" } else { "false" }
    LOG_LEVEL            = $logLevel
    LOG_DIR              = $logDir
    NANOMQ_LOG_DIR       = $logDir
    TENANT_API_BASE      = $tenantApiBase
    HEALTH_PATH          = $healthPath
    HEALTH_URL           = $healthUrl
    AUTH_PATH            = $authPath
    AUTH_URL             = $authUrl
    SUPER_PATH           = $superPath
    SUPER_URL            = $superUrl
    ACL_PATH             = $aclPath
    ACL_URL              = $aclUrl
    CHECK_HEALTH         = if ($checkHealth) { "true" } else { "false" }
    TLS_CERT_FILE        = $certFile
    TLS_KEY_FILE         = $keyFile
    TLS_CA_FILE          = $caFile
    HUB_USERNAME         = $hubUsername
    HUB_PASSWORD         = $hubPassword
    ADD_FIREWALL         = if ($addFirewall) { "true" } else { "false" }
}

if ($WriteEnvFile) {
    $lines = @(
        "# Generated by read-nanomq-config.ps1 from config.ini",
        "NANOMQ_PATH=$installPath",
        "NANOMQ_BIN=$nanomqExe",
        "NANOMQ_CONFIG=$confOutput",
        "MQTT_PORT=$mqttPort",
        "SSL_PORT=$sslPort",
        "SSL_ENABLED=$($config.SSL_ENABLED)",
        "HTTP_PORT=$httpPort",
        "WS_PORT=$wsPort",
        "LOG_LEVEL=$logLevel",
        "LOG_DIR=$logDir",
        "TENANT_API_BASE=$tenantApiBase",
        "HEALTH_URL=$healthUrl",
        "AUTH_URL=$authUrl",
        "SUPER_URL=$superUrl",
        "ACL_URL=$aclUrl",
        "CHECK_HEALTH=$($config.CHECK_HEALTH)",
        "HUB_USERNAME=$hubUsername"
    )
    if ($hubPassword) {
        $lines += "HUB_PASSWORD=$hubPassword"
    }
    if ($certFile) { $lines += "TLS_CERT_FILE=$certFile" }
    if ($keyFile) { $lines += "TLS_KEY_FILE=$keyFile" }
    if ($caFile) { $lines += "TLS_CA_FILE=$caFile" }
    $utf8NoBom = New-Object System.Text.UTF8Encoding $false
    [System.IO.File]::WriteAllText($WriteEnvFile, ($lines -join "`r`n") + "`r`n", $utf8NoBom)
    Write-Host "Wrote: $WriteEnvFile"
    return
}

if ($PrintSummary) {
    Write-Host "NanoMQ config ($ConfigIni)"
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
