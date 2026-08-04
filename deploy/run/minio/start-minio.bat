@echo off
title MinIO Server (YCWL)

set "SCRIPT_DIR=%~dp0"
set "SCRIPT_DIR=%SCRIPT_DIR:~0,-1%"

net session >nul 2>&1
if %errorLevel% neq 0 (
    echo Requesting administrator privileges...
    powershell start -FilePath """%~f0""" -verb runas >nul 2>&1
    exit /b
)

echo Loading config.ini ...
for /f "usebackq delims=" %%a in (`powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%\read-minio-config.ps1" -EmitBat`) do %%a
if not defined INSTALL_PATH (
    echo Failed to load configuration
    pause
    exit /b 1
)

if not exist "%INSTALL_PATH%" mkdir "%INSTALL_PATH%"
if not exist "%DATA_PATH%" mkdir "%DATA_PATH%"

if /I "%ADD_FIREWALL%"=="true" (
    netsh advfirewall firewall show rule name="MinIO API Port" >nul 2>&1
    if %errorLevel% neq 0 (
        netsh advfirewall firewall add rule name="MinIO API Port" dir=in action=allow protocol=tcp localport=%API_PORT%
    )
    netsh advfirewall firewall show rule name="MinIO Console Port" >nul 2>&1
    if %errorLevel% neq 0 (
        netsh advfirewall firewall add rule name="MinIO Console Port" dir=in action=allow protocol=TCP localport=%CONSOLE_PORT%
    )
)

if not exist "%MINIO_EXE%" (
    echo minio.exe not found, downloading to %INSTALL_PATH% ...
    set "DOWNLOAD_URL=https://dl.min.io/server/minio/release/windows-amd64/minio.exe"
    powershell -NoProfile -Command "& {[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12; Invoke-WebRequest -Uri '%DOWNLOAD_URL%' -OutFile '%MINIO_EXE%'}"
    if not exist "%MINIO_EXE%" (
        echo Download failed, check network connection
        pause
        exit /b 1
    )
)

set "MINIO_ROOT_USER=%MINIO_ROOT_USER%"
set "MINIO_ROOT_PASSWORD=%MINIO_ROOT_PASSWORD%"
if not "%BROWSER_REDIRECT%"=="" set "MINIO_BROWSER_REDIRECT_URL=%BROWSER_REDIRECT%"

if /I "%MINIO_ROOT_USER%"=="minioadmin" if /I "%MINIO_ROOT_PASSWORD%"=="minioadmin" (
    echo WARN: Default credentials minioadmin:minioadmin detected.
    echo       Edit [Credentials] in config.ini and restart MinIO.
    echo       Existing data dir may require the original credentials or a fresh data_path.
)

cd /d "%INSTALL_PATH%"

echo.
echo ========================================
echo MinIO startup
echo Install:   %INSTALL_PATH%
echo Data:      %DATA_PATH%
echo API:       http://%LISTEN_BIND%:%API_PORT%
echo Console:   http://%LISTEN_BIND%:%CONSOLE_PORT%
echo User:      %MINIO_ROOT_USER%
echo Bucket:    %BUCKET%
echo ========================================
echo Keep this window open; closing it stops MinIO
echo.

"%MINIO_EXE%" server "%DATA_PATH%" --console-address "%LISTEN_BIND%:%CONSOLE_PORT%" --address "%LISTEN_BIND%:%API_PORT%"

pause
