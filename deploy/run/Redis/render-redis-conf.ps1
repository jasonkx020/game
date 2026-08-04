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

function Set-ConfLine {
    param(
        [string]$Content,
        [string]$Key,
        [string]$Value,
        [switch]$CommentOut
    )
    $escaped = [regex]::Escape($Key)
    $activePattern = "(?m)^$escaped\s+.*$"
    if ($CommentOut) {
        if ([regex]::IsMatch($Content, $activePattern)) {
            return [regex]::Replace($Content, $activePattern, "# $Key")
        }
        return $Content
    }
    $replacement = "$Key $Value"
    if ([regex]::IsMatch($Content, $activePattern)) {
        return [regex]::Replace($Content, $activePattern, $replacement)
    }
    $commentPattern = "(?m)^#\s*$escaped\b.*$"
    if ([regex]::IsMatch($Content, $commentPattern)) {
        return [regex]::Replace($Content, $commentPattern, $replacement, 1)
    }
    return $Content + "`r`n$replacement"
}

$cfg = & (Join-Path $PSScriptRoot "read-redis-config.ps1") -ConfigIni $ConfigIni
if (-not $cfg) { throw "failed to load config" }

if (-not $Template) { $Template = $cfg.CONFIG_TEMPLATE }
if (-not $Output) { $Output = $cfg.CONF_OUTPUT }

if (-not (Test-Path $Template)) {
    throw "template not found: $Template"
}

New-Item -ItemType Directory -Force -Path $cfg.DATA_PATH | Out-Null
New-Item -ItemType Directory -Force -Path $cfg.LOG_PATH | Out-Null

$dataDir = $cfg.DATA_PATH -replace '\\', '/'
$logFile = $cfg.LOG_FILE -replace '\\', '/'

$content = Get-Content $Template -Raw -Encoding UTF8
$content = Set-ConfLine $content "bind" $cfg.LISTEN_BIND
$content = Set-ConfLine $content "port" $cfg.PORT
$content = Set-ConfLine $content "dir" $dataDir
$content = Set-ConfLine $content "logfile" "`"$logFile`""
$content = Set-ConfLine $content "appendonly" $(if ($cfg.APPENDONLY -eq "true") { "yes" } else { "no" })

if ([string]::IsNullOrWhiteSpace($cfg.REDIS_PASSWORD)) {
    $content = Set-ConfLine $content "requirepass" "" -CommentOut
} else {
    $content = Set-ConfLine $content "requirepass" $cfg.REDIS_PASSWORD
}

Write-Utf8NoBomFile -Path $Output -Content $content
Write-Host "Rendered: $Output"
