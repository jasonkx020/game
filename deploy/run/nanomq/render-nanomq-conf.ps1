param(
    [string]$ConfigIni = (Join-Path $PSScriptRoot "config.ini"),
    [string]$Template = "",
    [string]$Output = ""
)

function Write-Utf8NoBomFile {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][string]$Content
    )
    $dir = Split-Path -Parent $Path
    if ($dir -and -not (Test-Path $dir)) {
        New-Item -ItemType Directory -Force -Path $dir | Out-Null
    }
    $utf8NoBom = New-Object System.Text.UTF8Encoding $false
    [System.IO.File]::WriteAllText($Path, $Content, $utf8NoBom)
}

$cfg = & (Join-Path $PSScriptRoot "read-nanomq-config.ps1") -ConfigIni $ConfigIni
if (-not $cfg) { throw "failed to load config" }

if (-not $Template) { $Template = $cfg.CONF_TEMPLATE }
if (-not $Output) { $Output = $cfg.CONF_OUTPUT }

if (-not (Test-Path $Template)) {
    throw "template not found: $Template"
}

New-Item -ItemType Directory -Force -Path (Split-Path $Output) | Out-Null
New-Item -ItemType Directory -Force -Path $cfg.LOG_DIR | Out-Null

$logDirUnix = ($cfg.LOG_DIR -replace '\\', '/')
$certUnix = ($cfg.TLS_CERT_FILE -replace '\\', '/')
$keyUnix = ($cfg.TLS_KEY_FILE -replace '\\', '/')
$caUnix = ($cfg.TLS_CA_FILE -replace '\\', '/')

$content = Get-Content $Template -Raw -Encoding UTF8
$content = $content.Replace("__MQTT_PORT__", $cfg.MQTT_PORT)
$content = $content.Replace("__SSL_PORT__", $cfg.SSL_PORT)
$content = $content.Replace("__HTTP_PORT__", $cfg.HTTP_PORT)
$content = $content.Replace("__LOG_LEVEL__", $cfg.LOG_LEVEL)
$content = $content.Replace("__LOG_DIR__", $logDirUnix)
$content = $content.Replace("__TLS_CERT_FILE__", $certUnix)
$content = $content.Replace("__TLS_KEY_FILE__", $keyUnix)
$content = $content.Replace("__TLS_CA_FILE__", $caUnix)
$content = $content.Replace("__AUTH_URL__", $cfg.AUTH_URL)
$content = $content.Replace("__SUPER_URL__", $cfg.SUPER_URL)
$content = $content.Replace("__ACL_URL__", $cfg.ACL_URL)

Write-Utf8NoBomFile -Path $Output -Content $content
Write-Host "Rendered: $Output"
