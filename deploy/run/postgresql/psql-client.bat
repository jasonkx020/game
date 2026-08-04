@echo off
setlocal enabledelayedexpansion
title PostgreSQL Client (YCWL psql)

set "SCRIPT_DIR=%~dp0"
set "SCRIPT_DIR=%SCRIPT_DIR:~0,-1%"

for /f "usebackq delims=" %%a in (`powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%\read-postgres-config.ps1" -EmitBat`) do %%a
if not defined PSQL_EXE (
    echo Failed to load configuration
    pause
    exit /b 1
)

if not exist "%PSQL_EXE%" (
    echo psql not found: %PSQL_EXE%
    pause
    exit /b 1
)

if not "%1"=="" (
    set "PGPASSWORD=%POSTGRES_PASSWORD%"
    "%PSQL_EXE%" -h %POSTGRES_HOST% -p %PG_PORT% -U %POSTGRES_USER% -d %POSTGRES_DB_EDGE% %*
    exit /b %errorLevel%
)

:menu
cls
echo ========================================
echo PostgreSQL client (psql)
echo Host: %POSTGRES_HOST%:%PG_PORT%
echo User: %POSTGRES_USER%
echo ========================================
echo  1. Connect EDGE  ^(%POSTGRES_DB_EDGE%^)
echo  2. Connect CLOUD ^(%POSTGRES_DB_CLOUD%^)
echo  3. Connect as superuser ^(%POSTGRES_SUPERUSER%^, postgres DB^)
echo  4. Show migration status ^(tenant^)
echo  0. Exit
echo ========================================
set /p choice=Select [0-4]:

if "%choice%"=="1" (
    set "PGPASSWORD=%POSTGRES_PASSWORD%"
    "%PSQL_EXE%" -h %POSTGRES_HOST% -p %PG_PORT% -U %POSTGRES_USER% -d %POSTGRES_DB_EDGE%
    pause & goto :menu
)
if "%choice%"=="2" (
    set "PGPASSWORD=%POSTGRES_PASSWORD%"
    "%PSQL_EXE%" -h %POSTGRES_HOST% -p %PG_PORT% -U %POSTGRES_USER% -d %POSTGRES_DB_CLOUD%
    pause & goto :menu
)
if "%choice%"=="3" (
    set "PGPASSWORD=%POSTGRES_SUPERUSER_PASSWORD%"
    "%PSQL_EXE%" -h %POSTGRES_HOST% -p %PG_PORT% -U %POSTGRES_SUPERUSER% -d postgres
    pause & goto :menu
)
if "%choice%"=="4" (
    if exist "%DB_MIGRATE_SCRIPT%" (
        powershell -NoProfile -ExecutionPolicy Bypass -File "%DB_MIGRATE_SCRIPT%" status tenant
    ) else (
        echo Missing %DB_MIGRATE_SCRIPT%
    )
    pause & goto :menu
)
if "%choice%"=="0" exit /b
goto :menu
