@echo off
setlocal
title BeefTV Launcher
cd /d "%~dp0"

echo ==========================================
echo  BeefTV launcher (auto port)
echo ==========================================
echo.

where go >nul 2>nul
if errorlevel 1 (
  echo [ERROR] go not found in PATH. Install Go first: https://go.dev/dl/
  pause
  exit /b 1
)
where bun >nul 2>nul
if errorlevel 1 (
  echo [ERROR] bun not found in PATH. Install Bun first: https://bun.sh/
  pause
  exit /b 1
)

if not exist "web\node_modules" (
  echo [INFO] web\node_modules not found. Running "bun install" first...
  echo        This may take a few minutes on first run.
  echo.
  pushd "%~dp0web"
  call bun install
  if errorlevel 1 (
    echo.
    echo [ERROR] bun install failed. See messages above.
    popd
    pause
    exit /b 1
  )
  popd
  echo.
)

if not exist ".local\project-workbench-debug" mkdir ".local\project-workbench-debug"

rem ---- pick free ports: prefer 8080 / 3000, scan upward when busy ----
call :pick_port 8080 BACKEND_PORT
call :pick_port 3000 FRONTEND_PORT
if not defined BACKEND_PORT (
  echo [ERROR] no free backend port found near 8080.
  pause
  exit /b 1
)
if not defined FRONTEND_PORT (
  echo [ERROR] no free frontend port found near 3000.
  pause
  exit /b 1
)

echo [INFO] backend  port: %BACKEND_PORT%
echo [INFO] frontend port: %FRONTEND_PORT%
echo.

start "BeefTV Backend" cmd /k "cd /d "%~dp0backend" && set "CANVAS_BACKEND_DATA_DIR=../.local/project-workbench-debug" && set "CANVAS_BACKEND_ADDR=127.0.0.1:%BACKEND_PORT%" && go run ./cmd/server"
start "BeefTV Frontend" cmd /k "cd /d "%~dp0web" && set "VITE_API_PROXY_TARGET=http://127.0.0.1:%BACKEND_PORT%" && bun x vite --host 127.0.0.1 --port %FRONTEND_PORT% --strictPort"

echo Waiting for services to become ready (first run may compile for a while)...
set /a WAIT_TRIES=0
:wait_loop
set /a WAIT_TRIES+=1
netstat -ano | findstr "LISTENING" | findstr /c:":%BACKEND_PORT% " >nul
if errorlevel 1 goto wait_retry
netstat -ano | findstr "LISTENING" | findstr /c:":%FRONTEND_PORT% " >nul
if errorlevel 1 goto wait_retry
goto wait_done
:wait_retry
if %WAIT_TRIES% GEQ 180 goto wait_timeout
"%SystemRoot%\System32\ping.exe" -n 2 127.0.0.1 >nul
goto wait_loop

:wait_timeout
echo.
echo [WARN] Services did not become ready in 3 minutes.
echo        Check the "BeefTV Backend" / "BeefTV Frontend" windows for errors.
pause
exit /b 1

:wait_done
echo Services are ready.
start "" http://127.0.0.1:%FRONTEND_PORT%
echo.
echo Ready: http://127.0.0.1:%FRONTEND_PORT%   (backend: 127.0.0.1:%BACKEND_PORT%)
echo To stop: close the two command windows (BeefTV Backend / BeefTV Frontend).
"%SystemRoot%\System32\ping.exe" -n 7 127.0.0.1 >nul
exit /b 0

rem ---------------------------------------------------------------
rem %1 = start port, %2 = result variable name
:pick_port
setlocal
set /a "P=%~1"
:pick_loop
netstat -ano | findstr "LISTENING" | findstr /c:":%P% " >nul
if errorlevel 1 goto pick_done
set /a P+=1
if %P% LSS 8600 goto pick_loop
endlocal
exit /b 1
:pick_done
endlocal & set "%~2=%P%"
exit /b 0
