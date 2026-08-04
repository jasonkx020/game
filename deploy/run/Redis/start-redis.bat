@echo off
chcp 65001 >nul
title Redis Server (YCWL)

set "SCRIPT_DIR=%~dp0"
set "SCRIPT_DIR=%SCRIPT_DIR:~0,-1%"

echo Loading config.ini ...
for /f "usebackq delims=" %%a in (`powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%\read-redis-config.ps1" -EmitBat`) do %%a
if not defined PORT (
    echo Failed to load configuration
    pause
    exit /b 1
)

if not exist "%DATA_PATH%" mkdir "%DATA_PATH%"
if not exist "%LOG_PATH%" mkdir "%LOG_PATH%"
if not exist "%SCRIPT_DIR%\conf" mkdir "%SCRIPT_DIR%\conf"

if not exist "%REDIS_SERVER_EXE%" (
    echo Error: redis-server.exe not found: %REDIS_SERVER_EXE%
    pause
    exit /b 1
)

echo Rendering redis.conf ...
powershell -NoProfile -ExecutionPolicy Bypass -File "%RENDER_SCRIPT%" -ConfigIni "%CONFIG_FILE%"
if %errorLevel% neq 0 (
    echo Render failed
    pause
    exit /b 1
)

if not exist "%CONF_OUTPUT%" (
    echo Error: rendered config not found: %CONF_OUTPUT%
    pause
    exit /b 1
)

if /I "%ADD_FIREWALL%"=="true" (
    netsh advfirewall firewall show rule name="YCWL Redis Port" >nul 2>&1
    if %errorLevel% neq 0 (
        net session >nul 2>&1
        if %errorLevel% neq 0 (
            echo Requesting administrator privileges for firewall rule...
            powershell start -FilePath """%~f0""" -verb runas >nul 2>&1
            exit /b
        )
        echo Opening firewall port %PORT% ...
        netsh advfirewall firewall add rule name="YCWL Redis Port" dir=in action=allow protocol=tcp localport=%PORT%
    )
)

netstat -ano | findstr ":%PORT%" | findstr "LISTENING" >nul
if %errorLevel% equ 0 (
    echo Redis already listening on port %PORT%.
    goto :verify
)

echo Stopping any stale redis-server on port %PORT% ...
call "%SCRIPT_DIR%\stop-redis.bat" --no-pause >nul 2>&1
timeout /t 1 /nobreak >nul

cd /d "%INSTALL_PATH%"
echo Starting Redis (root: %INSTALL_PATH%)...
if /I "%RUN_MODE%"=="foreground" (
    echo Keep this window open; closing it stops Redis
    echo.
    "%REDIS_SERVER_EXE%" "%CONF_OUTPUT%"
    pause
    exit /b
)

powershell -NoProfile -Command "Start-Process -FilePath '%REDIS_SERVER_EXE%' -ArgumentList '%CONF_OUTPUT%' -WorkingDirectory '%INSTALL_PATH%' -WindowStyle Hidden"
ping -n 3 127.0.0.1 >nul

:verify
set "PING_ARGS=-h %LISTEN_BIND% -p %PORT%"
if not "%REDIS_PASSWORD%"=="" set "PING_ARGS=%PING_ARGS% -a %REDIS_PASSWORD%"
"%REDIS_CLI_EXE%" %PING_ARGS% ping 2>nul | findstr /i "PONG" >nul
if %errorLevel% equ 0 (
    echo Redis started successfully.
    echo Addr:   %REDIS_ADDR%
    echo Data:   %DATA_PATH%
    echo Config: %CONF_OUTPUT%
    echo Logs:   %LOG_FILE%
) else (
    echo Redis failed to start. Check configuration and logs.
    echo Log file: %LOG_FILE%
)
pause
