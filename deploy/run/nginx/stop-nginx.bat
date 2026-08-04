@echo off
chcp 65001 >nul
title Nginx Stopper

set "NGINX_ROOT=%~dp0"
set "NGINX_ROOT=%NGINX_ROOT:~0,-1%"

net session >nul 2>&1
if %errorLevel% neq 0 (
    echo Requesting administrator privileges...
    powershell start -FilePath """%~f0""" -verb runas >nul 2>&1
    exit /b
)

cd /d "%NGINX_ROOT%"

if not exist "%NGINX_ROOT%\nginx.exe" (
    echo nginx.exe not found
    pause
    exit /b
)

if exist "%NGINX_ROOT%\logs\nginx.pid" (
    echo Gracefully stopping Nginx...
    "%NGINX_ROOT%\nginx.exe" -p "%NGINX_ROOT%" -s quit
    timeout /t 3 /nobreak >nul
)

tasklist /fi "imagename eq nginx.exe" 2>nul | find /i "nginx.exe" >nul
if %errorLevel% equ 0 (
    echo Force stopping remaining Nginx processes...
    taskkill /f /im nginx.exe >nul
)
echo Nginx stopped.
pause