@echo off
setlocal enabledelayedexpansion
title PostgreSQL Init DB (YCWL)

set "SCRIPT_DIR=%~dp0"
set "SCRIPT_DIR=%SCRIPT_DIR:~0,-1%"

set "SCOPE="
set "WITH_SCHEMA=0"

:parse_args
if "%~1"=="" goto :run
if /I "%~1"=="tenant" set "SCOPE=tenant" & shift & goto :parse_args
if /I "%~1"=="platform" set "SCOPE=platform" & shift & goto :parse_args
if /I "%~1"=="all" set "SCOPE=all" & shift & goto :parse_args
if /I "%~1"=="--schema" set "WITH_SCHEMA=1" & shift & goto :parse_args
if /I "%~1"=="-Schema" set "WITH_SCHEMA=1" & shift & goto :parse_args
echo Unknown argument: %~1
echo Usage: init-db.bat [tenant^|platform^|all] [--schema]
pause
exit /b 1

:run
for /f "usebackq delims=" %%a in (`powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%\read-postgres-config.ps1" -EmitBat`) do %%a
if not defined DEFAULT_SCOPE (
    echo Failed to load configuration
    pause
    exit /b 1
)

if not defined SCOPE set "SCOPE=%DEFAULT_SCOPE%"

set "ENV_FILE=%SCRIPT_DIR%\postgres.env"
powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%\render-postgresql-conf.ps1"
if /I "%RUN_MODE%"=="pg_ctl" (
    if exist "%DATA_PATH%\PG_VERSION%" (
        if exist "%PG_CTL_EXE%" (
            "%PG_CTL_EXE%" status -D "%DATA_PATH%" >nul 2>&1
            if !errorLevel! equ 0 (
                echo Reloading pg_hba.conf ...
                "%PG_CTL_EXE%" reload -D "%DATA_PATH%" -s
            )
        )
    )
)
powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%\read-postgres-config.ps1" -WriteEnvFile "%ENV_FILE%"
if %errorLevel% neq 0 (
    echo Failed to write postgres.env
    pause
    exit /b 1
)

if not exist "%DB_INIT_SCRIPT%" (
    echo Missing: %DB_INIT_SCRIPT%
    pause
    exit /b 1
)

if "%PSQL_EXE%"=="" (
    echo psql not found under install_path in config.ini
    echo Set install_path = D:\path\to\pgsql  ^(must contain bin\psql.exe^)
    pause
    exit /b 1
)
if not exist "%PSQL_EXE%" (
    echo psql not found: %PSQL_EXE%
    pause
    exit /b 1
)

echo Running db-init: scope=%SCOPE% schema=%WITH_SCHEMA%
if "%WITH_SCHEMA%"=="1" (
    powershell -NoProfile -ExecutionPolicy Bypass -File "%DB_INIT_SCRIPT%" -Scope %SCOPE% -ConfigFile "%ENV_FILE%" -Schema
) else (
    powershell -NoProfile -ExecutionPolicy Bypass -File "%DB_INIT_SCRIPT%" -Scope %SCOPE% -ConfigFile "%ENV_FILE%"
)
if %errorLevel% neq 0 (
    echo db-init failed
    pause
    exit /b 1
)

echo.
echo Database init completed.
echo Verify: powershell -File "%DB_MIGRATE_SCRIPT%" status tenant
pause
