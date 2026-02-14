@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

title WalletBot - Local Startup

echo ============================================
echo   WalletBot - Local Quick Start
echo ============================================
echo.

REM ---- Set script directory as base ----
set "BASE_DIR=%~dp0"
cd /d "%BASE_DIR%"

REM ---- Check prerequisites ----
where go >nul 2>&1 || (echo [ERROR] Go not found. Install Go 1.22+ && pause && exit /b 1)

REM ---- Check config file ----
if not exist "configs\config.yaml" (
    echo [WARN] configs\config.yaml not found, creating from template...
    copy "configs\config.yaml.example" "configs\config.yaml" >nul 2>&1
    echo [INFO] Created configs\config.yaml
    echo [TODO] Please edit configs\config.yaml:
    echo.
    echo   Required:
    echo     bot.token          - Get from @BotFather
    echo     database.password  - MySQL root password
    echo.
    notepad "configs\config.yaml"
    echo Press any key after editing...
    pause >nul
)

REM ---- Detect local services ----
echo.
echo [1/4] Detecting infrastructure ...

set "USE_DOCKER_MYSQL=1"
set "USE_DOCKER_REDIS=1"

REM Check local MySQL (port 3306)
netstat -ano 2>nul | findstr ":3306 " | findstr "LISTENING" >nul 2>&1
if not errorlevel 1 (
    echo [OK] Local MySQL detected on port 3306
    set "USE_DOCKER_MYSQL=0"
)

REM Check local Redis (port 6379)
netstat -ano 2>nul | findstr ":6379 " | findstr "LISTENING" >nul 2>&1
if not errorlevel 1 (
    echo [OK] Local Redis detected on port 6379
    set "USE_DOCKER_REDIS=0"
)

REM Start Docker services only if needed
if "!USE_DOCKER_MYSQL!!USE_DOCKER_REDIS!"=="00" (
    echo [OK] All services running locally, skip Docker
    goto services_ready
)

REM Need Docker for something
where docker >nul 2>&1 || (echo [ERROR] Docker not found but needed for services && pause && exit /b 1)
docker info >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Docker Engine is not running. Please start Docker Desktop.
    pause
    exit /b 1
)

set "DOCKER_SERVICES="
if "!USE_DOCKER_MYSQL!"=="1" set "DOCKER_SERVICES=mysql"
if "!USE_DOCKER_REDIS!"=="1" set "DOCKER_SERVICES=!DOCKER_SERVICES! redis"

echo       Starting Docker: !DOCKER_SERVICES!
pushd "%BASE_DIR%deployments"
docker compose up -d !DOCKER_SERVICES!
set "DC_ERR=!errorlevel!"
popd
if not "!DC_ERR!"=="0" (echo [ERROR] Docker startup failed && pause && exit /b 1)

REM Wait for Docker MySQL if needed
if "!USE_DOCKER_MYSQL!"=="1" (
    echo [2/4] Waiting for Docker MySQL ready ...
    set RETRIES=0
    :wait_mysql
    docker exec tgbot-mysql mysqladmin ping -h localhost --silent >nul 2>&1
    if not errorlevel 1 goto mysql_ready
    set /a RETRIES+=1
    if !RETRIES! GEQ 30 (
        echo [ERROR] MySQL startup timeout 30s
        pause
        exit /b 1
    )
    timeout /t 1 /nobreak >nul
    goto wait_mysql
    :mysql_ready
    echo [OK] Docker MySQL is ready
)

:services_ready

REM ---- Create database if not exists ----
echo [2/4] Checking database ...
if "!USE_DOCKER_MYSQL!"=="1" (
    docker exec tgbot-mysql mysql -uroot -proot123456 -e "CREATE DATABASE IF NOT EXISTS walletbot CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci" >nul 2>&1
) else (
    where mysql >nul 2>&1
    if not errorlevel 1 (
        mysql -uroot -e "CREATE DATABASE IF NOT EXISTS walletbot CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci" 2>nul
        if errorlevel 1 (
            echo [INFO] Auto-create DB failed. Make sure database 'walletbot' exists.
            echo       Run: CREATE DATABASE walletbot CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
        )
    ) else (
        echo [INFO] mysql CLI not in PATH, cannot auto-create database.
        echo       Make sure database 'walletbot' exists in your local MySQL.
    )
)

REM ---- Database Migrations ----
echo [3/4] Running database migrations ...
go run "%BASE_DIR%cmd\migrate\main.go" -config "%BASE_DIR%configs\config.yaml" up
if errorlevel 1 (
    echo [WARN] Migration may have issues, check output above
)

REM ---- Start API + Bot ----
echo.
echo [4/4] Starting services ...
echo ============================================
echo   API server: http://localhost:8080
echo   Bot is running
echo   Close both windows to stop all services
echo ============================================
echo.

REM 独立窗口启动 Admin API 服务 (关闭窗口即停止)
start "WalletBot-API" cmd /k "title WalletBot API Server && go run "%BASE_DIR%cmd\api\main.go" -config "%BASE_DIR%configs\config.yaml" || echo [ERROR] API exited abnormally && pause"
echo [OK] Admin API started in separate window (port 8080)

REM 等待 API 启动
timeout /t 2 /nobreak >nul

REM 当前窗口启动 Bot 服务 (关闭窗口即停止)
go run "%BASE_DIR%cmd\bot\main.go" -config "%BASE_DIR%configs\config.yaml"