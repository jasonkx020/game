@echo off
setlocal enabledelayedexpansion
title GameS PostgreSQL Init DB

set "SCRIPT_DIR=%~dp0"
set "SCRIPT_DIR=%SCRIPT_DIR:~0,-1%"

set "SCOPE="
set "EXTRA="

:parse_args
if "%~1"=="" goto :run
if /I "%~1"=="game" set "SCOPE=game" & shift & goto :parse_args
if /I "%~1"=="auth" set "SCOPE=auth" & shift & goto :parse_args
if /I "%~1"=="all" set "SCOPE=all" & shift & goto :parse_args
if /I "%~1"=="--split-auth" set "EXTRA=!EXTRA! -SplitAuth" & shift & goto :parse_args
if /I "%~1"=="-SplitAuth" set "EXTRA=!EXTRA! -SplitAuth" & shift & goto :parse_args
if /I "%~1"=="--write-env" set "EXTRA=!EXTRA! -WriteEnv" & shift & goto :parse_args
if /I "%~1"=="-WriteEnv" set "EXTRA=!EXTRA! -WriteEnv" & shift & goto :parse_args
if /I "%~1"=="--no-write-env" set "EXTRA=!EXTRA! -NoWriteEnv" & shift & goto :parse_args
if /I "%~1"=="-NoWriteEnv" set "EXTRA=!EXTRA! -NoWriteEnv" & shift & goto :parse_args
if /I "%~1"=="--app-password" (
    set "EXTRA=!EXTRA! -AppPassword ""%~2"""
    shift & shift & goto :parse_args
)
echo Unknown argument: %~1
echo Usage: init-db.bat [game^|auth^|all] [--split-auth] [--write-env^|--no-write-env]
pause
exit /b 1

:run
call "%SCRIPT_DIR%\load-postgres-config.cmd"
if errorlevel 1 (
    pause
    exit /b 1
)

if not defined SCOPE if defined DEFAULT_SCOPE set "SCOPE=%DEFAULT_SCOPE%"
if "%SCOPE%"=="" set "SCOPE=game"

echo Running db-init: scope=%SCOPE%
powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%\db-init.ps1" -ConfigIni "%CONFIG_FILE%" -Scope %SCOPE% %EXTRA%
if errorlevel 1 (
    echo db-init failed
    pause
    exit /b 1
)

echo.
echo Database init completed.
echo Next: from repo root run .\scripts\dev.ps1 seed-dev then .\scripts\dev.ps1 run-auth
pause
