@echo off
chcp 65001 >nul
title Redis Config Reload (YCWL)

set "SCRIPT_DIR=%~dp0"
set "SCRIPT_DIR=%SCRIPT_DIR:~0,-1%"

net session >nul 2>&1
if %errorLevel% neq 0 (
    echo Requesting administrator privileges...
    powershell start -FilePath """%~f0""" -verb runas >nul 2>&1
    exit /b
)

echo Loading config.ini ...
for /f "usebackq delims=" %%a in (`powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%\read-redis-config.ps1" -EmitBat`) do %%a
if not defined PORT (
    echo Failed to load configuration
    pause
    exit /b 1
)

if not exist "%REDIS_SERVER_EXE%" (
    echo redis-server.exe not found
    pause
    exit /b 1
)

echo Rendering redis.conf ...
powershell -NoProfile -ExecutionPolicy Bypass -File "%RENDER_SCRIPT%" -ConfigIni "%CONFIG_FILE%"
if %errorLevel% neq 0 (
    echo Render failed.
    pause
    exit /b 1
)

netstat -ano | findstr ":%PORT%" | findstr "LISTENING" >nul
if %errorLevel% neq 0 (
    echo Redis is not running. Starting ...
    call "%SCRIPT_DIR%\start-redis.bat"
    exit /b
)

echo Redis is running. Restarting to apply configuration ...
call "%SCRIPT_DIR%\stop-redis.bat" --no-pause
timeout /t 2 /nobreak >nul

cd /d "%INSTALL_PATH%"
if /I "%RUN_MODE%"=="foreground" (
    echo Foreground mode: run start-redis.bat manually after reload.
    pause
    exit /b
)

powershell -NoProfile -Command "Start-Process -FilePath '%REDIS_SERVER_EXE%' -ArgumentList '%CONF_OUTPUT%' -WorkingDirectory '%INSTALL_PATH%' -WindowStyle Hidden"
ping -n 3 127.0.0.1 >nul

set "CLI_ARGS=-h %LISTEN_BIND% -p %PORT%"
if not "%REDIS_PASSWORD%"=="" set "CLI_ARGS=%CLI_ARGS% -a %REDIS_PASSWORD%"
"%REDIS_CLI_EXE%" %CLI_ARGS% ping 2>nul | findstr /i "PONG" >nul
if %errorLevel% equ 0 (
    echo Reload successful.
) else (
    echo Reload failed. Check %LOG_FILE%
)
pause
