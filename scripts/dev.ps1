#Requires -Version 5.1
<#
.SYNOPSIS
  Windows 本地开发脚本（生产构建请用 Makefile / server/deploy/Dockerfile，目标 Linux）

.EXAMPLE
  .\scripts\dev.ps1 up
  .\scripts\dev.ps1 seed-dev
  .\scripts\dev.ps1 run-game-api
  .\scripts\dev.ps1 run-game
#>
param(
    [Parameter(Position = 0)]
    [ValidateSet(
        'help', 'up', 'down', 'migrate', 'migrate-status', 'migrate-down', 'migrate-docker', 'seed-dev',
        'run-game-api', 'run-auth', 'run-game', 'run-admin-api', 'run-admin', 'run-admin-web', 'test', 'lint', 'tidy',
        'gen-proto', 'gen-client-proto', 'build-linux', 'docker-build', 'serve-bundles'
    )]
    [string]$Command = 'help',
    [int]$Steps = 1
)

# Console UTF-8 so Chinese help renders under Windows PowerShell 5.1
try {
    [Console]::OutputEncoding = [System.Text.UTF8Encoding]::new()
    $OutputEncoding = [Console]::OutputEncoding
} catch { }

$ErrorActionPreference = 'Stop'
$Root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$Server = Join-Path $Root 'server'
$Auth = Join-Path $Root 'auth-service'
$Admin = Join-Path $Root 'admin'
$Client = Join-Path $Root 'client'

function Import-DotEnv {
    $envFile = Join-Path $Server '.env'
    if (-not (Test-Path $envFile)) {
        $envFile = Join-Path $Root '.env'
    }
    if (-not (Test-Path $envFile)) { return }
    Get-Content $envFile | ForEach-Object {
        $line = $_.Trim()
        if ($line -eq '' -or $line.StartsWith('#')) { return }
        $idx = $line.IndexOf('=')
        if ($idx -lt 1) { return }
        $key = $line.Substring(0, $idx).Trim()
        $val = $line.Substring($idx + 1).Trim()
        Set-Item -Path "Env:$key" -Value $val
    }
}

function Get-MigrateDatabaseUrl {
    Import-DotEnv
    $url = $env:MIGRATE_DATABASE_URL
    if (-not $url) { $url = $env:DATABASE_URL }
    if (-not $url) {
        $url = 'postgres://game:game@localhost:5432/game?sslmode=disable'
    }
    return $url
}

function Invoke-Migrate {
    param([Parameter(ValueFromRemainingArguments = $true)][string[]]$MigrateArgs)
    Import-DotEnv
    $env:MIGRATE_DATABASE_URL = Get-MigrateDatabaseUrl
    Push-Location $Server
    try {
        go run ./cmd/migrate @MigrateArgs
    } finally {
        Pop-Location
    }
}

function Get-BufExe {
    $candidates = @(
        (Join-Path $(go env GOPATH) 'bin\buf.exe'),
        (Join-Path $env:USERPROFILE 'go\bin\buf.exe')
    )
    $cmd = Get-Command buf -ErrorAction SilentlyContinue
    if ($cmd -and $cmd.Source -and (Test-Path -LiteralPath $cmd.Source)) {
        $candidates += $cmd.Source
    }

    foreach ($path in $candidates) {
        if ($path -and (Test-Path -LiteralPath $path)) {
            return (Resolve-Path -LiteralPath $path).Path
        }
    }

    throw @"
未找到 buf CLI。请安装后重试:
  `$env:GOOS='windows'; `$env:GOARCH='amd64'
  go install github.com/bufbuild/buf/cmd/buf@v1.71.0
安装后 buf 位于 %USERPROFILE%\go\bin\buf.exe（本脚本会自动查找）
"@
}

function Invoke-BufGenerate {
    param([string]$Template = '')
    $buf = Get-BufExe
    Push-Location (Join-Path $Server 'proto')
    try {
        if ($Template) {
            & $buf generate --template $Template
        } else {
            & $buf generate
        }
    } finally {
        Pop-Location
    }
}

function Invoke-MigrateDocker {
    param([Parameter(ValueFromRemainingArguments = $true)][string[]]$MigrateArgs)
    $db = Get-MigrateDatabaseUrl
    if ($db -match '@localhost:' -or $db -match '@127\.0\.0\.1:') {
        $db = $db -replace '@localhost:', '@host.docker.internal:'
        $db = $db -replace '@127\.0\.0\.1:', '@host.docker.internal:'
    }
    Write-Host "migrate database (docker): $db"
    docker run --rm `
        -v "${Server}/migrations:/migrations" `
        migrate/migrate `
        -path=/migrations `
        -database $db `
        @MigrateArgs
}

function Show-Help {
    @"
Windows 开发命令（在仓库根目录执行）:

  .\scripts\dev.ps1 up              启动 Postgres + Redis (Docker)
  .\scripts\dev.ps1 down            停止基础设施
  .\scripts\dev.ps1 migrate-status   查看迁移状态（当前/最新/待执行）
  .\scripts\dev.ps1 migrate         升级到最新版本
  .\scripts\dev.ps1 migrate-down    回滚一步（-Steps 2 回滚两步）
  .\scripts\dev.ps1 seed-dev        同 migrate（升级 + 种子已在迁移中）
  .\scripts\dev.ps1 migrate-docker  在 Docker 内执行（可选，需 Docker）
  .\scripts\dev.ps1 run-game-api    启动 game-api (:8080)（玩家路径，不依赖 admin）
  .\scripts\dev.ps1 run-auth        启动 auth-api (:8081) 并应用 auth migrations
  .\scripts\dev.ps1 run-game        启动 Pitaya game (:3250)
  .\scripts\dev.ps1 run-admin       构建前端并启动 admin-api（:8082，API+SPA 同端口）
  .\scripts\dev.ps1 run-admin-api   仅启动 admin-api（:8082，需已有 web 构建产物）
  .\scripts\dev.ps1 run-admin-web   Vite HMR (:5173，需另开 run-admin-api)
  .\scripts\dev.ps1 test            go test ./...（server/）
  .\scripts\dev.ps1 lint            游戏模块 import 边界检查（server/）
  .\scripts\dev.ps1 gen-proto       生成 Go proto（自动查找 buf.exe）
  .\scripts\dev.ps1 gen-client-proto  生成客户端 TS proto
  .\scripts\dev.ps1 build-linux     交叉编译 Linux amd64 二进制到 server/bin/
  .\scripts\dev.ps1 docker-build    构建 Linux 生产镜像
  .\scripts\dev.ps1 serve-bundles   本地托管游戏 Bundle (:8787)

目录：server/（游戏）+ auth-service/ + admin/（运营）+ client/
VS Code: 使用 Run and Debug -> Server (api + game)

Linux 生产: make build-linux / make docker-build，见 server/deploy/
"@
}

Set-Location $Root
Import-DotEnv

switch ($Command) {
    'help' { Show-Help }
    'up' {
        docker compose -f (Join-Path $Server 'deploy/docker-compose.yml') up -d
    }
    'down' {
        docker compose -f (Join-Path $Server 'deploy/docker-compose.yml') down
    }
    'migrate' { Invoke-Migrate 'up' }
    'migrate-status' { Invoke-Migrate 'status' }
    'seed-dev' { Invoke-Migrate 'up' }
    'migrate-down' { Invoke-Migrate 'down', "$Steps" }
    'migrate-docker' { Invoke-MigrateDocker 'up' }
    'run-game-api' {
        Push-Location $Server
        try { go run ./cmd/game-api } finally { Pop-Location }
    }
    'run-auth' {
        Import-DotEnv
        # Prefer auth-service/.env; fall back to server/.env already loaded
        $authEnv = Join-Path $Auth '.env'
        if (Test-Path $authEnv) {
            Get-Content $authEnv | ForEach-Object {
                $line = $_.Trim()
                if ($line -eq '' -or $line.StartsWith('#')) { return }
                $idx = $line.IndexOf('=')
                if ($idx -lt 1) { return }
                Set-Item -Path ("Env:" + $line.Substring(0, $idx).Trim()) -Value $line.Substring($idx + 1).Trim()
            }
        }
        Push-Location $Auth
        try {
            go run ./cmd/migrate -path migrations
            go run ./cmd/auth-api
        } finally { Pop-Location }
    }
    'run-game' {
        Push-Location $Server
        try { go run ./cmd/game } finally { Pop-Location }
    }
    'run-admin-api' {
        Push-Location $Admin
        try { go run ./cmd/admin-api } finally { Pop-Location }
    }
    'run-admin' {
        $web = Join-Path $Admin 'web'
        Push-Location $web
        try {
            if (-not (Test-Path 'node_modules')) { npm install }
            npm run build
        } finally { Pop-Location }
        Push-Location $Admin
        try { go run ./cmd/admin-api } finally { Pop-Location }
    }
    'run-admin-web' {
        Push-Location (Join-Path $Admin 'web')
        try { npm run dev } finally { Pop-Location }
    }
    'test' {
        Push-Location $Server
        try { go test ./... } finally { Pop-Location }
    }
    'lint' {
        Push-Location $Server
        try { go run ./cmd/check-boundaries } finally { Pop-Location }
    }
    'tidy' {
        Push-Location $Server
        try { go mod tidy } finally { Pop-Location }
    }
    'gen-proto' { Invoke-BufGenerate }
    'gen-client-proto' {
        Invoke-BufGenerate -Template 'buf.gen.client.yaml'
        Push-Location $Client
        try { node scripts/patch-proto-long.mjs } finally { Pop-Location }
    }
    'build-linux' {
        Push-Location $Server
        try {
            $env:CGO_ENABLED = '0'
            $env:GOOS = 'linux'
            $env:GOARCH = 'amd64'
            New-Item -ItemType Directory -Force -Path bin | Out-Null
            go build -ldflags="-s -w" -o bin/game-api ./cmd/game-api
            go build -ldflags="-s -w" -o bin/game ./cmd/game
            Write-Host "Built server/bin/game-api, server/bin/game (linux/amd64)"
        } finally {
            Pop-Location
        }
    }
    'docker-build' {
        Push-Location $Server
        try {
            docker build -f deploy/Dockerfile --target game-api -t game-api:latest .
            docker build -f deploy/Dockerfile --target game -t game-server:latest .
            Write-Host "Built game-api:latest, game-server:latest (linux/amd64)"
        } finally {
            Pop-Location
        }
    }
    'serve-bundles' {
        $bundleRoot = Join-Path $Client 'build\bundles'
        New-Item -ItemType Directory -Force -Path $bundleRoot | Out-Null
        & (Join-Path $PSScriptRoot 'serve-bundles.ps1') -Root $bundleRoot
    }
}
