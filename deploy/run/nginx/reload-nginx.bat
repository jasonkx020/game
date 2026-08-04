@echo off
chcp 65001 >nul
title Nginx Config Reload

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

echo Testing configuration...
"%NGINX_ROOT%\nginx.exe" -p "%NGINX_ROOT%" -c conf/nginx.conf -t
if %errorLevel% neq 0 (
    echo Configuration test failed.
    pause
    exit /b
)

echo Reloading Nginx...
"%NGINX_ROOT%\nginx.exe" -p "%NGINX_ROOT%" -c conf/nginx.conf -s reload
if %errorLevel% equ 0 (
    echo Reload successful.
) else (
    echo Reload failed. Make sure Nginx is running.
)
pause