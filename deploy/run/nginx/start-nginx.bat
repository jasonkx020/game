@echo off
chcp 65001 >nul
title Nginx Starter (MinIO Proxy)

set "NGINX_ROOT=%~dp0"
set "NGINX_ROOT=%NGINX_ROOT:~0,-1%"

net session >nul 2>&1
if %errorLevel% neq 0 (
    echo Requesting administrator privileges...
    powershell start -FilePath """%~f0""" -verb runas >nul 2>&1
    exit /b
)

cd /d "%NGINX_ROOT%"

if not exist "%NGINX_ROOT%\logs" mkdir "%NGINX_ROOT%\logs"
if not exist "%NGINX_ROOT%\conf\ssl" mkdir "%NGINX_ROOT%\conf\ssl"
if not exist "%NGINX_ROOT%\temp\client_body_temp" mkdir "%NGINX_ROOT%\temp\client_body_temp"
if not exist "%NGINX_ROOT%\temp\proxy_temp" mkdir "%NGINX_ROOT%\temp\proxy_temp"
if not exist "%NGINX_ROOT%\temp\fastcgi_temp" mkdir "%NGINX_ROOT%\temp\fastcgi_temp"
if not exist "%NGINX_ROOT%\temp\uwsgi_temp" mkdir "%NGINX_ROOT%\temp\uwsgi_temp"
if not exist "%NGINX_ROOT%\temp\scgi_temp" mkdir "%NGINX_ROOT%\temp\scgi_temp"

if not exist "%NGINX_ROOT%\nginx.exe" (
    echo Error: nginx.exe not found in %NGINX_ROOT%
    pause
    exit /b
)

if not exist "%NGINX_ROOT%\conf\nginx.conf" (
    echo Error: conf\nginx.conf not found
    pause
    exit /b
)

if not exist "%NGINX_ROOT%\conf\ssl\dev-server.crt" (
    echo Warning: TLS cert not found. Place dev-server.crt/key under conf\ssl\
)

netsh advfirewall firewall show rule name="Nginx HTTPS Port" >nul 2>&1
if %errorLevel% neq 0 (
    echo Opening firewall ports 80,8081,8084,8883,9002,9003...
    netsh advfirewall firewall add rule name="YCWL Nginx HTTP" dir=in action=allow protocol=tcp localport=80
    netsh advfirewall firewall add rule name="YCWL Nginx HTTPS" dir=in action=allow protocol=tcp localport=8081,8084,8883,9002,9003
)

echo Stopping any existing Nginx instance...
"%NGINX_ROOT%\nginx.exe" -s stop -p "%NGINX_ROOT%" -c "conf/nginx.conf" >nul 2>&1
timeout /t 1 /nobreak >nul

echo Starting Nginx (root: %NGINX_ROOT%)...
start "" "%NGINX_ROOT%\nginx.exe" -p "%NGINX_ROOT%" -c "conf/nginx.conf"

timeout /t 2 /nobreak >nul
tasklist /fi "imagename eq nginx.exe" 2>nul | find /i "nginx.exe" >nul
if %errorLevel% equ 0 (
    echo Nginx started successfully.
    echo Tenant:        https://127.0.0.1:8081/
    echo MinIO API:     https://127.0.0.1:9002/
    echo MinIO Console: https://127.0.0.1:9003/
    echo Platform admin: https://127.0.0.1:8080/
    echo Logs: %NGINX_ROOT%\logs
) else (
    echo Nginx failed to start. Check configuration.
    echo Error log: %NGINX_ROOT%\logs\error.log
)
pause