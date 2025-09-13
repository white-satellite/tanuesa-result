@echo off
setlocal

set "EXE=%~dp0..\gacha.exe"
set "PORT=%~1"
if "%PORT%"=="" set "PORT=3010"

if not exist "%EXE%" (
  echo gacha.exe not found: "%EXE%"
  pause
  exit /b 1
)

echo [INFO] Launching API in new console: http://127.0.0.1:%PORT%
start "gacha-api" "%ComSpec%" /k ""%EXE%" serve %PORT%"
exit /b 0

