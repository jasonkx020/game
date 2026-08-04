@echo off
title MinIO Stop (YCWL)

set "SCRIPT_DIR=%~dp0"
set "SCRIPT_DIR=%SCRIPT_DIR:~0,-1%"

echo Loading config.ini ...
for /f "usebackq delims=" %%a in (`powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%\read-minio-config.ps1" -EmitBat`) do %%a
if not defined API_PORT (
    echo Failed to load configuration
    pause
    exit /b 1
)

echo Looking for MinIO on API port %API_PORT% ...
set "PID="
for /f "tokens=5" %%a in ('netstat -ano ^| findstr ":%API_PORT%" ^| findstr "LISTENING"') do (
    set "PID=%%a"
    goto :found
)

echo Not found on API port %API_PORT%, trying console port %CONSOLE_PORT% ...
for /f "tokens=5" %%a in ('netstat -ano ^| findstr ":%CONSOLE_PORT%" ^| findstr "LISTENING"') do (
    set "PID=%%a"
    goto :found
)

echo No running MinIO (ports %API_PORT% / %CONSOLE_PORT% not listening)
pause
exit /b

:found
echo Found PID: %PID%
tasklist /fi "PID eq %PID%" | findstr /i "minio.exe" >nul
if errorlevel 1 (
    echo Warning: PID %PID% is not minio.exe, aborted
    pause
    exit /b 1
)

taskkill /PID %PID% /F >nul 2>&1
if %errorlevel% equ 0 (
    echo MinIO stopped
) else (
    echo Stop failed, try running as administrator
)
pause
