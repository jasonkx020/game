@echo off
setlocal enabledelayedexpansion
title GameS PostgreSQL Client

set "SCRIPT_DIR=%~dp0"
set "SCRIPT_DIR=%SCRIPT_DIR:~0,-1%"

call "%SCRIPT_DIR%\load-postgres-config.cmd"
if errorlevel 1 (
    pause
    exit /b 1
)
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
    "%PSQL_EXE%" -h %POSTGRES_HOST% -p %PG_PORT% -U %POSTGRES_USER% -d %POSTGRES_DB% %*
    exit /b %errorLevel%
)

:menu
cls
echo ========================================
echo PostgreSQL client (psql)
echo Host: %POSTGRES_HOST%:%PG_PORT%
echo User: %POSTGRES_USER%
echo ========================================
echo  1. Connect game DB ^(%POSTGRES_DB%^)
if defined POSTGRES_DB_AUTH if not "%POSTGRES_DB_AUTH%"=="" (
    echo  2. Connect auth DB ^(%POSTGRES_DB_AUTH%^)
) else (
    echo  2. Connect auth DB ^(not configured^)
)
echo  3. Connect as superuser ^(%POSTGRES_SUPERUSER%^, postgres DB^)
echo  0. Exit
echo ========================================
set /p choice=Select [0-3]:

if "%choice%"=="1" (
    set "PGPASSWORD=%POSTGRES_PASSWORD%"
    "%PSQL_EXE%" -h %POSTGRES_HOST% -p %PG_PORT% -U %POSTGRES_USER% -d %POSTGRES_DB%
    pause & goto :menu
)
if "%choice%"=="2" (
    if "%POSTGRES_DB_AUTH%"=="" (
        echo Application.db_auth is empty in config.ini
        pause & goto :menu
    )
    set "PGPASSWORD=%POSTGRES_PASSWORD%"
    "%PSQL_EXE%" -h %POSTGRES_HOST% -p %PG_PORT% -U %POSTGRES_USER% -d %POSTGRES_DB_AUTH%
    pause & goto :menu
)
if "%choice%"=="3" (
    if not "%POSTGRES_SUPERUSER_PASSWORD%"=="" set "PGPASSWORD=%POSTGRES_SUPERUSER_PASSWORD%"
    if "%POSTGRES_SUPERUSER_PASSWORD%"=="" set "PGPASSWORD="
    "%PSQL_EXE%" -h %POSTGRES_HOST% -p %PG_PORT% -U %POSTGRES_SUPERUSER% -d postgres
    pause & goto :menu
)
if "%choice%"=="0" exit /b
goto :menu
