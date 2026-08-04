@echo off
setlocal enabledelayedexpansion
chcp 65001 >nul
title Redis Client (YCWL redis-cli)

set "SCRIPT_DIR=%~dp0"
set "SCRIPT_DIR=%SCRIPT_DIR:~0,-1%"

for /f "usebackq delims=" %%a in (`powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%\read-redis-config.ps1" -EmitBat`) do %%a
if not defined REDIS_CLI_EXE (
    echo Failed to load configuration
    pause
    exit /b 1
)

if not exist "%REDIS_CLI_EXE%" (
    echo redis-cli.exe not found: %REDIS_CLI_EXE%
    pause
    exit /b 1
)

set "CLI_ARGS=-h %LISTEN_BIND% -p %PORT%"
if not "%REDIS_PASSWORD%"=="" set "CLI_ARGS=%CLI_ARGS% -a %REDIS_PASSWORD%"

if not "%1"=="" (
    "%REDIS_CLI_EXE%" %CLI_ARGS% %*
    exit /b %errorLevel%
)

:menu
cls
echo ========================================
echo Redis client (redis-cli)
echo Addr: %REDIS_ADDR%
echo Alias: %REDIS_ALIAS%
echo ========================================
echo  1. PING
echo  2. INFO server
echo  3. Interactive shell
echo  4. DBSIZE
echo  5. MONITOR ^(Ctrl+C to stop^)
echo  0. Exit
echo ========================================
set /p choice=Select [0-5]:

if "%choice%"=="1" (
    "%REDIS_CLI_EXE%" %CLI_ARGS% ping
    pause & goto :menu
)
if "%choice%"=="2" (
    "%REDIS_CLI_EXE%" %CLI_ARGS% info server
    pause & goto :menu
)
if "%choice%"=="3" (
    "%REDIS_CLI_EXE%" %CLI_ARGS%
    pause & goto :menu
)
if "%choice%"=="4" (
    "%REDIS_CLI_EXE%" %CLI_ARGS% dbsize
    pause & goto :menu
)
if "%choice%"=="5" (
    "%REDIS_CLI_EXE%" %CLI_ARGS% monitor
    pause & goto :menu
)
if "%choice%"=="0" exit /b
goto :menu
