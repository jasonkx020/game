param(
    [string]$ConfigIni = (Join-Path $PSScriptRoot "config.ini")
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

$cfg = & (Join-Path $PSScriptRoot "read-postgres-config.ps1") -ConfigIni $ConfigIni
if (-not $cfg) { throw "failed to load config" }

$templateConf = Join-Path $PSScriptRoot "postgresql.conf.template"
$templateHba = Join-Path $PSScriptRoot "pg_hba.conf.template"
$outputConf = Join-Path $cfg.DATA_PATH "postgresql.conf"
$outputHba = Join-Path $cfg.DATA_PATH "pg_hba.conf"

New-Item -ItemType Directory -Force -Path $cfg.DATA_PATH | Out-Null
New-Item -ItemType Directory -Force -Path $cfg.LOG_DIR | Out-Null

$logDirUnix = ($cfg.LOG_DIR -replace '\\', '/')
$superPassword = [string]$cfg.POSTGRES_SUPERUSER_PASSWORD
$hasSuperPassword = -not [string]::IsNullOrWhiteSpace($superPassword)
$localAuth = if ($hasSuperPassword) { "scram-sha-256" } else { "trust" }
$authHint = if ($hasSuperPassword) {
    "# scram-sha-256 (superuser password configured in config.ini)"
} else {
    "# trust on 127.0.0.1 when [Superuser] password is empty (local dev only)"
}

if (Test-Path $templateConf) {
    $content = Get-Content $templateConf -Raw -Encoding UTF8
    $content = $content.Replace("__LISTEN_BIND__", $cfg.LISTEN_BIND)
    $content = $content.Replace("__PORT__", $cfg.PG_PORT)
    $content = $content.Replace("__LOG_DIR__", $logDirUnix)
    Write-Utf8NoBomFile -Path $outputConf -Content $content
    Write-Host "Rendered: $outputConf"
}

if (Test-Path $templateHba) {
    $hba = Get-Content $templateHba -Raw -Encoding UTF8
    $hba = $hba.Replace("__AUTH_HINT__", $authHint)
    $hba = $hba.Replace("__LOCAL_AUTH__", $localAuth)
    Write-Utf8NoBomFile -Path $outputHba -Content $hba
    Write-Host "Rendered: $outputHba (local auth: $localAuth)"
}
