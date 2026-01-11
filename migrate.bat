@echo off
setlocal enabledelayedexpansion

:: aiplusall-kb Database Migration Script for Windows
:: This script handles database migrations using golang-migrate

:: Load environment variables from .env file
if exist ".env" (
    echo Loading .env file...
    for /f "usebackq tokens=1,2 delims==" %%a in (".env") do (
        if not "%%a"=="" if not "%%a:~0,1%"=="#" (
            set "%%a=%%b"
        )
    )
) else (
    echo Warning: .env file not found
)

:: Set default values if not provided
if "%DB_HOST%"=="" set "DB_HOST=localhost"
if "%DB_PORT%"=="" set "DB_PORT=5432"
if "%DB_USER%"=="" set "DB_USER=postgres"
if "%DB_PASSWORD%"=="" set "DB_PASSWORD=postgres"
if "%DB_NAME%"=="" set "DB_NAME=aiplusall_kb"

:: Use versioned migrations directory
set "MIGRATIONS_DIR=migrations/versioned"

:: Check if migrate tool is installed
migrate version >nul 2>&1
if errorlevel 1 (
    echo Error: migrate tool is not installed
    echo Install it with: go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@latest
    pause
    exit /b 1
)

:: Construct database URL
set "DB_URL=postgres://%DB_USER%:%DB_PASSWORD%@%DB_HOST%:%DB_PORT%/%DB_NAME%?sslmode=disable"

:: Parse command line arguments
set "COMMAND=%1"
set "ARG2=%2"

if "%COMMAND%"=="" (
    goto show_help
)

if "%COMMAND%"=="up" (
    goto migrate_up
)

if "%COMMAND%"=="down" (
    goto migrate_down
)

if "%COMMAND%"=="version" (
    goto migrate_version
)

if "%COMMAND%"=="force" (
    goto migrate_force
)

if "%COMMAND%"=="goto" (
    goto migrate_goto
)

if "%COMMAND%"=="create" (
    goto migrate_create
)

goto show_help

:migrate_up
echo Running migrations up...
echo DB_URL: %DB_URL%
echo DB_USER: %DB_USER%
echo DB_HOST: %DB_HOST%
echo DB_PORT: %DB_PORT%
echo DB_NAME: %DB_NAME%
echo MIGRATIONS_DIR: %MIGRATIONS_DIR%
migrate -path %MIGRATIONS_DIR% -database "%DB_URL%" up
if errorlevel 1 (
    echo Migration failed!
    pause
    exit /b 1
)
echo Migration completed successfully!
goto end

:migrate_down
echo Running migrations down...
migrate -path %MIGRATIONS_DIR% -database "%DB_URL%" down
if errorlevel 1 (
    echo Migration rollback failed!
    pause
    exit /b 1
)
echo Migration rollback completed successfully!
goto end

:migrate_version
echo Checking current migration version...
migrate -path %MIGRATIONS_DIR% -database "%DB_URL%" version
goto end

:migrate_force
if "%ARG2%"=="" (
    echo Error: Version number is required
    echo Usage: %0 force ^<version^>
    echo Note: Use -1 to reset to no version
    pause
    exit /b 1
)
echo Forcing migration version to %ARG2%...
migrate -path %MIGRATIONS_DIR% -database "%DB_URL%" force %ARG2%
if errorlevel 1 (
    echo Force migration failed!
    pause
    exit /b 1
)
echo Force migration completed successfully!
goto end

:migrate_goto
if "%ARG2%"=="" (
    echo Error: Version number is required
    echo Usage: %0 goto ^<version^>
    pause
    exit /b 1
)
echo Migrating to version %ARG2%...
migrate -path %MIGRATIONS_DIR% -database "%DB_URL%" goto %ARG2%
if errorlevel 1 (
    echo Migration to version %ARG2% failed!
    pause
    exit /b 1
)
echo Migration to version %ARG2% completed successfully!
goto end

:migrate_create
if "%ARG2%"=="" (
    echo Error: Migration name is required
    echo Usage: %0 create ^<migration_name^>
    pause
    exit /b 1
)
echo Creating migration files for %ARG2%...
migrate create -ext sql -dir %MIGRATIONS_DIR% -seq %ARG2%
if errorlevel 1 (
    echo Migration creation failed!
    pause
    exit /b 1
)
echo Migration files created successfully!
goto end

:show_help
echo aiplusall-kb Database Migration Script
echo.
echo Usage: %0 {up^|down^|create ^<migration_name^>^|version^|force ^<version^>^|goto ^<version^>}
echo.
echo Commands:
echo   up                    Apply all pending migrations
echo   down                  Rollback the last migration
echo   version               Show current migration version
echo   force ^<version^>      Force set migration version
echo   goto ^<version^>       Migrate to specific version
echo   create ^<name^>        Create new migration files
echo.
echo Examples:
echo   %0 up                 Apply all migrations
echo   %0 version            Check current version
echo   %0 force -1           Reset to no migrations (allows re-running all)
echo   %0 create add_users   Create new migration named "add_users"
echo.
pause
exit /b 1

:end
echo.
pause