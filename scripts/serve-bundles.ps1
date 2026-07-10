#Requires -Version 5.1
<#
.SYNOPSIS
  本地托管游戏 Remote Bundle（开发用，默认 :8787）

.EXAMPLE
  .\scripts\serve-bundles.ps1
#>
param(
    [int]$Port = 8787,
    [string]$Root = ""
)

$ErrorActionPreference = 'Stop'
if (-not $Root) {
    $Root = Join-Path (Resolve-Path (Join-Path $PSScriptRoot '..')).Path 'client\build\bundles'
}
New-Item -ItemType Directory -Force -Path $Root | Out-Null

Write-Host "Serving bundles from $Root on http://localhost:$Port/"
Write-Host "Build bundles with Cocos Creator (Remote) or place stub folders under client/build/bundles/{gameId}"

# 简易静态文件服务（Python 优先，否则提示）
$py = Get-Command python -ErrorAction SilentlyContinue
if ($py) {
    Push-Location $Root
    try {
        & python -m http.server $Port
    } finally {
        Pop-Location
    }
} else {
    Write-Error '需要 Python 3 以启动静态服务，或自行托管 client/build/bundles'
}
