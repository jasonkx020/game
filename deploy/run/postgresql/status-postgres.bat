@echo off
title PostgreSQL Status (YCWL)

set "SCRIPT_DIR=%~dp0"
set "SCRIPT_DIR=%SCRIPT_DIR:~0,-1%"

for /f "usebackq delims=" %%a in (`powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%\read-postgres-config.ps1" -EmitBat`) do %%a
if not defined PG_PORT (
    echo Failed to load configuration
    pause
    exit /b 1
)

echo ========================================
powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%\read-postgres-config.ps1" -PrintSummary
echo ========================================

netstat -ano | findstr ":%PG_PORT%" | findstr "LISTENING" >nul
if %errorLevel% equ 0 (
    echo Port %PG_PORT%: listening
) else (
    echo Port %PG_PORT%: not listening
)

if /I "%RUN_MODE%"=="service" (
    sc query "%WINDOWS_SERVICE_NAME%" 2>nul | findstr /i "STATE"
)

if exist "%PG_ISREADY_EXE%" (
    "%PG_ISREADY_EXE%" -h %POSTGRES_HOST% -p %PG_PORT% -U %POSTGRES_SUPERUSER%
    if %errorLevel% equ 0 (
        echo pg_isready: accepting connections
    ) else (
        echo pg_isready: not ready
    )
) else (
    echo pg_isready.exe not found
)

if exist "%DATA_PATH%\PG_VERSION%" (
    echo Cluster data: %DATA_PATH% ^(PG_VERSION present^)
) else if /I "%RUN_MODE%"=="pg_ctl" (
    echo Cluster data: not initialized ^(run start-postgres.bat^)
)

if exist "%PSQL_EXE%" (
    echo Client: %PSQL_EXE%
) else (
    echo Client: psql not found ^(add PostgreSQL bin to PATH or install_path^)
)

pause
