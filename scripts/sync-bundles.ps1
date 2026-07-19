#Requires -Version 5.1
<#
.SYNOPSIS
  Sync Cocos remote bundles: web-desktop/remote/{gameId} -> build/bundles/{gameId}

.EXAMPLE
  .\scripts\sync-bundles.ps1
  .\scripts\sync-bundles.ps1 -Version 1.0.0
#>
param(
    [string]$Version = '1.0.0',
    [string]$RemoteRoot = '',
    [string]$BundleRoot = ''
)

$ErrorActionPreference = 'Stop'
$repo = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
if (-not $RemoteRoot) {
    $RemoteRoot = Join-Path $repo 'client\build\web-desktop\remote'
}
if (-not $BundleRoot) {
    $BundleRoot = Join-Path $repo 'client\build\bundles'
}

if (-not (Test-Path $RemoteRoot)) {
    Write-Error ("Remote build not found: {0}. Build Web Desktop in Cocos Creator first." -f $RemoteRoot)
}

New-Item -ItemType Directory -Force -Path $BundleRoot | Out-Null
$gameDirs = Get-ChildItem $RemoteRoot -Directory -ErrorAction SilentlyContinue
if (-not $gameDirs) {
    Write-Error ("No game folders under: {0}" -f $RemoteRoot)
}

foreach ($g in $gameDirs) {
    $dst = Join-Path $BundleRoot $g.Name
    if (Test-Path $dst) {
        Remove-Item -Recurse -Force $dst
    }
    New-Item -ItemType Directory -Force -Path $dst | Out-Null
    Copy-Item -Recurse -Force (Join-Path $g.FullName '*') $dst

    $index = Join-Path $dst 'index.js'
    $config = Join-Path $dst 'config.json'
    if (Test-Path $index) {
        Copy-Item -Force $index (Join-Path $dst ("index.{0}.js" -f $Version))
    }
    if (Test-Path $config) {
        Copy-Item -Force $config (Join-Path $dst ("config.{0}.json" -f $Version))
    }

    $hasIndex = Test-Path (Join-Path $dst ("index.{0}.js" -f $Version))
    Write-Host ("Synced {0} -> bundles/{0} (index.{1}.js={2})" -f $g.Name, $Version, $hasIndex)
}

Write-Host 'Done. Next: .\scripts\dev.ps1 serve-bundles'
