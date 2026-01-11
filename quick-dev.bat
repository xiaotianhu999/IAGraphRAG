@echo off
setlocal enabledelayedexpansion

:: aiplusall-kb 快速开发环境启动脚本 (Windows版)
:: 此脚本会启动所有必需的基础设施服务

echo.
echo ========================================
echo   aiplusall-kb 快速开发环境启动
echo ========================================
echo.

:: 检查Docker是否运行
docker info >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Docker is not running. Please start Docker first.
    pause
    exit /b 1
)

:: 检查.env文件
if not exist ".env" (
    echo [ERROR] .env file not found. Please create it first.
    pause
    exit /b 1
)

echo [INFO] Step 1/4: Starting infrastructure services...
echo.

:: 启动基础设施服务
docker-compose -f docker-compose.dev.yml --profile full up -d

if errorlevel 1 (
    echo [ERROR] Failed to start infrastructure services
    pause
    exit /b 1
)

echo.
echo [SUCCESS] Infrastructure services started!
echo.
echo Service access addresses:
echo   - PostgreSQL:    localhost:5432
echo   - Redis:         localhost:6379
echo   - DocReader:     localhost:50051
echo   - MinIO:         localhost:9000 (Console: localhost:9001)
echo   - Qdrant:        localhost:6333 (gRPC: localhost:6334)
echo   - Neo4j:         localhost:7474 (Bolt: localhost:7687)
echo   - Jaeger:        localhost:16686
echo.

:: 等待服务启动
echo [INFO] Waiting for services to be ready...
timeout /t 10 /nobreak >nul

echo [INFO] Step 2/4: Check service status...
docker-compose -f docker-compose.dev.yml ps

echo.
echo [INFO] Step 3/4: Backend application
echo.
set /p start_backend="Start backend in a new window? (y/N): "
if /i "%start_backend%"=="y" (
    echo [INFO] Starting backend in new window...
    start "aiplusall-kb Backend" cmd /k "echo Starting backend... && go run cmd/server/main.go"
    echo [SUCCESS] Backend started in new window
) else (
    echo [WARNING] Backend startup skipped
    echo [INFO] To start backend later, run: go run cmd/server/main.go
)

echo.
echo [INFO] Step 4/4: Frontend application
echo.
set /p start_frontend="Start frontend in a new window? (y/N): "
if /i "%start_frontend%"=="y" (
    echo [INFO] Starting frontend in new window...
    start "aiplusall-kb Frontend" cmd /k "cd frontend && echo Installing dependencies... && npm install && echo Starting frontend... && npm run dev"
    echo [SUCCESS] Frontend started in new window
) else (
    echo [WARNING] Frontend startup skipped
    echo [INFO] To start frontend later, run: cd frontend && npm run dev
)

echo.
echo ========================================
echo   Startup Complete!
echo ========================================
echo.
echo Access URLs:
echo   - Frontend:      http://localhost:5173
echo   - Backend API:   http://localhost:8080
echo   - MinIO Console: http://localhost:9001
echo   - Jaeger UI:     http://localhost:16686
echo   - Neo4j Browser: http://localhost:7474
echo.
echo Management commands:
echo   - View status:   docker-compose -f docker-compose.dev.yml ps
echo   - View logs:     docker-compose -f docker-compose.dev.yml logs -f
echo   - Stop services: docker-compose -f docker-compose.dev.yml down
echo.
echo [SUCCESS] Development environment is ready!
echo.
pause