@echo off
title NanoMQ Stopper (YCWL)

set "SCRIPT_DIR=%~dp0"
set "SCRIPT_DIR=%SCRIPT_DIR:~0,-1%"

echo Loading config.ini ...
for /f "usebackq delims=" %%a in (`powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%\read-nanomq-config.ps1" -EmitBat`) do %%a
if not defined NANOMQ_BIN (
    echo Failed to load configuration
    pause
    exit /b 1
)

if not exist "%NANOMQ_BIN%" (
    echo Error: NanoMQ executable not found: %NANOMQ_BIN%
    pause
    exit /b 1
)

echo Stopping NanoMQ ...
"%NANOMQ_BIN%" stop

if %errorLevel% equ 0 (
    echo NanoMQ has been stopped.
) else (
    echo No running NanoMQ process found.
)
pause
