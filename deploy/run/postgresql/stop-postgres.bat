@echo off
title PostgreSQL Stop (YCWL)

set "SCRIPT_DIR=%~dp0"
set "SCRIPT_DIR=%SCRIPT_DIR:~0,-1%"

echo Loading config.ini ...
for /f "usebackq delims=" %%a in (`powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%\read-postgres-config.ps1" -EmitBat`) do %%a
if not defined PG_PORT (
    echo Failed to load configuration
    pause
    exit /b 1
)

if /I "%RUN_MODE%"=="service" goto :stop_service
if /I "%RUN_MODE%"=="pg_ctl" goto :stop_pgctl
echo Unknown run_mode: %RUN_MODE%
pause
exit /b 1

:stop_service
echo Stopping Windows service: %WINDOWS_SERVICE_NAME%
net stop "%WINDOWS_SERVICE_NAME%" 2>nul
if %errorLevel% equ 0 (
    echo PostgreSQL service stopped
) else (
    sc query "%WINDOWS_SERVICE_NAME%" | findstr /i "STOPPED" >nul
    if %errorLevel% equ 0 (
        echo Service already stopped
    ) else (
        echo Stop failed; run as administrator or check service name
    )
)
pause
exit /b

:stop_pgctl
if not exist "%PG_CTL_EXE%" (
    echo pg_ctl.exe not found: %PG_CTL_EXE%
    pause
    exit /b 1
)
if not exist "%DATA_PATH%\PG_VERSION%" (
    echo No cluster at %DATA_PATH%
    pause
    exit /b
)
echo Stopping cluster: %DATA_PATH%
"%PG_CTL_EXE%" stop -D "%DATA_PATH%" -m fast -w
if %errorLevel% equ 0 (
    echo PostgreSQL stopped
) else (
    echo pg_ctl stop failed or server not running
)
pause
