@echo off
title NanoMQ Server (YCWL)

set "SCRIPT_DIR=%~dp0"
set "SCRIPT_DIR=%SCRIPT_DIR:~0,-1%"

net session >nul 2>&1
if %errorLevel% neq 0 (
    echo Requesting administrator privileges...
    powershell start -FilePath """%~f0""" -verb runas >nul 2>&1
    exit /b
)

echo Loading config.ini ...
for /f "usebackq delims=" %%a in (`powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%\read-nanomq-config.ps1" -EmitBat`) do %%a
if not defined MQTT_PORT (
    echo Failed to load configuration
    pause
    exit /b 1
)

if not exist "%LOG_DIR%" mkdir "%LOG_DIR%"
if not exist "%NANOMQ_BIN%" (
    echo Error: NanoMQ executable not found: %NANOMQ_BIN%
    pause
    exit /b 1
)

echo Rendering nanomq.conf ...
powershell -NoProfile -ExecutionPolicy Bypass -File "%RENDER_SCRIPT%" -ConfigIni "%CONFIG_FILE%"
if %errorLevel% neq 0 (
    echo Render failed
    pause
    exit /b 1
)

if /I "%CHECK_HEALTH%"=="true" (
    echo Checking tenant-api: %HEALTH_URL%
    powershell -NoProfile -Command "try { $r=Invoke-WebRequest -UseBasicParsing -TimeoutSec 5 '%HEALTH_URL%'; if ($r.StatusCode -ne 200) { exit 1 } } catch { Write-Host 'tenant-api unreachable, continuing NanoMQ start'; exit 0 }"
)

if /I "%ADD_FIREWALL%"=="true" (
    netsh advfirewall firewall show rule name="NanoMQ MQTT Port" >nul 2>&1
    if %errorLevel% neq 0 (
        netsh advfirewall firewall add rule name="NanoMQ MQTT Port" dir=in action=allow protocol=tcp localport=%MQTT_PORT%
    )
)

cd /d "%NANOMQ_PATH%"
echo Starting NanoMQ ...
"%NANOMQ_BIN%" start --conf "%NANOMQ_CONFIG%" --daemon

if %errorLevel% equ 0 (
    echo ========================================
    echo NanoMQ started
    echo MQTT:      tcp://127.0.0.1:%MQTT_PORT%
    echo HTTP admin http://127.0.0.1:%HTTP_PORT% ^(local only^)
    echo Auth HTTP: %TENANT_API_BASE%/api/v1/mqtt/*
    echo Set tenant-api YCWL_MQTT_BROKER_TYPE=nanomq
    echo ========================================
) else (
    echo Start failed, check %NANOMQ_CONFIG%
)
pause
