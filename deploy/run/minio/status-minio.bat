@echo off
title MinIO Status (YCWL)

set "SCRIPT_DIR=%~dp0"
set "SCRIPT_DIR=%SCRIPT_DIR:~0,-1%"

for /f "usebackq delims=" %%a in (`powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%\read-minio-config.ps1" -EmitBat`) do %%a

echo ========================================
powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%\read-minio-config.ps1" -PrintSummary
echo ========================================

netstat -ano | findstr ":%API_PORT%" | findstr "LISTENING" >nul
if %errorLevel% equ 0 (
    echo API port %API_PORT%: listening
) else (
    echo API port %API_PORT%: not listening
)

netstat -ano | findstr ":%CONSOLE_PORT%" | findstr "LISTENING" >nul
if %errorLevel% equ 0 (
    echo Console port %CONSOLE_PORT%: listening
) else (
    echo Console port %CONSOLE_PORT%: not listening
)

if exist "%MINIO_EXE%" (
    echo Executable: %MINIO_EXE%
) else (
    echo Executable: not found %MINIO_EXE%
)

pause
