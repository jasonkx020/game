@echo off
title PostgreSQL Server (YCWL)

set "SCRIPT_DIR=%~dp0"
set "SCRIPT_DIR=%SCRIPT_DIR:~0,-1%"

net session >nul 2>&1
if %errorLevel% neq 0 (
    echo Requesting administrator privileges...
    powershell start -FilePath """%~f0""" -verb runas >nul 2>&1
    exit /b
)

echo Loading config.ini ...
for /f "usebackq delims=" %%a in (`powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%\read-postgres-config.ps1" -EmitBat`) do %%a
if not defined PG_PORT (
    echo Failed to load configuration
    pause
    exit /b 1
)

if not exist "%LOG_DIR%" mkdir "%LOG_DIR%"

if /I "%RUN_MODE%"=="service" goto :start_service
if /I "%RUN_MODE%"=="pg_ctl" goto :start_pgctl
echo Unknown run_mode: %RUN_MODE% ^(use service or pg_ctl^)
pause
exit /b 1

:start_service
echo Starting Windows service: %WINDOWS_SERVICE_NAME%
net start "%WINDOWS_SERVICE_NAME%" 2>nul
if %errorLevel% neq 0 (
    sc query "%WINDOWS_SERVICE_NAME%" | findstr /i "RUNNING" >nul
    if %errorLevel% neq 0 (
        echo Service start failed. Install PostgreSQL %PG_VERSION% or set windows_service_name in config.ini
        pause
        exit /b 1
    )
    echo Service already running
)
goto :post_start

:start_pgctl
if not exist "%PG_CTL_EXE%" (
    echo pg_ctl.exe not found: %PG_CTL_EXE%
    echo Install PostgreSQL %PG_VERSION% and set install_path, or place binaries under bin\
    pause
    exit /b 1
)

if not exist "%DATA_PATH%\PG_VERSION%" (
    if not exist "%INITDB_EXE%" (
        echo initdb.exe not found: %INITDB_EXE%
        pause
        exit /b 1
    )
    echo Initializing cluster: %DATA_PATH%
    "%INITDB_EXE%" -D "%DATA_PATH%" -U %POSTGRES_SUPERUSER% --encoding=UTF8 --locale=C
    if %errorLevel% neq 0 (
        echo initdb failed
        pause
        exit /b 1
    )
    powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%\render-postgresql-conf.ps1"
)

powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%\render-postgresql-conf.ps1"

"%PG_CTL_EXE%" status -D "%DATA_PATH%" >nul 2>&1
if %errorLevel% equ 0 (
    echo Cluster already running: %DATA_PATH%
    echo Reloading configuration ...
    "%PG_CTL_EXE%" reload -D "%DATA_PATH%" -s
    goto :post_start
)

echo Starting PostgreSQL ^(pg_ctl^) ...
"%PG_CTL_EXE%" start -D "%DATA_PATH%" -l "%LOG_DIR%\postgresql.log" -w
if %errorLevel% neq 0 (
    echo pg_ctl start failed, check %LOG_DIR%\postgresql.log
    pause
    exit /b 1
)
goto :post_start

:post_start
if /I "%ADD_FIREWALL%"=="true" (
    netsh advfirewall firewall show rule name="PostgreSQL Port" >nul 2>&1
    if %errorLevel% neq 0 (
        netsh advfirewall firewall add rule name="PostgreSQL Port" dir=in action=allow protocol=tcp localport=%PG_PORT%
    )
)

echo Waiting for PostgreSQL on %POSTGRES_HOST%:%PG_PORT% ...
powershell -NoProfile -Command ^
  "$deadline=(Get-Date).AddSeconds(30); $ok=$false; while((Get-Date) -lt $deadline){ if(& '%PG_ISREADY_EXE%' -h '%POSTGRES_HOST%' -p %PG_PORT% -U '%POSTGRES_SUPERUSER%' 2>$null){ $ok=$true; break }; Start-Sleep -Seconds 1 }; if(-not $ok){ exit 1 }"
if %errorLevel% neq 0 (
    echo pg_isready timeout; server may still be starting
)

echo.
echo ========================================
echo PostgreSQL started
echo Mode:     %RUN_MODE%
echo Host:     %POSTGRES_HOST%:%PG_PORT%
echo App user: %POSTGRES_USER%
echo EDGE DB:  %POSTGRES_DB_EDGE%
echo CLOUD DB: %POSTGRES_DB_CLOUD%
echo Next:     init-db.bat tenant --schema
echo tenant-api DSN:
echo   %TENANT_DSN%
echo ========================================
pause
