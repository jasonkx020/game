@echo off
chcp 65001 >nul
title Redis Status (YCWL)

set "SCRIPT_DIR=%~dp0"
set "SCRIPT_DIR=%SCRIPT_DIR:~0,-1%"

for /f "usebackq delims=" %%a in (`powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%\read-redis-config.ps1" -EmitBat`) do %%a
if not defined PORT (
    echo Failed to load configuration
    pause
    exit /b 1
)

echo ========================================
powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%\read-redis-config.ps1" -PrintSummary
echo ========================================

netstat -ano | findstr ":%PORT%" | findstr "LISTENING" >nul
if %errorLevel% equ 0 (
    echo Port %PORT%: listening
    for /f "tokens=5" %%a in ('netstat -ano ^| findstr ":%PORT%" ^| findstr "LISTENING"') do (
        echo PID: %%a
        goto :after_pid
    )
) else (
    echo Port %PORT%: not listening
)
:after_pid

if exist "%REDIS_SERVER_EXE%" (
    echo Server: %REDIS_SERVER_EXE%
) else (
    echo Server: not found %REDIS_SERVER_EXE%
)

if exist "%CONF_OUTPUT%" (
    echo Runtime config: %CONF_OUTPUT%
) else (
    echo Runtime config: not rendered ^(run start-redis.bat^)
)

set "CLI_ARGS=-h %LISTEN_BIND% -p %PORT%"
if not "%REDIS_PASSWORD%"=="" set "CLI_ARGS=%CLI_ARGS% -a %REDIS_PASSWORD%"
if exist "%REDIS_CLI_EXE%" (
    "%REDIS_CLI_EXE%" %CLI_ARGS% ping 2>nul | findstr /i "PONG" >nul
    if %errorLevel% equ 0 (
        echo Health: PONG
    ) else (
        echo Health: unreachable
    )
)

if exist "%LOG_FILE%" (
    echo Log: %LOG_FILE%
)

pause
