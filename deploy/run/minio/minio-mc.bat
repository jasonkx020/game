@echo off
setlocal enabledelayedexpansion
title MinIO Client (YCWL mc)

set "SCRIPT_DIR=%~dp0"
set "SCRIPT_DIR=%SCRIPT_DIR:~0,-1%"

echo Loading config.ini ...
for /f "usebackq delims=" %%a in (`powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%\read-minio-config.ps1" -EmitBat`) do %%a
if not defined MC_EXE (
    echo Failed to load configuration
    pause
    exit /b 1
)

if not exist "%MC_EXE%" (
    echo mc.exe not found, downloading...
    powershell -NoProfile -Command "& {[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12; Invoke-WebRequest -Uri 'https://dl.min.io/client/mc/release/windows-amd64/mc.exe' -OutFile '%MC_EXE%'}"
    if not exist "%MC_EXE%" (
        echo Download failed
        pause
        exit /b 1
    )
)

echo Setting alias %MINIO_ALIAS% -^> %MINIO_ENDPOINT%
"%MC_EXE%" alias set %MINIO_ALIAS% %MINIO_ENDPOINT% %MINIO_ROOT_USER% %MINIO_ROOT_PASSWORD%
if %errorLevel% neq 0 (
    echo Alias setup failed; ensure MinIO is running and credentials match config.ini
    pause
    exit /b 1
)

if not "%1"=="" (
    echo Running: mc %*
    "%MC_EXE%" %*
    exit /b %errorLevel%
)

:menu

echo ========================================
echo MinIO client (mc)
echo Alias:  %MINIO_ALIAS% -^> %MINIO_ENDPOINT%
echo Bucket: %BUCKET%
echo ========================================
echo  1. List buckets
echo  2. List bucket contents
echo  3. Create default bucket (%BUCKET%)
echo  4. Upload file
echo  5. Mirror directory
echo  0. Exit
echo ========================================
set /p choice=Select [0-5]:

if "%choice%"=="1" "%MC_EXE%" ls %MINIO_ALIAS%/ & pause & goto :menu
if "%choice%"=="2" (
    echo Bucket name:
    set /p bucket=
    "%MC_EXE%" ls %MINIO_ALIAS%/!bucket!/ & pause & goto :menu
)
if "%choice%"=="3" (
    "%MC_EXE%" mb %MINIO_ALIAS%/%BUCKET% 2>nul
    "%MC_EXE%" anonymous set none %MINIO_ALIAS%/%BUCKET%
    pause & goto :menu
)
if "%choice%"=="4" (
    echo Input Local file path:
    set /p localpath=
    echo Input Remote path:   e.g. freetv-assets/images/object.png
    set /p remotepath=
    "%MC_EXE%" cp "!localpath!" %MINIO_ALIAS%/!remotepath!
    pause & goto :menu
)
if "%choice%"=="5" (
    echo Local directory:
    set /p localdir=
    "%MC_EXE%" mirror "!localdir!" %MINIO_ALIAS%/%BUCKET%/
    pause & goto :menu
)
if "%choice%"=="0" exit /b
goto :menu
