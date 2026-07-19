#Requires -Version 5.1
<#
.SYNOPSIS
  Serve game Remote Bundles locally with CORS + correct JS MIME (default :8787).

.EXAMPLE
  .\scripts\serve-bundles.ps1
#>
param(
    [int]$Port = 8787,
    [string]$Root = '',
    [switch]$NoSync
)

$ErrorActionPreference = 'Stop'
if (-not $Root) {
    $Root = Join-Path (Resolve-Path (Join-Path $PSScriptRoot '..')).Path 'client\build\bundles'
}
New-Item -ItemType Directory -Force -Path $Root | Out-Null

if (-not $NoSync) {
    $sync = Join-Path $PSScriptRoot 'sync-bundles.ps1'
    if (Test-Path $sync) {
        $remote = Join-Path (Resolve-Path (Join-Path $PSScriptRoot '..')).Path 'client\build\web-desktop\remote'
        if (Test-Path $remote) {
            Write-Host 'Syncing web-desktop/remote -> bundles ...'
            & $sync
        } else {
            Write-Host 'Skip sync: remote build not found (Creator build first, or -NoSync)'
        }
    }
}

# Lobby seed URL = http://localhost:8787/bundles/{gameId}
$serveRoot = $Root
if ((Split-Path $Root -Leaf) -eq 'bundles') {
    $parent = Split-Path $Root -Parent
    if (Test-Path $parent) { $serveRoot = $parent }
}

$corsPy = Join-Path $PSScriptRoot 'serve_bundles_cors.py'
if (-not (Test-Path $corsPy)) {
    Write-Error "Missing $corsPy"
}

$py = Get-Command python -ErrorAction SilentlyContinue
if (-not $py) {
    Write-Error 'Python 3 required to serve static bundles'
}

Write-Host ("HTTP root: {0}" -f $serveRoot)
Write-Host ("Expect: http://localhost:{0}/bundles/liuzichong/index.1.0.0.js" -f $Port)
Write-Host 'CORS: *  |  .js Content-Type: application/javascript'

& python $corsPy $serveRoot $Port
