@echo off
REM QT-1 Mockware Startup Script for Windows
REM Starts both the Load Testing Client and Mock Provider

setlocal EnableDelayedExpansion

REM Configuration
set TESTING_CLIENT_PORT=3000
set MOCK_PROVIDER_PORT=8081

echo.
echo =============================================
echo ^|  🤖 Starting QT-1 Mockware Suite        ^|
echo =============================================
echo.

REM Check if Node.js is installed
node --version >nul 2>&1
if errorlevel 1 (
    echo ❌ ERROR: Node.js is not installed. Please install Node.js and try again.
    pause
    exit /b 1
)

echo ✅ Node.js is available

REM Check if directories exist
if not exist "testing-client" (
    echo ❌ ERROR: testing-client directory not found!
    pause
    exit /b 1
)

if not exist "mock-provider" (
    echo ❌ ERROR: mock-provider directory not found!
    pause
    exit /b 1
)

REM Create logs directory
if not exist "logs" mkdir logs

echo 📦 Installing dependencies...

REM Install testing client dependencies
echo   - Installing testing client dependencies...
cd testing-client
if not exist "node_modules" (
    call npm install >"..\logs\testing-client-install.log" 2>&1
    if errorlevel 1 (
        echo ❌ ERROR: Failed to install testing client dependencies
        echo Check logs\testing-client-install.log for details
        pause
        exit /b 1
    )
) else (
    echo   - Testing client dependencies already installed
)
cd ..

REM Install mock provider dependencies
echo   - Installing mock provider dependencies...
cd mock-provider
if not exist "node_modules" (
    call npm install >"..\logs\mock-provider-install.log" 2>&1
    if errorlevel 1 (
        echo ❌ ERROR: Failed to install mock provider dependencies
        echo Check logs\mock-provider-install.log for details
        pause
        exit /b 1
    )
) else (
    echo   - Mock provider dependencies already installed
)
cd ..

echo ✅ Dependencies ready

echo.
echo 🚀 Starting services...

REM Start Mock Provider
echo   - Starting Mock Provider...
cd mock-provider
start "QT-1 Mock Provider" cmd /k "npm start"
cd ..

REM Wait a moment
timeout /t 3 /nobreak >nul

REM Start Testing Client
echo   - Starting Load Testing Client...
cd testing-client
start "QT-1 Testing Client" cmd /k "npm start"
cd ..

REM Wait for services to start
echo   - Waiting for services to initialize...
timeout /t 5 /nobreak >nul

echo.
echo ✅ Mockware started successfully!
echo.
echo 📊 Services Running:
echo   • Load Testing Client : http://localhost:%TESTING_CLIENT_PORT%
echo   • Mock Provider API   : http://localhost:%MOCK_PROVIDER_PORT%
echo   • Provider Health     : http://localhost:%MOCK_PROVIDER_PORT%/health
echo.
echo 🌐 Quick Access:
echo   • Testing Dashboard   : http://localhost:%TESTING_CLIENT_PORT%
echo   • Provider Dashboard  : mock-provider\dashboard\index.html
echo.
echo 💡 Tips:
echo   • Both services are running in separate windows
echo   • Close the service windows to stop them
echo   • Check logs\ folder for service logs
echo.
echo 🎯 Ready for QT-1 middleware testing!
echo.
pause