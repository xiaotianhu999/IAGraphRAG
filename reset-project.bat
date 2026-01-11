@echo off
setlocal enabledelayedexpansion

:: aiplusall-kb Project Reset Script for Windows
:: This script will clean all debugging data and reset the project to initial state

echo.
echo ========================================
echo aiplusall-kb Project Reset
echo ========================================
echo.
echo This will delete ALL data and reset the project to initial state.
echo.
set /p confirm="Continue? (y/N): "
if /i not "%confirm%"=="y" (
    echo Reset cancelled.
    exit /b 0
)

echo.
echo Starting project reset...
echo.

:: Check if Docker is running
docker info >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Docker is not running. Please start Docker first.
    pause
    exit /b 1
)

echo [INFO] Stopping all containers...
docker-compose down --remove-orphans 2>nul
docker-compose -f docker-compose.dev.yml down --remove-orphans 2>nul

:: Stop individual containers that might be running
echo [INFO] Stopping individual containers...
for %%c in (aiplusall-kb-app aiplusall-kb-docreader aiplusall-kb-frontend postgres redis minio neo4j ollama) do (
    docker ps -q -f name=%%c >nul 2>&1
    if not errorlevel 1 (
        echo Stopping container: %%c
        docker stop %%c >nul 2>&1
    )
)

echo [INFO] Removing Docker volumes...
:: Remove common volume patterns
for %%v in (aiplusall-kb_postgres-data aiplusall-kb_minio-data aiplusall-kb_redis-data aiplusall-kb_neo4j-data weknora_postgres-data weknora_minio_data weknora_redis_data weknora_neo4j_data) do (
    docker volume ls -q | findstr /c:"%%v" >nul 2>&1
    if not errorlevel 1 (
        echo Removing volume: %%v
        docker volume rm %%v >nul 2>&1
    )
)

echo [INFO] Cleaning local storage directories...
if exist "data" (
    echo Cleaning ./data directory...
    rmdir /s /q "data" 2>nul
    mkdir "data" 2>nul
)

if exist "tmp" (
    echo Cleaning ./tmp directory...
    rmdir /s /q "tmp" 2>nul
    mkdir "tmp" 2>nul
)

if exist "uploads" (
    echo Cleaning ./uploads directory...
    rmdir /s /q "uploads" 2>nul
    mkdir "uploads" 2>nul
)

if exist "files" (
    echo Cleaning ./files directory...
    rmdir /s /q "files" 2>nul
    mkdir "files" 2>nul
)

if exist "storage" (
    echo Cleaning ./storage directory...
    rmdir /s /q "storage" 2>nul
    mkdir "storage" 2>nul
)

echo [INFO] Cleaning temporary files...
if exist "aiplusall-kb.exe" del /q "aiplusall-kb.exe" 2>nul
if exist "aiplusall-kb" del /q "aiplusall-kb" 2>nul

if exist "frontend\node_modules" (
    echo Cleaning frontend node_modules...
    rmdir /s /q "frontend\node_modules" 2>nul
)

if exist "frontend\dist" (
    echo Cleaning frontend build artifacts...
    rmdir /s /q "frontend\dist" 2>nul
)

:: Clean temporary files
echo Cleaning temporary files...
for /r . %%f in (*.tmp *.log .DS_Store) do (
    if exist "%%f" del /q "%%f" 2>nul
)

echo [INFO] Pruning Docker system...
docker volume prune -f >nul 2>&1

echo.
echo ========================================
echo Reset Summary:
echo ========================================
echo   ✓ All containers stopped
echo   ✓ Docker volumes removed
echo   ✓ Local storage cleaned
echo   ✓ Temporary files removed
echo   ✓ Docker system pruned
echo.
echo ✅ Project has been reset to initial state!
echo.
echo To start fresh:
echo   1. docker-compose -f docker-compose.dev.yml up -d  (start infrastructure)
echo   2. Run database migrations
echo   3. Start backend application
echo   4. Start frontend application
echo.
echo Or use the quick development script if available.
echo.
pause