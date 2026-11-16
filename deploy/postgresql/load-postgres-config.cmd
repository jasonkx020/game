@echo off
rem Load config.ini into current cmd environment via read-postgres-config.ps1 -WriteBatFile
set "SCRIPT_DIR=%~dp0"
set "SCRIPT_DIR=%SCRIPT_DIR:~0,-1%"
set "_PG_CFG_TMP=%TEMP%\games-pg-cfg-%RANDOM%%RANDOM%.cmd"
powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%\read-postgres-config.ps1" -WriteBatFile "%_PG_CFG_TMP%"
if errorlevel 1 (
    echo Failed to read config.ini
    if exist "%_PG_CFG_TMP%" del /f /q "%_PG_CFG_TMP%" >nul 2>&1
    exit /b 1
)
if not exist "%_PG_CFG_TMP%" (
    echo Failed to write temp config bat
    exit /b 1
)
call "%_PG_CFG_TMP%"
del /f /q "%_PG_CFG_TMP%" >nul 2>&1
if not defined REPO_ROOT (
    echo Failed to load PostgreSQL config
    exit /b 1
)
exit /b 0
