@echo off
chcp 65001 >nul
title Redis Stopper (YCWL)

set "SCRIPT_DIR=%~dp0"
set "SCRIPT_DIR=%SCRIPT_DIR:~0,-1%"
set "NO_PAUSE="

if /I "%1"=="--no-pause" set "NO_PAUSE=1"

echo Loading config.ini ...
for /f "usebackq delims=" %%a in (`powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%\read-redis-config.ps1" -EmitBat`) do %%a
if not defined PORT (
    echo Failed to load configuration
    if not defined NO_PAUSE pause
    exit /b 1
)

if not exist "%REDIS_CLI_EXE%" (
    echo redis-cli.exe not found: %REDIS_CLI_EXE%
    if not defined NO_PAUSE pause
    exit /b 1
)

set "CLI_ARGS=-h %LISTEN_BIND% -p %PORT%"
if not "%REDIS_PASSWORD%"=="" set "CLI_ARGS=%CLI_ARGS% -a %REDIS_PASSWORD%"

netstat -ano | findstr ":%PORT%" | findstr "LISTENING" >nul
if %errorLevel% equ 0 (
    echo Gracefully stopping Redis on %REDIS_ADDR% ...
    "%REDIS_CLI_EXE%" %CLI_ARGS% shutdown save >nul 2>&1
    timeout /t 3 /nobreak >nul
)

set "PID="
for /f "tokens=5" %%a in ('netstat -ano ^| findstr ":%PORT%" ^| findstr "LISTENING"') do (
    set "PID=%%a"
    goto :found
)

echo Redis not listening on port %PORT%.
if not defined NO_PAUSE pause
exit /b 0

:found
tasklist /fi "PID eq %PID%" | findstr /i "redis-server.exe" >nul
if errorlevel 1 (
    echo Warning: PID %PID% is not redis-server.exe, aborted
    if not defined NO_PAUSE pause
    exit /b 1
)

echo Force stopping redis-server.exe (PID %PID%) ...
taskkill /PID %PID% /F >nul 2>&1
if %errorLevel% equ 0 (
    echo Redis stopped.
) else (
    echo Stop failed, try running as administrator.
)
if not defined NO_PAUSE pause
